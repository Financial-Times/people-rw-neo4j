package main

import (
	"errors"
	"github.com/jmcvetta/neoism"
	"github.com/stretchr/testify/assert"
	"testing"
	"time"
)

func TestBatchByCount(t *testing.T) {
	assert := assert.New(t)
	mr := &mockRunner{}
	batchCypherRunner := NewBatchCypherRunner(mr, 3, time.Millisecond*20)

	errCh := make(chan error)

	go func() {
		errCh <- batchCypherRunner.CypherBatch([]*neoism.CypherQuery{
			&neoism.CypherQuery{Statement: "First"},
			&neoism.CypherQuery{Statement: "Second"},
		})
	}()

	go func() {
		time.Sleep(time.Millisecond * 10)
		errCh <- batchCypherRunner.CypherBatch([]*neoism.CypherQuery{
			&neoism.CypherQuery{Statement: "Third"},
		})
	}()

	for i := 0; i < 2; i++ {
		err := <-errCh
		assert.NoError(err, "Got an error for %d", i)
	}

	expected := []*neoism.CypherQuery{
		&neoism.CypherQuery{Statement: "First"},
		&neoism.CypherQuery{Statement: "Second"},
		&neoism.CypherQuery{Statement: "Third"},
	}

	assert.Equal(expected, mr.queriesRun, "queries didn't match")
}

func TestBatchByTimeout(t *testing.T) {
	assert := assert.New(t)
	mr := &mockRunner{}
	batchCypherRunner := NewBatchCypherRunner(mr, 3, time.Millisecond*20)

	errCh := make(chan error)

	go func() {
		errCh <- batchCypherRunner.CypherBatch([]*neoism.CypherQuery{
			&neoism.CypherQuery{Statement: "First"},
		})
	}()

	timer := time.NewTimer(time.Millisecond * 10)

	select {
	case <-timer.C:
		assert.NoError(<-errCh, "Got an error") //expect the timer to expire first, so check we didn't get an error
	case <-errCh:
		t.Fatal("Processed query ahead of timeout")
	}

	expected := []*neoism.CypherQuery{
		&neoism.CypherQuery{Statement: "First"},
	}

	assert.Equal(expected, mr.queriesRun, "queries didn't match")
}

func TestEveryoneGetsErrorOnFailure(t *testing.T) {
	assert := assert.New(t)
	mr := &failRunner{}
	batchCypherRunner := NewBatchCypherRunner(mr, 3, time.Millisecond*20)

	errCh := make(chan error)

	go func() {
		errCh <- batchCypherRunner.CypherBatch([]*neoism.CypherQuery{
			&neoism.CypherQuery{Statement: "First"},
			&neoism.CypherQuery{Statement: "Second"},
		})
	}()

	go func() {
		errCh <- batchCypherRunner.CypherBatch([]*neoism.CypherQuery{
			&neoism.CypherQuery{Statement: "Third"},
		})
	}()

	for i := 0; i < 2; i++ {
		err := <-errCh
		assert.Error(err, "Didn't get an error for %d", i)
	}

	assert.Equal(len(errCh), 0, "too many errors")
}

type mockRunner struct {
	queriesRun []*neoism.CypherQuery
}

func (mr *mockRunner) CypherBatch(queries []*neoism.CypherQuery) error {
	if mr.queriesRun != nil {
		return errors.New("Should not have any queries waiting")
	}
	mr.queriesRun = queries
	return nil
}

type failRunner struct {
}

func (mr *failRunner) CypherBatch(queries []*neoism.CypherQuery) error {
	return errors.New("Fail for every query")
}
