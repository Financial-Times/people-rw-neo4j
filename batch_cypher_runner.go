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
