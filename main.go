package main

import (
	"encoding/json"
	"fmt"
	"github.com/Financial-Times/go-fthealth/v1a"
	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"
	"github.com/jawher/mow.cli"
	"github.com/jmcvetta/neoism"
	"io"
	"net/http"
	"os"
)

var peopleCypherWriter PeopleWriter

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

	peopleCypherWriter = NewPeopleCypherWriter(db)

	r := mux.NewRouter()
	r.HandleFunc("/people/{uuid}", peopleWrite).Methods("PUT")
	r.HandleFunc("/__health", v1a.Handler("PeopleReadWriteNeo4j Healthchecks",
		"Checks for accessing neo4j", setUpHealthCheck()))
	r.HandleFunc("/ping", ping)
	http.ListenAndServe(":"+port, handlers.CombinedLoggingHandler(os.Stdout, r))
}

func setUpHealthCheck() v1a.Check {
	return v1a.Check{
		BusinessImpact:   "blah",
		Name:             "My check",
		PanicGuide:       "Don't panic",
		Severity:         1,
		TechnicalSummary: "Something technical",
		Checker:          checker,
	}
}

func checker() (string, error) {
	return "", nil
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

	peopleCypherWriter.Write(p)
}

func parsePerson(jsonInput io.Reader) (person, error) {
	dec := json.NewDecoder(jsonInput)
	var p person
	err := dec.Decode(&p)
	return p, err
}

type person struct {
	Identifiers []struct {
		Authority       string `json:"authority"`
		IdentifierValue string `json:"identifierValue"`
	} `json:"identifiers"`
	Name string `json:"name"`
	UUID string `json:"uuid"`
}
