// +build !jenkins

package people

import (
	"fmt"
	"os"
	"sort"
	"sync"
	"testing"

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
	defer cleanDB(db, t, assert)

	assert.NoError(peopleDriver.Write(fullPerson), "Failed to write person")

	readPeopleAndCompare(fullPerson, t, db)
}

func TestCreateNotAllValuesPresent(t *testing.T) {
	assert := assert.New(t)
	db := getDatabaseConnectionAndCheckClean(t, assert)
	peopleDriver := getCypherDriver(db)
	defer cleanDB(db, t, assert)

	assert.NoError(peopleDriver.Write(minimalPerson), "Failed to write person")

	readPeopleAndCompare(minimalPerson, t, db)
}

func TestCreateHandlesSpecialCharacters(t *testing.T) {
	assert := assert.New(t)
	db := getDatabaseConnectionAndCheckClean(t, assert)
	peopleDriver := getCypherDriver(db)
	defer cleanDB(db, t, assert)

	personToWrite := person{UUID: uniquePersonUuid, Name: "Thomas M. O'Gara", BirthYear: 1974, Salutation: "Mr", AlternativeIdentifiers: alternativeIdentifiers{FactsetIdentifier: "FACTSET_ID", UUIDS: []string{uniquePersonUuid}, TME: []string{}}, Aliases: []string{"alias 1", "alias 2"}, Types: defaultTypes}

	assert.NoError(peopleDriver.Write(personToWrite), "Failed to write person")

	readPeopleAndCompare(personToWrite, t, db)
}

func TestUpdateWillRemovePropertiesNoLongerPresent(t *testing.T) {
	assert := assert.New(t)
	db := getDatabaseConnectionAndCheckClean(t, assert)
	peopleDriver := getCypherDriver(db)
	defer cleanDB(db, t, assert)

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
	defer cleanDB(db, t, assert)
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
	defer cleanDB(db, t, assert)
	assert.NoError(cypherDriver.Write(fullPerson))
	err := cypherDriver.Write(minimalPerson)
	assert.Error(err)
	assert.IsType(&neoutils.ConstraintViolationError{}, err)
}

func TestPrefLabelIsEqualToPrefLabelAndAbleToBeRead(t *testing.T) {
	assert := assert.New(t)
	db := getDatabaseConnectionAndCheckClean(t, assert)
	peopleDriver := getCypherDriver(db)
	defer cleanDB(db, t, assert)

	storedPerson := peopleDriver.Write(fullPerson)

	fmt.Printf("", storedPerson)

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

func TestDelete(t *testing.T) {
	assert := assert.New(t)
	db := getDatabaseConnectionAndCheckClean(t, assert)
	peopleDriver := getCypherDriver(db)
	defer cleanDB(db, t, assert)

	assert.NoError(peopleDriver.Write(minimalPerson), "Failed to write person")

	found, err := peopleDriver.Delete(minimalPersonUuid)
	assert.True(found, "Didn't manage to delete person for uuid %", minimalPersonUuid)
	assert.NoError(err, "Error deleting person for uuid %s", minimalPersonUuid)

	p, found, err := peopleDriver.Read(minimalPersonUuid)

	assert.Equal(person{}, p, "Found person %s who should have been deleted", p)
	assert.False(found, "Found person for uuid %s who should have been deleted", minimalPersonUuid)
	assert.NoError(err, "Error trying to find person for uuid %s", minimalPersonUuid)
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
	cleanDB(db, t, assert)
	checkDbClean(db, t)
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

func cleanDB(db neoutils.NeoConnection, t *testing.T, assert *assert.Assertions) {
	qs := []*neoism.CypherQuery{
		{
			Statement: fmt.Sprintf("MATCH (mp:Thing {uuid: '%v'})<-[:IDENTIFIES*0..]-(i:Identifier) DETACH DELETE mp, i", minimalPersonUuid),
		},
		{
			Statement: fmt.Sprintf("MATCH (fp:Thing {uuid: '%v'})<-[:IDENTIFIES*0..]-(i:Identifier) DETACH DELETE fp, i", fullPersonUuid),
		},
		{
			Statement: fmt.Sprintf("MATCH (fp:Thing {uuid: '%v'})<-[:IDENTIFIES*0..]-(i:Identifier) DETACH DELETE fp, i", uniquePersonUuid),
		},
	}

	err := db.CypherBatch(qs)
	assert.NoError(err)
}

func checkDbClean(db neoutils.NeoConnection, t *testing.T) {
	assert := assert.New(t)

	result := []struct {
		Uuid string `json:"org.uuid"`
	}{}

	checkGraph := neoism.CypherQuery{
		Statement: `
			MATCH (org:Thing) WHERE org.uuid in {uuids} RETURN org.uuid
		`,
		Parameters: neoism.Props{
			"uuids": []string{fullPersonUuid, minimalPersonUuid, uniquePersonUuid},
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
