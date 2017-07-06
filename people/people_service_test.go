// +build !jenkins

package people

import (
	"fmt"
	"os"
	"sort"
	"sync"
	"testing"

	"encoding/json"
	"github.com/Financial-Times/annotations-rw-neo4j/annotations"
	"github.com/Financial-Times/base-ft-rw-app-go/baseftrwapp"
	"github.com/Financial-Times/content-rw-neo4j/content"
	"github.com/Financial-Times/neo-utils-go/neoutils"
	"github.com/Financial-Times/up-rw-app-api-go/rwapi"
	"github.com/jmcvetta/neoism"
	"github.com/pborman/uuid"
	"github.com/stretchr/testify/assert"
)

const (
	minimalPersonUuid    = "180cec41-23fa-4148-806b-0602924e6858"
	fullPersonUuid       = "bbc4f575-edb3-4f51-92f0-5ce6c708d1ea"
	fullPersonSecondUuid = "026bc6ab-3581-476f-bdee-ad44934d8255"
	fullPersonThirdUuid  = "38431a92-dda3-4eb9-a367-60145a8e659f"
	uniquePersonUuid     = "bb596d64-78c5-4b00-a88f-e8248c956073"
	contentUUID          = "3fc9fe3e-af8c-4f7f-961a-e5065392bb31"
)

var minimalPerson = person{
	UUID:                   minimalPersonUuid,
	Name:                   "Minimal Person",
	PrefLabel:              "Pref Label",
	AlternativeIdentifiers: alternativeIdentifiers{FactsetIdentifier: fsIdentifier, UUIDS: []string{minimalPersonUuid}, TME: []string{}},
	Types: defaultTypes,
}

var fullPerson = person{
	UUID:                   fullPersonUuid,
	Name:                   "Full Person",
	PrefLabel:              "Pref Label",
	BirthYear:              1900,
	Salutation:             "Dr.",
	AlternativeIdentifiers: alternativeIdentifiers{FactsetIdentifier: fsIdentifier, UUIDS: []string{fullPersonUuid, fullPersonSecondUuid, fullPersonThirdUuid}, TME: []string{firstTmeIdentifier, secondTmeIdentifier}},
	Aliases:                []string{"Diff Name"},
	Types:                  defaultTypes,
	EmailAddress:           "email_address@example.com",
	TwitterHandle:          "@twitter_handle",
	FacebookProfile:        "facebook-profile",
	LinkedinProfile:        "linkedin-profile",
	Description:            "Plain text description",
	DescriptionXML:         "<p><strong>Richer</strong> description</p>",
	ImageURL:               "http://media.ft.com/validColumnistImage.png",
}

const (
	fsIdentifier        string = "012345-E"
	firstTmeIdentifier  string = "tmeIdentifier"
	secondTmeIdentifier string = "tmeIdentifier2"
)

var defaultTypes = []string{"Thing", "Concept", "Person"}

func TestCreateAllValuesPresent(t *testing.T) {
	assert := assert.New(t)

	db := getDatabaseConnectionAndCheckClean(t, assert)
	peopleDriver := getCypherDriver(db)

	defer cleanDB([]string{fullPersonUuid}, db, t, assert)

	assert.NoError(peopleDriver.Write(fullPerson), "Failed to write person")

	readPeopleAndCompare(fullPerson, t, db)
}

func TestCreateNotAllValuesPresent(t *testing.T) {
	assert := assert.New(t)
	db := getDatabaseConnectionAndCheckClean(t, assert)
	peopleDriver := getCypherDriver(db)

	defer cleanDB([]string{minimalPersonUuid}, db, t, assert)

	assert.NoError(peopleDriver.Write(minimalPerson), "Failed to write person")

	readPeopleAndCompare(minimalPerson, t, db)
}

func TestCreateHandlesSpecialCharacters(t *testing.T) {
	assert := assert.New(t)
	db := getDatabaseConnectionAndCheckClean(t, assert)
	peopleDriver := getCypherDriver(db)

	defer cleanDB([]string{uniquePersonUuid}, db, t, assert)

	personToWrite := person{UUID: uniquePersonUuid, Name: "Thomas M. O'Gara", BirthYear: 1974, Salutation: "Mr", AlternativeIdentifiers: alternativeIdentifiers{FactsetIdentifier: "FACTSET_ID", UUIDS: []string{uniquePersonUuid}, TME: []string{}}, Aliases: []string{"alias 1", "alias 2"}, Types: defaultTypes}

	assert.NoError(peopleDriver.Write(personToWrite), "Failed to write person")

	readPeopleAndCompare(personToWrite, t, db)
}

func TestUpdateWillRemovePropertiesNoLongerPresent(t *testing.T) {
	assert := assert.New(t)
	db := getDatabaseConnectionAndCheckClean(t, assert)
	peopleDriver := getCypherDriver(db)

	defer cleanDB([]string{fullPersonUuid}, db, t, assert)

	assert.NoError(peopleDriver.Write(fullPerson), "Failed to write person")
	storedFullPerson, _, err := peopleDriver.Read(fullPersonUuid)

	assert.NoError(err)
	assert.NotEmpty(storedFullPerson)

	var minimalPerson = person{
		UUID: fullPersonUuid,
		Name: "Minimal Person",
		AlternativeIdentifiers: alternativeIdentifiers{FactsetIdentifier: fsIdentifier, UUIDS: []string{fullPersonUuid}, TME: []string{}},
		Types: defaultTypes,
	}

	assert.NoError(peopleDriver.Write(minimalPerson), "Failed to write updated person")

	readPeopleAndCompare(minimalPerson, t, db)
}

func TestAliasesAreWrittenAndAreAbleToBeReadInOrder(t *testing.T) {
	assert := assert.New(t)
	db := getDatabaseConnectionAndCheckClean(t, assert)
	peopleDriver := getCypherDriver(db)

	defer cleanDB([]string{uniquePersonUuid}, db, t, assert)
	personToWrite := person{UUID: uniquePersonUuid, Name: "Test", BirthYear: 1974, Salutation: "Mr", AlternativeIdentifiers: alternativeIdentifiers{FactsetIdentifier: "FACTSET_ID", UUIDS: []string{uniquePersonUuid}}, Aliases: []string{"alias 1", "alias 2"}}

	peopleDriver.Write(personToWrite)

	result := []struct {
		Aliases []string `json:"t.aliases"`
	}{}

	getPrefLabelQuery := &neoism.CypherQuery{
		Statement: `
				MATCH (t:Person {uuid:{uuid}}) RETURN t.aliases
				`,
		Parameters: map[string]interface{}{
			"uuid": uniquePersonUuid,
		},
		Result: &result,
	}

	err := peopleDriver.conn.CypherBatch([]*neoism.CypherQuery{getPrefLabelQuery})
	assert.NoError(err)
	assert.Equal("alias 1", result[0].Aliases[0], "PrefLabel should be 'alias 1")
}

func TestAddingPersonWithExistingIdentifiersShouldFail(t *testing.T) {
	assert := assert.New(t)

	db := getDatabaseConnectionAndCheckClean(t, assert)
	cypherDriver := getCypherDriver(db)

	defer cleanDB([]string{minimalPersonUuid, fullPersonUuid}, db, t, assert)

	assert.NoError(cypherDriver.Write(fullPerson))
	err := cypherDriver.Write(minimalPerson)
	assert.Error(err)
	assert.IsType(rwapi.ConstraintOrTransactionError{}, err)
}

func TestPrefLabelIsEqualToPrefLabelAndAbleToBeRead(t *testing.T) {
	assert := assert.New(t)
	db := getDatabaseConnectionAndCheckClean(t, assert)
	peopleDriver := getCypherDriver(db)

	defer cleanDB([]string{fullPersonUuid}, db, t, assert)

	storedPerson := peopleDriver.Write(fullPerson)

	fmt.Printf("%v", storedPerson)

	result := []struct {
		PrefLabel string `json:"t.prefLabel"`
	}{}

	getPrefLabelQuery := &neoism.CypherQuery{
		Statement: `
				MATCH (t:Person {uuid:{uuid}}) RETURN t.prefLabel
				`,
		Parameters: map[string]interface{}{
			"uuid": fullPersonUuid,
		},
		Result: &result,
	}

	err := peopleDriver.conn.CypherBatch([]*neoism.CypherQuery{getPrefLabelQuery})
	assert.NoError(err)
	assert.Equal(fullPerson.PrefLabel, result[0].PrefLabel, "PrefLabel should be 'Pref Label")
}

func TestDeleteWillDeleteEntireNodeIfNoRelationship(t *testing.T) {
	assert := assert.New(t)
	db := getDatabaseConnectionAndCheckClean(t, assert)
	peopleDriver := getCypherDriver(db)

	defer cleanDB([]string{minimalPersonUuid}, db, t, assert)

	assert.NoError(peopleDriver.Write(minimalPerson), "Failed to write person")

	found, err := peopleDriver.Delete(minimalPersonUuid)
	assert.True(found, "Didn't manage to delete person for uuid %", minimalPersonUuid)
	assert.NoError(err, "Error deleting person for uuid %s", minimalPersonUuid)

	p, found, err := peopleDriver.Read(minimalPersonUuid)

	assert.Equal(person{}, p, "Found person %s who should have been deleted", p)
	assert.False(found, "Found person for uuid %s who should have been deleted", minimalPersonUuid)
	assert.NoError(err, "Error trying to find person for uuid %s", minimalPersonUuid)
	assert.Equal(false, doesThingExistAtAll(minimalPersonUuid, db, t, assert), "Found thing who should have been deleted uuid: %s", minimalPersonUuid)
}

func TestDeleteWithRelationshipsMaintainsRelationships(t *testing.T) {
	assert := assert.New(t)
	db := getDatabaseConnectionAndCheckClean(t, assert)
	peopleDriver := getCypherDriver(db)

	defer cleanDB([]string{fullPersonUuid, contentUUID}, db, t, assert)

	assert.NoError(peopleDriver.Write(fullPerson), "Failed to write person")
	writeContent(assert, db)
	writeAnnotation(assert, db)

	found, err := peopleDriver.Delete(fullPersonUuid)

	assert.True(found, "Didn't manage to delete person for uuid %", fullPersonUuid)
	assert.NoError(err, "Error deleting person for uuid %s", fullPersonUuid)

	p, found, err := peopleDriver.Read(fullPersonUuid)

	assert.Equal(person{}, p, "Found person %s who should have been deleted", p)
	assert.False(found, "Found person for uuid %s who should have been deleted", fullPersonUuid)
	assert.NoError(err, "Error trying to find person for uuid %s", fullPersonUuid)
	assert.Equal(true, doesThingExistWithIdentifiers(fullPersonUuid, db, t, assert), "Unable to find a Thing with any Identifiers, uuid: %s", fullPersonUuid)
}

func TestIDs(t *testing.T) {

	assert := assert.New(t)
	db := getDatabaseConnection(assert)
	peopleDriver := getCypherDriver(db)

	uuids := make(map[string]struct{})
	toDelete := []string{}

	defer func() {
		var wg sync.WaitGroup
		for _, u := range toDelete {
			wg.Add(1)
			go func(u string) {
				defer wg.Done()
				peopleDriver.Delete(u)
			}(u)
		}
		wg.Wait()
	}()

	var wg sync.WaitGroup

	for i := 0; i < 3000; i++ {
		u := uuid.New()
		uuids[u] = struct{}{}
		toDelete = append(toDelete, u)
		wg.Add(1)
		go func(i int, u string) {
			defer wg.Done()

			err := peopleDriver.Write(
				person{
					UUID:       u,
					Name:       fmt.Sprintf("Test %d", i),
					BirthYear:  1066,
					Salutation: "Dr",
					AlternativeIdentifiers: alternativeIdentifiers{
						FactsetIdentifier: fmt.Sprintf("FACTSET_ID_%d", i),
						UUIDS:             []string{u},
					},
					Aliases: []string{fmt.Sprintf("alias for %d", i)},
				},
			)
			assert.NoError(err)
		}(i, u)
	}

	wg.Wait()

	assert.NoError(peopleDriver.IDs(func(id rwapi.IDEntry) (bool, error) {
		_, found := uuids[id.ID]
		if !found {
			t.Errorf("unexpected uuid %s", id.ID)
		} else {
			delete(uuids, id.ID)
		}
		return true, nil
	}))

	for u, _ := range uuids {
		t.Errorf("missing uuid %s", u)
	}

}

func writeAnnotation(assert *assert.Assertions, db neoutils.NeoConnection) annotations.Service {
	annotationsRW := annotations.NewCypherAnnotationsService(db, "v2", "annotations-v2")
	assert.NoError(annotationsRW.Initialise())
	writeJSONToAnnotationsService(annotationsRW, contentUUID, "./fixtures/Annotations-3fc9fe3e-af8c-4f7f-961a-e5065392bb31-v2.json", assert)
	return annotationsRW
}

func writeContent(assert *assert.Assertions, db neoutils.NeoConnection) baseftrwapp.Service {
	contentRW := content.NewCypherContentService(db)
	assert.NoError(contentRW.Initialise())
	writeJSONToService(contentRW, "./fixtures/Content-3fc9fe3e-af8c-4f7f-961a-e5065392bb31.json", assert)
	return contentRW
}

func writeJSONToAnnotationsService(service annotations.Service, contentUUID string, pathToJSONFile string, assert *assert.Assertions) {
	f, err := os.Open(pathToJSONFile)
	assert.NoError(err)
	dec := json.NewDecoder(f)
	inst, errr := service.DecodeJSON(dec)
	assert.NoError(errr, "Error parsing file %s", pathToJSONFile)
	errrr := service.Write(contentUUID, inst)
	assert.NoError(errrr)
}

func writeJSONToService(service baseftrwapp.Service, pathToJSONFile string, assert *assert.Assertions) {
	f, err := os.Open(pathToJSONFile)
	assert.NoError(err)
	dec := json.NewDecoder(f)
	inst, _, errr := service.DecodeJSON(dec)
	assert.NoError(errr)
	errrr := service.Write(inst)
	assert.NoError(errrr)
}

func doesThingExistAtAll(uuid string, db neoutils.NeoConnection, t *testing.T, assert *assert.Assertions) bool {
	result := []struct {
		Uuid string `json:"thing.uuid"`
	}{}

	checkGraph := neoism.CypherQuery{
		Statement: `
			MATCH (a:Thing {uuid: "%s"}) return a.uuid
		`,
		Parameters: neoism.Props{
			"uuid": uuid,
		},
		Result: &result,
	}

	err := db.CypherBatch([]*neoism.CypherQuery{&checkGraph})
	assert.NoError(err)

	if len(result) == 0 {
		return false
	}

	return true
}

func doesThingExistWithIdentifiers(uuid string, db neoutils.NeoConnection, t *testing.T, assert *assert.Assertions) bool {

	result := []struct {
		uuid string `json:"thing.uuid"`
	}{}

	checkGraph := neoism.CypherQuery{
		Statement: `
			MATCH (a:Thing {uuid: "%s"})-[:IDENTIFIES]-(:Identifier)
			WITH collect(distinct a.uuid) as uuid
			RETURN uuid
		`,
		Parameters: neoism.Props{
			"uuid": uuid,
		},
		Result: &result,
	}

	err := db.CypherBatch([]*neoism.CypherQuery{&checkGraph})
	assert.NoError(err)

	if len(result) == 0 {
		return false
	}

	return true
}

func readPeopleAndCompare(expected person, t *testing.T, db neoutils.NeoConnection) {
	sort.Strings(expected.Types)
	sort.Strings(expected.AlternativeIdentifiers.TME)
	sort.Strings(expected.AlternativeIdentifiers.UUIDS)

	actual, found, err := getCypherDriver(db).Read(expected.UUID)
	assert.NoError(t, err)
	assert.True(t, found)

	actualPeople := actual.(person)
	sort.Strings(actualPeople.Types)
	sort.Strings(actualPeople.AlternativeIdentifiers.TME)
	sort.Strings(actualPeople.AlternativeIdentifiers.UUIDS)

	assert.EqualValues(t, expected, actualPeople)
}

func getDatabaseConnectionAndCheckClean(t *testing.T, assert *assert.Assertions) neoutils.NeoConnection {
	db := getDatabaseConnection(assert)
	checkDbClean([]string{fullPersonUuid, contentUUID, minimalPersonUuid, fullPersonSecondUuid, fullPersonThirdUuid, uniquePersonUuid}, db, t)
	return db
}

func getDatabaseConnection(assert *assert.Assertions) neoutils.NeoConnection {
	url := os.Getenv("NEO4J_TEST_URL")
	if url == "" {
		url = "http://localhost:7474/db/data"
	}

	conf := neoutils.DefaultConnectionConfig()
	conf.Transactional = false
	db, err := neoutils.Connect(url, conf)
	assert.NoError(err, "Failed to connect to Neo4j")
	return db
}

func cleanDB(uuidsToClean []string, db neoutils.NeoConnection, t *testing.T, assert *assert.Assertions) {
	qs := make([]*neoism.CypherQuery, len(uuidsToClean))
	for i, uuid := range uuidsToClean {
		qs[i] = &neoism.CypherQuery{
			Statement: fmt.Sprintf(`
			MATCH (a:Thing {uuid: "%s"})
			OPTIONAL MATCH (a)-[rel]-(i)
			DELETE rel, i
			DETACH DELETE a`, uuid)}
	}
	err := db.CypherBatch(qs)
	assert.NoError(err)
}

func checkDbClean(uuidsCleaned []string, db neoutils.NeoConnection, t *testing.T) {
	assert := assert.New(t)

	result := []struct {
		Uuid string `json:"thing.uuid"`
	}{}

	checkGraph := neoism.CypherQuery{
		Statement: `
			MATCH (thing) WHERE thing.uuid in {uuids} RETURN thing.uuid
		`,
		Parameters: neoism.Props{
			"uuids": uuidsCleaned,
		},
		Result: &result,
	}
	err := db.CypherBatch([]*neoism.CypherQuery{&checkGraph})
	assert.NoError(err)
	assert.Empty(result)
}

func getCypherDriver(db neoutils.NeoConnection) service {
	cr := NewCypherPeopleService(db)
	cr.Initialise()
	return cr
}
