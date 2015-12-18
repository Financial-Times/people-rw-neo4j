# People Reader/Writer for Neo4j (people-rw-neo4j)

__An API for reading/writing people into Neo4j. Expects the people json supplied to be in the format that comes out of the people transformer.__

## Installation

For the first time: 

`go get github.com/Financial-Times/people-rw-neo4j` 

or update: 

`go get -u github.com/Financial-Times/people-rw-neo4j`
	
## Running

`$GOPATH/bin/people-rw-neo4j --neo-url={neo4jUrl} --port={port} --batchSize=50 --timeoutMs=20

All arguments are optional, they default to a local Neo4j install on the default port (7474), application running on port 8080, batchSize of 1024 and timeoutMs of 50. NB: the default batchSize is much higher than the throughput the instance data ingester currently can cope with.

## Try it!

`curl -XPUT -H "X-Request-Id: 123" -H "Content-Type: application/json" localhost:8080/people/3fa70485-3a57-3b9b-9449-774b001cd965 --data '{"uuid":"3fa70485-3a57-3b9b-9449-774b001cd965", "name":"Robert W. Addington", "identifiers":[{ "authority":"http://api.ft.com/system/FACTSET-PPL", "identifierValue":"000BJG-E"}]}'`

`curl -H "X-Request-Id: 123" localhost:8080/people/3fa70485-3a57-3b9b-9449-774b001cd965`

Healthchecks: [http://localhost:8080/__health](http://localhost:8080/__health)

Good-to-go: [http://localhost:8080/__gtg](http://localhost:8080/__gtg)

Ping: [http://localhost:8080/ping](http://localhost:8080/ping)
