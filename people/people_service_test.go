// +build !jenkins

package people

import (
	"fmt"
	"os"
	"testing"

	"github.com/Financial-Times/base-ft-rw-app-go/baseftrwapp"
	"github.com/Financial-Times/neo-utils-go/neoutils"
	"github.com/jmcvetta/neoism"
	"github.com/stretchr/testify/assert"
)

const (
	minimalPersonUuid = "180cec41-23fa-4148-806b-0602924e6858"
	fullPersonUuid    = "bbc4f575-edb3-4f51-92f0-5ce6c708d1ea"
	uniquePersonUuid  = "bb596d64-78c5-4b00-a88f-e8248c956073"
)

var minimalPerson = person{
	UUID:        minimalPersonUuid,
	Name:        "Minimal Person",
	Identifiers: []identifier{fsIdentifier},
}

var fullPerson = person{
	UUID:        fullPersonUuid,
	Name:        "Full Person",
	BirthYear:   1900,
	Salutation:  "Dr.",
	Identifiers: []identifier{fsIdentifier, firstTmeIdentifier, secondTmeIdentifier},
	Aliases:     []string{"Diff Name"},
}

var fsIdentifier = identifier{
	Authority:       fsAuthority,
	IdentifierValue: "012345-E",
}

var firstTmeIdentifier = identifier{
	Authority:       tmeAuthority,
	IdentifierValue: "tmeIdentifier",
}

var secondTmeIdentifier = identifier{
	Authority:       tmeAuthority,
	IdentifierValue: "tmeIdentifier2",
}

var invalidIdentifier = identifier{
	Authority:       "Invalid Authority",
	IdentifierValue: "tmeIdentifier2",
}

var peopleDriver baseftrwapp.Service

func TestCreateAllValuesPresent(t *testing.T) {
	assert := assert.New(t)

	db := getDatabaseConnectionAndCheckClean(t, assert)
	peopleDriver := getCypherDriver(db)
	defer cleanDB(db, t, assert)

	assert.NoError(peopleDriver.Write(fullPerson), "Failed to write person")

	storedPerson, _, err := peopleDriver.Read(fullPersonUuid)

	assert.NoError(err)
	assert.NotEmpty(storedPerson)
}

func TestCreateNotAllValuesPresent(t *testing.T) {
	assert := assert.New(t)
	db := getDatabaseConnectionAndCheckClean(t, assert)
	peopleDriver := getCypherDriver(db)
	defer cleanDB(db, t, assert)

	assert.NoError(peopleDriver.Write(minimalPerson), "Failed to write person")

	storedPerson, _, err := peopleDriver.Read(minimalPersonUuid)

	assert.NoError(err)
	assert.NotEmpty(storedPerson)
}

func TestCreateHandlesSpecialCharacters(t *testing.T) {
	assert := assert.New(t)
	db := getDatabaseConnectionAndCheckClean(t, assert)
	peopleDriver := getCypherDriver(db)
	defer cleanDB(db, t, assert)

	personToWrite := person{UUID: uniquePersonUuid, Name: "Thomas M. O'Gara", BirthYear: 1974, Salutation: "Mr",
		Identifiers: []identifier{identifier{fsAuthority, "FACTSET_ID"}}, Aliases: []string{"alias 1", "alias 2"}}

	assert.NoError(peopleDriver.Write(personToWrite), "Failed to write person")

	storedPerson, _, err := peopleDriver.Read(uniquePersonUuid)

	assert.NoError(err)
	assert.NotEmpty(storedPerson)
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
		UUID:        fullPersonUuid,
		Name:        "Minimal Person",
		Identifiers: []identifier{fsIdentifier},
	}

	assert.NoError(peopleDriver.Write(minimalPerson), "Failed to write updated person")
	storedMinimalPerson, _, err := peopleDriver.Read(fullPersonUuid)

	assert.NoError(err)
	assert.NotEmpty(storedMinimalPerson)
}

func TestWritePersonWithUnsupportedAuthority(t *testing.T) {
	assert := assert.New(t)
	db := getDatabaseConnectionAndCheckClean(t, assert)
	peopleDriver := getCypherDriver(db)
	defer cleanDB(db, t, assert)

	personToWrite := person{UUID: uniquePersonUuid, Name: "Test", BirthYear: 1974, Salutation: "Mr",
		Identifiers: []identifier{invalidIdentifier}, Aliases: []string{"alias 1", "alias 2"}}

	assert.Error(peopleDriver.Write(personToWrite))
}

func TestAliasesAreWrittenAndAreAbleToBeReadInOrder(t *testing.T) {
	assert := assert.New(t)
	db := getDatabaseConnectionAndCheckClean(t, assert)
	peopleDriver := getCypherDriver(db)
	defer cleanDB(db, t, assert)
	personToWrite := person{UUID: uniquePersonUuid, Name: "Test", BirthYear: 1974, Salutation: "Mr",
		Identifiers: []identifier{identifier{fsAuthority, "FACTSET_ID"}}, Aliases: []string{"alias 1", "alias 2"}}

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

	err := peopleDriver.cypherRunner.CypherBatch([]*neoism.CypherQuery{getPrefLabelQuery})
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

func TestPrefLabelIsEqualToNameAndAbleToBeRead(t *testing.T) {
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

	err := peopleDriver.cypherRunner.CypherBatch([]*neoism.CypherQuery{getPrefLabelQuery})
	assert.NoError(err)
	assert.Equal(fullPerson.Name, result[0].PrefLabel, "PrefLabel should be 'Full Person")
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

func getDatabaseConnectionAndCheckClean(t *testing.T, assert *assert.Assertions) *neoism.Database {
	db := getDatabaseConnection(assert)
	cleanDB(db, t, assert)
	checkDbClean(db, t)
	return db
}

func getDatabaseConnection(assert *assert.Assertions) *neoism.Database {
	url := os.Getenv("NEO4J_TEST_URL")
	if url == "" {
		url = "http://localhost:7474/db/data"
	}

	db, err := neoism.Connect(url)
	assert.NoError(err, "Failed to connect to Neo4j")
	return db
}

func cleanDB(db *neoism.Database, t *testing.T, assert *assert.Assertions) {
	qs := []*neoism.CypherQuery{
		{
			Statement: fmt.Sprintf("MATCH (mp:Thing {uuid: '%v'})<-[:IDENTIFIES*0..]-(i:Identifier) DETACH DELETE mp, i", minimalPersonUuid),
		},
		{
			Statement: fmt.Sprintf("MATCH (fp:Thing {uuid: '%v'})<-[:IDENTIFIES*0..]-(i:Identifier) DETACH DELETE fp, i", fullPersonUuid),
		},
	}

	err := db.CypherBatch(qs)
	assert.NoError(err)
}

func checkDbClean(db *neoism.Database, t *testing.T) {
	assert := assert.New(t)

	result := []struct {
		Uuid string `json:"org.uuid"`
	}{}

	checkGraph := neoism.CypherQuery{
		Statement: `
			MATCH (org:Thing) WHERE org.uuid in {uuids} RETURN org.uuid
		`,
		Parameters: neoism.Props{
			"uuids": []string{fullPersonUuid, minimalPersonUuid},
		},
		Result: &result,
	}
	err := db.Cypher(&checkGraph)
	assert.NoError(err)
	assert.Empty(result)
}

func getCypherDriver(db *neoism.Database) service {
	cr := NewCypherPeopleService(neoutils.NewBatchCypherRunner(neoutils.StringerDb{db}, 3), db)
	cr.Initialise()
	return cr
}
