package main

import (
	"errors"
	"fmt"
	"github.com/Financial-Times/go-fthealth/v1a"
	"github.com/Financial-Times/neo-cypher-runner-go"
	"github.com/jmcvetta/neoism"
)

type hcUUIDResult struct {
	UUID string `json:"uuid"`
}

func setUpHealthCheck(cr neocypherrunner.CypherRunner) v1a.Check {

	checker := func() (string, error) {
		var result []hcUUIDResult

		query := &neoism.CypherQuery{
			Statement: `MATCH (n:Person) 
					return  n.uuid as uuid
					limit 1`,
			Result: &result,
		}

		err := cr.CypherBatch([]*neoism.CypherQuery{query})

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
		TechnicalSummary: fmt.Sprintf("Cannot connect to Neo4j instance %s with at least one person loaded in it", cr),
		Checker:          checker,
	}
}
