package main

import (
	"github.com/jmcvetta/neoism"
)

type PeopleWriter interface {
	Write(p person)
}

type PeopleCypherWriter struct {
	db *neoism.Database
}

func NewPeopleCypherWriter(db *neoism.Database) PeopleCypherWriter {
	return PeopleCypherWriter{db}
}

func (pcw PeopleCypherWriter) Write(p person) {
	result := []struct {
		N neoism.Node
	}{}

	params := map[string]interface{}{
		"name": p.Name,
		"uuid": p.UUID,
	}

	for _, identifier := range p.Identifiers {
		if identifier.Authority == "http://api.ft.com/system/FACTSET-PPL" {
			params["factsetIdentifier"] = identifier.IdentifierValue
		}
	}

	query := &neoism.CypherQuery{
		Statement: `MERGE (n:Person {uuid: {uuid}}) 
					set n={allprops}
					return  n`,
		Parameters: map[string]interface{}{
			"uuid":     p.UUID,
			"allprops": params,
		},
		Result: result,
	}

	err := pcw.db.Cypher(query)

	if err != nil {
		panic(err)
	}

}
