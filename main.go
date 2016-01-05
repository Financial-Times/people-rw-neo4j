package main

import (
	"encoding/json"
	"fmt"
	"github.com/Financial-Times/go-fthealth/v1a"
	"github.com/cyberdelia/go-metrics-graphite"
	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"
	"github.com/jawher/mow.cli"
	"github.com/jmcvetta/neoism"
	"github.com/rcrowley/go-metrics"
	"io"
	"log"
	"net"
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
	timeoutMs := app.IntOpt("timeoutMs", 20,
		"Number of milliseconds to wait before executing a batch of statements regardless of its size")
	graphiteTCPAddress := app.StringOpt("graphiteTCPAddress", "",
		"Graphite TCP address, e.g. graphite.ft.com:2003. Leave as default if you do NOT want to output to graphite (e.g. if running locally)")
	graphitePrefix := app.StringOpt("graphitePrefix", "",
		"Prefix to use. Should start with content, include the environment, and the host name. e.g. content.test.people.rw.neo4j.ftaps58938-law1a-eu-t")
	logMetrics := app.BoolOpt("logMetrics", false, "Whether to log metrics. Set to true if running locally and you want metrics output")

	app.Action = func() {
		runServer(*neoURL, *port, *batchSize, *timeoutMs, *graphiteTCPAddress, *graphitePrefix, *logMetrics)
	}

	app.Run(os.Args)
}

func runServer(neoURL string, port string, batchSize int, timeoutMs int, graphiteTCPAddress string,
	graphitePrefix string, logMetrics bool) {

	flags := log.Ldate | log.Ltime | log.Lshortfile

	log.SetFlags(flags)

	db, err := neoism.Connect(neoURL)
	if err != nil {
		log.Println("ERROR Could not connect to neo4j, error=[%s]", err)
	}

	ensureIndex(db, "Person", "uuid")

	peopleDriver = NewPeopleCypherDriver(NewBatchCypherRunner(db, batchSize, time.Millisecond*time.Duration(timeoutMs)))

	if graphiteTCPAddress != "" {
		addr, _ := net.ResolveTCPAddr("tcp", graphiteTCPAddress)
		go graphite.Graphite(metrics.DefaultRegistry, 1*time.Minute, graphitePrefix, addr)
	}
	if logMetrics { //useful locally
		go metrics.Log(metrics.DefaultRegistry, 60*time.Second, log.New(os.Stderr, "metrics: ", log.Lmicroseconds))
	}

	r := mux.NewRouter()
	r.HandleFunc("/people/{uuid}", peopleWrite).Methods("PUT")
	r.HandleFunc("/people/{uuid}", peopleRead).Methods("GET")
	r.HandleFunc("/__health", v1a.Handler("PeopleReadWriteNeo4j Healthchecks",
		"Checks for accessing neo4j", setUpHealthCheck(db)))
	r.HandleFunc("/ping", ping)
	http.ListenAndServe(":"+port, HttpMetricsHandler(handlers.CombinedLoggingHandler(os.Stdout, r)))
}

func ensureIndex(db *neoism.Database, label string, property string) {

	personIndexes, err := db.Indexes(label)

	if err != nil {
		log.Println("ERROR Error on creating index=%v", err)
	}

	var indexFound bool

	for _, index := range personIndexes {
		if len(index.PropertyKeys) == 1 && index.PropertyKeys[0] == property {
			indexFound = true
			break
		}
	}
	if !indexFound {
		log.Println("INFO Creating index for person for neo4j instance at %s", db.Url)
		db.CreateIndex(label, property)
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
		log.Println("ERROR Error on parse=%v", err)
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	err = peopleDriver.Write(p)
	if err != nil {
		log.Println("ERROR Error on write=%v", err)
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
		log.Println("ERROR Error on read=%v", err)
		w.WriteHeader(http.StatusServiceUnavailable)
		return
	}

	if !found {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	enc := json.NewEncoder(w)

	if err := enc.Encode(p); err != nil {
		log.Println("ERROR Error on json encoding=%v", err)
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
