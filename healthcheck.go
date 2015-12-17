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
			return "", errors.New("UUID not set")
		}
		if result[0].UUID == "" {
			return "", errors.New("UUID not set")
		}
		return fmt.Sprintf("Found a person with a valid uuid = %v", result[0].UUID), nil
	}

	return v1a.Check{
		BusinessImpact:   "blah",
		Name:             "My check",
		PanicGuide:       "Don't panic",
		Severity:         1,
		TechnicalSummary: "Something technical",
		Checker:          checker,
	}
}
