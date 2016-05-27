package people

import (
	"encoding/json"
	"fmt"

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
		"Thing":             "uuid",
		"Concept":           "uuid",
		"Person":            "uuid",
		"FactsetIdentifier": "value",
		"TMEIdentifier":     "value"})
}

func (s service) Read(uuid string) (interface{}, bool, error) {
	results := []person{}

	readQuery := &neoism.CypherQuery{
		Statement: `MATCH (p:Person {uuid:{uuid}})
					OPTIONAL MATCH (p)<-[rel:IDENTIFIES]-(i:Identifier)
					WITH p,collect({authority:i.authority, identifierValue:i.value}) as identifiers
						return p.uuid as uuid,
									 p.name as name,
									 identifiers,
									 p.birthYear as birthYear,
									 p.salutation as salutation,
									 p.aliases as aliases`,
		Parameters: map[string]interface{}{
			"uuid": uuid,
		},
		Result: &results,
	}

	if err := s.cypherRunner.CypherBatch([]*neoism.CypherQuery{readQuery}); err != nil || len(results) == 0 {
		return person{}, false, err
	}

	result := results[0]

	if len(result.Identifiers) == 1 && (result.Identifiers[0].IdentifierValue == "") {
		result.Identifiers = []identifier{}
	}

	p := person{
		UUID:        result.UUID,
		Name:        result.Name,
		BirthYear:   result.BirthYear,
		Salutation:  result.Salutation,
		Identifiers: result.Identifiers,
		Aliases:     result.Aliases,
	}

	sortIdentifiers(p.Identifiers)

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

	var aliases []string

	for _, alias := range p.Aliases {
		aliases = append(aliases, alias)
	}

	if len(aliases) > 0 {
		params["aliases"] = aliases
	}

	deleteEntityRelationshipsQuery := &neoism.CypherQuery{
		Statement: `MATCH (t:Thing {uuid:{uuid}})
					OPTIONAL MATCH (i:Identifier)-[ir:IDENTIFIES]->(t)
					DELETE ir, i`,
		Parameters: map[string]interface{}{
			"uuid": p.UUID,
		},
	}

	queries := []*neoism.CypherQuery{deleteEntityRelationshipsQuery}

	writeQuery := &neoism.CypherQuery{
		Statement: `MERGE (n:Thing{uuid: {uuid}})
						set n={props}
						set n :Concept
						set n :Person `,
		Parameters: map[string]interface{}{
			"uuid":  p.UUID,
			"props": params,
		},
	}

	queries = append(queries, writeQuery)

	identifierLabels := map[string]string{
		fsAuthority:  "FactsetIdentifier",
		tmeAuthority: "TMEIdentifier",
	}

	for _, identifier := range p.Identifiers {
		if identifierLabels[identifier.Authority] == "" {
			return requestError{fmt.Sprintf("This identifier type- %v, is not supported. Only '%v' and '%v' are currently supported", identifier.Authority, fsAuthority, tmeAuthority)}
		}
		addIdentifierQuery := addIdentifierQuery(identifier, p.UUID, identifierLabels[identifier.Authority])
		queries = append(queries, addIdentifierQuery)
	}

	return s.cypherRunner.CypherBatch(queries)
}

func addIdentifierQuery(identifier identifier, uuid string, identifierLabel string) *neoism.CypherQuery {
	statementTemplate := fmt.Sprintf(`MERGE (o:Thing {uuid:{uuid}})
								CREATE (i:Identifier {value:{value} , authority:{authority}})
								CREATE (o)<-[:IDENTIFIES]-(i)
								set i : %s `, identifierLabel)
	query := &neoism.CypherQuery{
		Statement: statementTemplate,
		Parameters: map[string]interface{}{
			"uuid":      uuid,
			"value":     identifier.IdentifierValue,
			"authority": identifier.Authority,
		},
	}
	return query
}

func (s service) Delete(uuid string) (bool, error) {
	clearNode := &neoism.CypherQuery{
		Statement: `
			MATCH (p:Thing {uuid: {uuid}})
			OPTIONAL MATCH (p)<-[ir:IDENTIFIES]-(i:Identifier)
			REMOVE p:Concept
			REMOVE p:Person
			DELETE ir, i
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

type requestError struct {
	details string
}

func (re requestError) Error() string {
	return "Invalid Request"
}

func (re requestError) InvalidRequestDetails() string {
	return re.details
}

const (
	fsAuthority  = "http://api.ft.com/system/FACTSET-PPL"
	tmeAuthority = "http://api.ft.com/system/FT-TME"
)
