package main

import (
	"github.com/jmcvetta/neoism"
	"github.com/rcrowley/go-metrics"
	"log"
	"time"
)

type CypherRunner interface {
	CypherBatch(queries []*neoism.CypherQuery) error
}

func NewBatchCypherRunner(cypherRunner CypherRunner, count int, duration time.Duration) CypherRunner {
	cr := BatchCypherRunner{cypherRunner, make(chan cypherQueryBatch), count, duration}

	go cr.batcher()

	return &cr
}

type BatchCypherRunner struct {
	cr       CypherRunner
	ch       chan cypherQueryBatch
	count    int
	duration time.Duration
}

func (bcr *BatchCypherRunner) CypherBatch(queries []*neoism.CypherQuery) error {

	errCh := make(chan error)
	bcr.ch <- cypherQueryBatch{queries, errCh}
	return <-errCh
}

type cypherQueryBatch struct {
	queries []*neoism.CypherQuery
	err     chan error
}

func (bcr *BatchCypherRunner) batcher() {
	g := metrics.GetOrRegisterGauge("batchQueueSize", metrics.DefaultRegistry)
	b := metrics.GetOrRegisterMeter("batchThroughput", metrics.DefaultRegistry)
	var currentQueries []*neoism.CypherQuery
	var currentErrorChannels []chan error
	timer := time.NewTimer(1 * time.Second)
	for {
		select {
		case cb := <-bcr.ch:
			for _, query := range cb.queries {
				currentQueries = append(currentQueries, query)
				g.Update(int64(len(currentQueries)))
			}
			currentErrorChannels = append(currentErrorChannels, cb.err)

			if len(currentQueries) < bcr.count {
				timer.Reset(bcr.duration)
				continue
			}
		case <-timer.C:
			//do nothing
		}
		if len(currentQueries) > 0 {
			t := metrics.GetOrRegisterTimer("execute-neo4j-batch", metrics.DefaultRegistry)
			var err error
			t.Time(func() {
				err = bcr.cr.CypherBatch(currentQueries)
			})
			if err != nil {
				log.Printf("Got error running batch, error=%v", err)
			}
			for _, cec := range currentErrorChannels {
				cec <- err
			}
			b.Mark(int64(len(currentQueries)))
			g.Update(0)
			currentQueries = currentQueries[0:0] // clears the slice
			currentErrorChannels = currentErrorChannels[0:0]
		}
	}
}
