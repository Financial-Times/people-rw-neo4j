package main

import (
	"errors"
	"fmt"
	"github.com/Financial-Times/go-fthealth/v1a"
	"github.com/jmcvetta/neoism"
)

func setUpHealthCheck(db *neoism.Database) v1a.Check {

	checker := func() (string, error) {
		var result []struct {
			UUID string `json:"uuid"`
		}

		query := &neoism.CypherQuery{
			Statement: `MATCH (n:Person) 
					return  n.uuid as uuid
					limit 1`,
			Result: &result,
		}

		err := db.Cypher(query)

		if err != nil {
			return "", err
		}
		if len(result) == 0 {
			return "", errors.New("No Person found")
		}
		if result[0].UUID == "" {
			return "", errors.New("UUID not set")
		}
		return fmt.Sprintf("Found a person with a valid uuid = %v", result[0].UUID), nil
	}

	return v1a.Check{
		BusinessImpact:   "Cannot read/write people via this writer",
		Name:             "Check connectivity to Neo4j - neoUrl is a parameter in hieradata for this service",
		PanicGuide:       "TODO - write panic guide",
		Severity:         1,
		TechnicalSummary: "Cannot connect to a Neo4j instance with at least one person loaded in it",
		Checker:          checker,
	}
}
