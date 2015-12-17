package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/Financial-Times/go-fthealth/v1a"
	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"
	"github.com/jawher/mow.cli"
	"github.com/jmcvetta/neoism"
	"io"
	"log"
	"net/http"
	_ "net/http/pprof"
	"os"
	"time"
)

var peopleDriver PeopleDriver

func main() {
	fmt.Println(os.Args)
	app := cli.App("people-rw-neo4j", "A RESTful API for managing People in neo4j")
	neoURL := app.StringOpt("neo-url", "http://localhost:7474/db/data", "neo4j endpoint URL")
	port := app.StringOpt("port", "8080", "Port to listen on")

	app.Action = func() {
		runServer(*neoURL, *port)
	}

	app.Run(os.Args)
}

func runServer(neoURL string, port string) {
	db, err := neoism.Connect(neoURL)
	if err != nil {
		panic(err)
	}

	personIndexes, err := db.Indexes("Person")

	if err != nil {
		panic(err)
	}

	var indexFound bool

	for _, index := range personIndexes {
		if len(index.PropertyKeys) == 1 && index.PropertyKeys[0] == "uuid" {
			indexFound = true
			break
		}
	}
	if !indexFound {
		log.Printf("Creating index for person for neo4j instance at %s", neoURL)
		db.CreateIndex("Person", "uuid")
	}

	peopleDriver = NewPeopleCypherDriver(NewBatchCypherRunner(db, 1024, time.Millisecond*20))

	r := mux.NewRouter()
	r.HandleFunc("/people/{uuid}", peopleWrite).Methods("PUT")
	r.HandleFunc("/people/{uuid}", peopleRead).Methods("GET")
	r.HandleFunc("/__health", v1a.Handler("PeopleReadWriteNeo4j Healthchecks",
		"Checks for accessing neo4j", setUpHealthCheck(db)))
	r.HandleFunc("/ping", ping)
	http.ListenAndServe(":"+port, handlers.CombinedLoggingHandler(os.Stdout, r))
}

func setUpHealthCheck(db *neoism.Database) v1a.Check {

	checker := func() (string, error) {
		var result []struct {
			UUID string `json:"uuid"`
		}

		query := &neoism.CypherQuery{
			Statement: `MATCH (n:Person) 
					return  n.uuid as uuid
					limit 1`,
			Result: &result,
		}

		err := db.Cypher(query)

		if err != nil {
			return "", err
		}
		if len(result) == 0 {
			return "", errors.New("UUID not set")
		}
		if result[0].UUID == "" {
			return "", errors.New("UUID not set")
		}
		return fmt.Sprintf("Found a person with a valid uuid = %v", result[0].UUID), nil
	}

	return v1a.Check{
		BusinessImpact:   "blah",
		Name:             "My check",
		PanicGuide:       "Don't panic",
		Severity:         1,
		TechnicalSummary: "Something technical",
		Checker:          checker,
	}
}

func ping(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "pong")
}

func peopleWrite(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	uuid := vars["uuid"]
	p, err := parsePerson(r.Body)
	if err != nil || p.UUID != uuid {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	err = peopleDriver.Write(p)
	if err != nil {
		w.WriteHeader(http.StatusServiceUnavailable)
		return
	}
}

func peopleRead(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	uuid := vars["uuid"]

	p, found, err := peopleDriver.Read(uuid)

	w.Header().Add("Content-Type", "application/json")

	if err != nil {
		w.WriteHeader(http.StatusServiceUnavailable)
		return
	}

	if !found {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	enc := json.NewEncoder(w)

	if err := enc.Encode(p); err != nil {
		w.WriteHeader(http.StatusServiceUnavailable)
		return
	}

}

func parsePerson(jsonInput io.Reader) (person, error) {
	dec := json.NewDecoder(jsonInput)
	var p person
	err := dec.Decode(&p)
	return p, err
}

type person struct {
	Identifiers []identifier `json:"identifiers"`
	Name        string       `json:"name"`
	UUID        string       `json:"uuid"`
}

type identifier struct {
	Authority       string `json:"authority"`
	IdentifierValue string `json:"identifierValue"`
}
