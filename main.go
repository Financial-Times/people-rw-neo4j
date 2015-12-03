package main

import (
	"encoding/json"
	"fmt"
	"github.com/Financial-Times/go-fthealth/v1a"
	"github.com/gorilla/mux"
	"github.com/jmcvetta/neoism"
	"io"
	"net/http"
)

var db *neoism.Database

func main() {
	var err error
	db, err = neoism.Connect("http://localhost:7474/db/data")
	if err != nil {
		panic(err)
	}

	r := mux.NewRouter()
	r.HandleFunc("/people/{uuid}", peopleWrite).Methods("PUT")
	r.HandleFunc("/__health", v1a.Handler("PeopleReadWriteNeo4j Healthchecks",
		"Checks for accessing neo4j", setUpHealthCheck()))
	http.ListenAndServe(":19080", r)
}

func setUpHealthCheck() v1a.Check {
	return v1a.Check{
		BusinessImpact:   "blah",
		Name:             "My check",
		PanicGuide:       "Don't panic",
		Severity:         1,
		TechnicalSummary: "Something technical",
		Checker:          func() (string, error) { return "", nil }, //TODO: create the real check
	}
}

func peopleWrite(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	uuid := vars["uuid"]
	p, err := parsePerson(r.Body)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	peopleCypherWriter := NewPeopleCypherWriter()

	peopleCypherWriter.Write(p)

	io.WriteString(w, fmt.Sprintf("Hello %s!", uuid))
}

func parsePerson(jsonInput io.Reader) (person, error) {
	dec := json.NewDecoder(jsonInput)
	var p person
	err := dec.Decode(&p)
	return p, err
}

func writeCypher(p person, peopleWriter PeopleCypherWriter) error {
	fmt.Println(p.UUID)
	//peopleWriter.write()
	return nil
}

type person struct {
	Identifiers []struct {
		Authority       string `json:"authority"`
		IdentifierValue string `json:"identifierValue"`
	} `json:"identifiers"`
	Name string `json:"name"`
	UUID string `json:"uuid"`
}

type PeopleWriter interface {
	Write(p person)
}

type PeopleCypherWriter struct {
}

func NewPeopleCypherWriter() PeopleCypherWriter {
	return PeopleCypherWriter{}
}

func (pcw *PeopleCypherWriter) Write(p person) {
	fmt.Println("writing cypher")

	result := []struct {
		N neoism.Node
	}{}

	params := map[string]interface{}{
		"name": p.Name,
		"uuid": p.UUID,
	}

	for _, identifier := range p.Identifiers {
		if identifier.Authority == "http://api.ft.com/system/FACTSET-PPL" {
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
		Result: result,
	}

	err := db.Cypher(query)

	if err != nil {
		panic(err)
	}

}
