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
	peopleDriver = getPeopleCypherDriver(t)

	assert.NoError(peopleDriver.Write(fullPerson), "Failed to write person")

	readPersonForUUIDAndCheckFieldsMatch(t, fullPersonUuid, fullPerson)

	cleanUp(t, fullPersonUuid)
}

func TestCreateNotAllValuesPresent(t *testing.T) {
	assert := assert.New(t)
	peopleDriver = getPeopleCypherDriver(t)

	assert.NoError(peopleDriver.Write(minimalPerson), "Failed to write person")

	readPersonForUUIDAndCheckFieldsMatch(t, minimalPersonUuid, minimalPerson)

	cleanUp(t, minimalPersonUuid)
}

func TestCreateHandlesSpecialCharacters(t *testing.T) {
	assert := assert.New(t)
	uuid := "12345"
	peopleDriver = getPeopleCypherDriver(t)

	personToWrite := person{UUID: uuid, Name: "Thomas M. O'Gara", BirthYear: 1974, Salutation: "Mr",
		Identifiers: []identifier{identifier{fsAuthority, "FACTSET_ID"}}, Aliases: []string{"alias 1", "alias 2"}}

	assert.NoError(peopleDriver.Write(personToWrite), "Failed to write person")

	readPersonForUUIDAndCheckFieldsMatch(t, uuid, personToWrite)

	cleanUp(t, uuid)
}

func TestUpdateWillRemovePropertiesNoLongerPresent(t *testing.T) {
	assert := assert.New(t)
	peopleDriver = getPeopleCypherDriver(t)

	assert.NoError(peopleDriver.Write(fullPerson), "Failed to write person")
	readPersonForUUIDAndCheckFieldsMatch(t, fullPersonUuid, fullPerson)

	assert.NoError(peopleDriver.Write(minimalPerson), "Failed to write updated person")
	readPersonForUUIDAndCheckFieldsMatch(t, minimalPersonUuid, minimalPerson)

	cleanUp(t, minimalPersonUuid)
}

func TestWritePersonWithUnsupportedAuthority(t *testing.T) {
	assert := assert.New(t)
	uuid := "12345"
	peopleDriver = getPeopleCypherDriver(t)

	personToWrite := person{UUID: uuid, Name: "Test", BirthYear: 1974, Salutation: "Mr",
		Identifiers: []identifier{invalidIdentifier}, Aliases: []string{"alias 1", "alias 2"}}

	assert.Error(peopleDriver.Write(personToWrite))
}

func TestAliasesAreWrittenAndAreAbleToBeReadInOrder(t *testing.T) {
	assert := assert.New(t)
	peopleDriver := getPeopleCypherDriver(t)
	uuid := "12345"
	personToWrite := person{UUID: uuid, Name: "Test", BirthYear: 1974, Salutation: "Mr",
		Identifiers: []identifier{identifier{fsAuthority, "FACTSET_ID"}}, Aliases: []string{"alias 1", "alias 2"}}

	peopleDriver.Write(personToWrite)

	result := []struct {
		Aliases []string `json:"t.aliases"`
	}{}

	getPrefLabelQuery := &neoism.CypherQuery{
		Statement: `
				MATCH (t:Person {uuid:"12345"}) RETURN t.aliases
				`,
		Result: &result,
	}

	err := peopleDriver.cypherRunner.CypherBatch([]*neoism.CypherQuery{getPrefLabelQuery})
	assert.NoError(err)
	assert.Equal("alias 1", result[0].Aliases[0], "PrefLabel should be 'alias 1")
	cleanUp(t, uuid)
}

func TestPrefLabelIsEqualToNameAndAbleToBeRead(t *testing.T) {
	assert := assert.New(t)
	peopleDriver := getPeopleCypherDriver(t)

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
	cleanUp(t, fullPersonUuid)
}

func TestDelete(t *testing.T) {
	assert := assert.New(t)
	peopleDriver = getPeopleCypherDriver(t)

	assert.NoError(peopleDriver.Write(minimalPerson), "Failed to write person")

	found, err := peopleDriver.Delete(minimalPersonUuid)
	assert.True(found, "Didn't manage to delete person for uuid %", minimalPersonUuid)
	assert.NoError(err, "Error deleting person for uuid %s", minimalPersonUuid)

	p, found, err := peopleDriver.Read(minimalPersonUuid)

	assert.Equal(person{}, p, "Found person %s who should have been deleted", p)
	assert.False(found, "Found person for uuid %s who should have been deleted", minimalPersonUuid)
	assert.NoError(err, "Error trying to find person for uuid %s", minimalPersonUuid)
}

func TestConnectivityCheck(t *testing.T) {
	assert := assert.New(t)
	peopleDriver = getPeopleCypherDriver(t)
	err := peopleDriver.Check()
	assert.NoError(err, "Unexpected error on connectivity check")
}

func getPeopleCypherDriver(t *testing.T) service {
	assert := assert.New(t)
	url := os.Getenv("NEO4J_TEST_URL")
	if url == "" {
		url = "http://localhost:7474/db/data"
	}

	db, err := neoism.Connect(url)
	assert.NoError(err, "Failed to connect to Neo4j")
	return NewCypherPeopleService(neoutils.StringerDb{db}, db)
}

func readPersonForUUIDAndCheckFieldsMatch(t *testing.T, uuid string, expectedPerson person) {
	assert := assert.New(t)
	storedPerson, found, err := peopleDriver.Read(uuid)

	assert.NoError(err, "Error finding person for uuid %s", uuid)
	assert.True(found, "Didn't find person for uuid %s", uuid)
	assert.Equal(expectedPerson, storedPerson, "people should be the same")
}

func cleanUp(t *testing.T, uuid string) {
	assert := assert.New(t)
	found, err := peopleDriver.Delete(uuid)
	assert.True(found, "Didn't manage to delete person for uuid %", uuid)
	assert.NoError(err, "Error deleting person for uuid %s", uuid)
}
