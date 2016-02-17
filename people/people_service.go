package people

import (
	"encoding/json"

	"github.com/Financial-Times/neo-utils-go/neoutils"
	"github.com/jmcvetta/neoism"
)

type service struct {
	cypherRunner neoutils.CypherRunner
	indexManager neoutils.IndexManager
}

// NewCypherPeopleService provides functions for create, update, delete operations on people in Neo4j,
// plus other utility functions needed for a service
func NewCypherPeopleService(cypherRunner neoutils.CypherRunner, indexManager neoutils.IndexManager) service {
	return service{cypherRunner, indexManager}
}

func (s service) Initialise() error {
	return neoutils.EnsureConstraints(s.indexManager, map[string]string{
		"Thing":   "uuid",
		"Concept": "uuid",
		"Person":  "uuid"})
}

func (s service) Read(uuid string) (interface{}, bool, error) {
	results := []struct {
		UUID              string   `json:"uuid"`
		Name              string   `json:"name"`
		BirthYear         int      `json:"birthYear"`
		Salutation        string   `json:"salutation"`
		FactsetIdentifier string   `json:"factsetIdentifier"`
		TMEIdentifiers    []string `json:"tmeIdentifiers"`
		Aliases           []string `json:"aliases"`
	}{}

	query := &neoism.CypherQuery{
		Statement: `MATCH (n:Person {uuid:{uuid}}) return n.uuid
		as uuid, n.name as name,
		n.factsetIdentifier as factsetIdentifier,
		n.tmeIdentifiers as tmeIdentifiers,
		n.birthYear as birthYear,
		n.salutation as salutation,
		n.aliases as aliases`,
		Parameters: map[string]interface{}{
			"uuid": uuid,
		},
		Result: &results,
	}

	err := s.cypherRunner.CypherBatch([]*neoism.CypherQuery{query})

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
		Aliases:    result.Aliases,
	}

	if result.FactsetIdentifier != "" {
		p.Identifiers = append(p.Identifiers, identifier{fsAuthority, result.FactsetIdentifier})
	}

	// if len(result.TMEIdentifiers) > 0 {
	// 	for _, tmeValue := range result.TMEIdentifiers {
	// 		p.Identifiers = append(p.Identifiers, identifier{tmeAuthority, tmeValue})
	// 	}
	// }

	for _, tmeValue := range result.TMEIdentifiers {
		p.Identifiers = append(p.Identifiers, identifier{tmeAuthority, tmeValue})
	}

	return p, true, nil

}

func (s service) Write(thing interface{}) error {

	p := thing.(person)

	params := map[string]interface{}{
		"uuid": p.UUID,
	}

	if p.Name != "" {
		params["name"] = p.Name
		params["prefLabel"] = p.Name
	}

	if p.BirthYear != 0 {
		params["birthYear"] = p.BirthYear
	}

	if p.Salutation != "" {
		params["salutation"] = p.Salutation
	}

	var tmeIdentifiers []string

	for _, identifier := range p.Identifiers {
		if identifier.Authority == fsAuthority {
			params["factsetIdentifier"] = identifier.IdentifierValue
		}
		if identifier.Authority == tmeAuthority {
			tmeIdentifiers = append(tmeIdentifiers, identifier.IdentifierValue)
		}
	}

	if len(tmeIdentifiers) > 0 {
		params["tmeIdentifiers"] = tmeIdentifiers
	}

	var aliases []string

	for _, alias := range p.Aliases {
		aliases = append(aliases, alias)
	}

	if len(aliases) > 0 {
		params["aliases"] = aliases
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

	return s.cypherRunner.CypherBatch([]*neoism.CypherQuery{query})

}

func (s service) Delete(uuid string) (bool, error) {
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

	err := s.cypherRunner.CypherBatch([]*neoism.CypherQuery{clearNode, removeNodeIfUnused})

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

func (s service) DecodeJSON(dec *json.Decoder) (interface{}, string, error) {
	p := person{}
	err := dec.Decode(&p)
	return p, p.UUID, err
}

func (s service) Check() error {
	return neoutils.Check(s.cypherRunner)
}

func (s service) Count() (int, error) {

	results := []struct {
		Count int `json:"c"`
	}{}

	query := &neoism.CypherQuery{
		Statement: `MATCH (n:Person) return count(n) as c`,
		Result:    &results,
	}

	err := s.cypherRunner.CypherBatch([]*neoism.CypherQuery{query})

	if err != nil {
		return 0, err
	}

	return results[0].Count, nil
}

const (
	fsAuthority = "http://api.ft.com/system/FACTSET-PPL"
)

const (
	tmeAuthority = "http://api.ft.com/system/TME"
)
