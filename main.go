package main

import (
	"encoding/json"
	"fmt"
	"github.com/Financial-Times/go-fthealth/v1a"
	//"github.com/cyberdelia/go-metrics-graphite"
	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"
	"github.com/jawher/mow.cli"
	"github.com/jmcvetta/neoism"
	"github.com/rcrowley/go-metrics"
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
	batchSize := app.IntOpt("batchSize", 1024, "Maximum number of statements to execute per batch")
	timeoutMs := app.IntOpt("timeoutMs", 20, "Number of milliseconds to wait before executing a batch of statements regardless of its size")

	app.Action = func() {
		runServer(*neoURL, *port, *batchSize, *timeoutMs)
	}

	app.Run(os.Args)
}

func runServer(neoURL string, port string, batchSize int, timeoutMs int) {
	db, err := neoism.Connect(neoURL)
	if err != nil {
		panic(err) //TODO change to log
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

	peopleDriver = NewPeopleCypherDriver(NewBatchCypherRunner(db, batchSize, time.Millisecond*time.Duration(timeoutMs)))

	//TODO - only do this for local running. For deployments, use a new arg to specify a graphite server to write to
	// and set up graphite integration
	go metrics.Log(metrics.DefaultRegistry, 60*time.Second, log.New(os.Stderr, "metrics: ", log.Lmicroseconds))

	r := mux.NewRouter()
	r.HandleFunc("/people/{uuid}", peopleWrite).Methods("PUT")
	r.HandleFunc("/people/{uuid}", peopleRead).Methods("GET")
	r.HandleFunc("/__health", v1a.Handler("PeopleReadWriteNeo4j Healthchecks",
		"Checks for accessing neo4j", setUpHealthCheck(db)))
	r.HandleFunc("/ping", ping)
	http.ListenAndServe(":"+port, HttpMetricsHandler(handlers.CombinedLoggingHandler(os.Stdout, r)))
}

func ping(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "pong")
}

func peopleWrite(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	uuid := vars["uuid"]
	p, err := parsePerson(r.Body)
	if err != nil || p.UUID != uuid {
		log.Printf("Error on parse=%v", err)
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	err = peopleDriver.Write(p)
	if err != nil {
		log.Printf("Error on write=%v", err)
		w.WriteHeader(http.StatusServiceUnavailable)
		return
	}
	//Not necessary for a 200 to be returned, but for PUT requests, if don't specify, don't see 200 status logged in request logs
	w.WriteHeader(http.StatusOK)
}

func peopleRead(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	uuid := vars["uuid"]

	p, found, err := peopleDriver.Read(uuid)

	w.Header().Add("Content-Type", "application/json")

	if err != nil {
		log.Printf("Error on read=%v", err)
		w.WriteHeader(http.StatusServiceUnavailable)
		return
	}

	if !found {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	enc := json.NewEncoder(w)

	if err := enc.Encode(p); err != nil {
		log.Printf("Error on json encoding=%v", err)
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

func HttpMetricsHandler(h http.Handler) http.Handler {
	return httpMetricsHandler{h}
}

type httpMetricsHandler struct {
	handler http.Handler
}

func (h httpMetricsHandler) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	t := metrics.GetOrRegisterTimer(req.Method, metrics.DefaultRegistry)
	t.Time(func() { h.handler.ServeHTTP(w, req) })
}
