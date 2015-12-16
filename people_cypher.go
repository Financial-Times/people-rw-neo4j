package main

import (
	"github.com/jmcvetta/neoism"
)

type CypherRunner interface {
	Cypher() error
}

type PeopleDriver interface {
	Write(p person) error
	Read(uuid string) (p person, found bool, err error)
}

type PeopleCypherDriver struct {
	db *neoism.Database
}

func NewPeopleCypherDriver(db *neoism.Database) PeopleCypherDriver {
	return PeopleCypherDriver{db}
}

func (pcd PeopleCypherDriver) Read(uuid string) (person, bool, error) {
	results := []struct {
		UUID              string `json:"uuid"`
		Name              string `json: "name"`
		FactsetIdentifier string `json: "factsetIdentifier"`
	}{}

	query := &neoism.CypherQuery{
		Statement: `MATCH (n:Person {uuid:{uuid}}) return n.uuid as uuid, n.name as name, n.factsetIdentifier as factsetIdentifier`,
		Parameters: map[string]interface{}{
			"uuid": uuid,
		},
		Result: &results,
	}

	err := pcd.db.CypherBatch([]*neoism.CypherQuery{query})

	if err != nil {
		return person{}, false, err
	}

	if len(results) == 0 {
		return person{}, false, nil
	}

	result := results[0]

	p := person{
		UUID: result.UUID,
		Name: result.Name,
	}

	if result.FactsetIdentifier != "" {
		p.Identifiers = append(p.Identifiers, identifier{fsAuthority, result.FactsetIdentifier})
	}

	return p, true, nil

}

func (pcd PeopleCypherDriver) Write(p person) error {

	params := map[string]interface{}{
		"name": p.Name,
		"uuid": p.UUID,
	}

	for _, identifier := range p.Identifiers {
		if identifier.Authority == fsAuthority {
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
	}

	return pcd.db.CypherBatch([]*neoism.CypherQuery{query})

}

const (
	fsAuthority = "http://api.ft.com/system/FACTSET-PPL"
)
