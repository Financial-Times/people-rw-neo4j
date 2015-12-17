package main

import (
	"github.com/jmcvetta/neoism"
	"time"
)

type CypherRunner interface {
	CypherBatch(queries []*neoism.CypherQuery) error
}

func NewBatchCypherRunner(cypherRunner CypherRunner, count int, duration time.Duration) CypherRunner {
	cr := BatchCypherRunner{cypherRunner, make(chan cypherBatch), count, duration}

	go cr.batcher()

	return &cr
}

type BatchCypherRunner struct {
	cr       CypherRunner
	ch       chan cypherBatch
	count    int
	duration time.Duration
}

func (bcr *BatchCypherRunner) CypherBatch(queries []*neoism.CypherQuery) error {

	errCh := make(chan error)
	bcr.ch <- cypherBatch{queries, errCh}
	return <-errCh
}

type cypherBatch struct {
	queries []*neoism.CypherQuery
	err     chan error
}

func (bcr *BatchCypherRunner) batcher() {
	var currentQueries []*neoism.CypherQuery
	var currentErrorChannels []chan error
	var timeCh <-chan time.Time
	for {
		select {
		case cb := <-bcr.ch:
			timeCh = time.NewTimer(bcr.duration).C

			for _, query := range cb.queries {
				currentQueries = append(currentQueries, query)
			}
			currentErrorChannels = append(currentErrorChannels, cb.err)

			if len(currentQueries) < bcr.count {
				continue
			}
		case <-timeCh:
			//do nothing
		}
		err := bcr.cr.CypherBatch(currentQueries)
		for _, cec := range currentErrorChannels {
			cec <- err
		}
		currentQueries = currentQueries[0:0] // clears the slice
		currentErrorChannels = currentErrorChannels[0:0]
		timeCh = nil
	}
}

type PeopleDriver interface {
	Write(p person) error
	Read(uuid string) (p person, found bool, err error)
}

type PeopleCypherDriver struct {
	cypherRunner CypherRunner
}

func NewPeopleCypherDriver(cypherRunner CypherRunner) PeopleCypherDriver {
	return PeopleCypherDriver{cypherRunner}
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

	err := pcd.cypherRunner.CypherBatch([]*neoism.CypherQuery{query})

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

	return pcd.cypherRunner.CypherBatch([]*neoism.CypherQuery{query})

}

const (
	fsAuthority = "http://api.ft.com/system/FACTSET-PPL"
)
