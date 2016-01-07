package main

import (
	"github.com/Financial-Times/neo-cypher-runner-go"
	log "github.com/Sirupsen/logrus"
	"github.com/jmcvetta/neoism"
)

type PeopleDriver interface {
	Write(p person) error
	Read(uuid string) (p person, found bool, err error)
	Delete(uuid string) (found bool, err error)
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

	log.Debugf("Executing query %s", query)

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

func (pcd PeopleCypherDriver) Delete(uuid string) (bool, error) {
	clearNode := &neoism.CypherQuery{
		Statement: `
			MATCH (p:Thing {uuid: {uuid}})
			REMOVE p:Concept
			REMOVE p:Person
			SET p={props}
		`,
		Parameters: map[string]interface{}{
			"uuid": uuid,
			"props": map[string]interface{}{
				"uuid": uuid,
			},
		},
		IncludeStats: true,
	}

	removeNodeIfUnused := &neoism.CypherQuery{
		Statement: `
			MATCH (p:Thing {uuid: {uuid}})
			OPTIONAL MATCH (p)-[a]-(x)
			WITH p, count(a) AS relCount
			WHERE relCount = 0
			DELETE p
		`,
		Parameters: map[string]interface{}{
			"uuid": uuid,
		},
	}

	err := pcd.cypherRunner.CypherBatch([]*neoism.CypherQuery{clearNode, removeNodeIfUnused})

	s1, err := clearNode.Stats()
	if err != nil {
		return false, err
	}

	var deleted bool
	if s1.ContainsUpdates && s1.LabelsRemoved > 0 {
		deleted = true
	}

	return deleted, err
}

const (
	fsAuthority = "http://api.ft.com/system/FACTSET-PPL"
)
