package main

import (
	"github.com/Financial-Times/neo-cypher-runner-go"
	"github.com/jmcvetta/neoism"
)

type PeopleDriver interface {
	Write(p person) error
	Read(uuid string) (p person, found bool, err error)
	Delete(uuid string) error
}

type PeopleCypherDriver struct {
	cypherRunner neocypherrunner.CypherRunner
}

func NewPeopleCypherDriver(cypherRunner neocypherrunner.CypherRunner) PeopleCypherDriver {
	return PeopleCypherDriver{cypherRunner}
}

func (pcd PeopleCypherDriver) Read(uuid string) (person, bool, error) {
	results := []struct {
		UUID              string `json:"uuid"`
		Name              string `json: "name"`
		BirthYear         int    `json: "birthYear"`
		Salutation        string `json: "salutation"`
		FactsetIdentifier string `json: "factsetIdentifier"`
	}{}

	query := &neoism.CypherQuery{
		Statement: `MATCH (n:Person {uuid:{uuid}}) return n.uuid 
		as uuid, n.name as name, 
		n.factsetIdentifier as factsetIdentifier,
		n.birthYear as birthYear,
		n.salutation as salutation`,
		Parameters: map[string]interface{}{
			"uuid": uuid,
		},
		Result: &results,
	}

	err := pcd.cypherRunner.CypherBatch([]*neoism.CypherQuery{query})

	if err != nil {
		return person{}, false, err
	}

	if len(results) == 0 {
		return person{}, false, nil
	}

	result := results[0]

	p := person{
		UUID:       result.UUID,
		Name:       result.Name,
		BirthYear:  result.BirthYear,
		Salutation: result.Salutation,
	}

	if result.FactsetIdentifier != "" {
		p.Identifiers = append(p.Identifiers, identifier{fsAuthority, result.FactsetIdentifier})
	}

	return p, true, nil

}

func (pcd PeopleCypherDriver) Write(p person) error {

	params := map[string]interface{}{
		"uuid": p.UUID,
	}

	if p.Name != "" {
		params["name"] = p.Name
	}

	if p.BirthYear != 0 {
		params["birthYear"] = p.BirthYear
	}

	if p.Salutation != "" {
		params["salutation"] = p.Salutation
	}

	for _, identifier := range p.Identifiers {
		if identifier.Authority == fsAuthority {
			params["factsetIdentifier"] = identifier.IdentifierValue
		}
	}

	query := &neoism.CypherQuery{
		Statement: `MERGE (n:Thing {uuid: {uuid}}) 
					set n={allprops}
					set n :Concept
					set n :Person
		`,
		Parameters: map[string]interface{}{
			"uuid":     p.UUID,
			"allprops": params,
		},
	}

	return pcd.cypherRunner.CypherBatch([]*neoism.CypherQuery{query})

}

func (pcd PeopleCypherDriver) Delete(uuid string) error {
	//TODO: this need to use the approach described in :
	// https://docs.google.com/document/d/1Ec-umbNOZa9zht2FImAY-fsMDFgDDxBUMeokDTZZ3tQ/edit#heading=h.pgfg88uoy07a
	query := &neoism.CypherQuery{
		Statement: `MATCH (n:Person {uuid:{uuid}}) DELETE n`,
		Parameters: map[string]interface{}{
			"uuid": uuid,
		},
	}

	return pcd.cypherRunner.CypherBatch([]*neoism.CypherQuery{query})
}

const (
	fsAuthority = "http://api.ft.com/system/FACTSET-PPL"
)
