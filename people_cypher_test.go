// +build !jenkins

package main

import (
	"github.com/jmcvetta/neoism"
	"github.com/stretchr/testify/assert"
	"os"
	"testing"
)

func TestCreate(t *testing.T) {
	assert := assert.New(t)
	person := person{UUID: "123", Name: "Test", Identifiers: []identifier{identifier{fsAuthority, "FACTSET_ID"}}}

	url := os.Getenv("NEO4J_TEST_URL")
	if url == "" {
		url := "http://localhost:7474/db/data"
	}

	db, err := neoism.Connect(url)
	assert.NoError(err, "Failed to connect to Neo4j")
	peopleDriver = NewPeopleCypherDriver(db)

	assert.NoError(peopleDriver.Write(person), "Failed to write person")

	storedPerson, found, err := peopleDriver.Read("123")

	assert.NoError(err, "Error finding person")
	assert.True(found, "Didn't find person")
	assert.Equal(person, storedPerson, "people should be the same")
}
