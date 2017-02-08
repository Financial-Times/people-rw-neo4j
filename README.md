# People Reader/Writer for Neo4j (people-rw-neo4j)

[![Circle CI](https://circleci.com/gh/Financial-Times/people-rw-neo4j.svg?style=shield)](https://circleci.com/gh/Financial-Times/people-rw-neo4j)[![Go Report Card](https://goreportcard.com/badge/github.com/Financial-Times/people-rw-neo4j)](https://goreportcard.com/report/github.com/Financial-Times/people-rw-neo4j) [![Coverage Status](https://coveralls.io/repos/github/Financial-Times/people-rw-neo4j/badge.svg)](https://coveralls.io/github/Financial-Times/people-rw-neo4j)
__An API for reading/writing people into Neo4j. Expects the people json supplied to be in the format that comes out of the people transformer.__

## Installation

For the first time:

`go get github.com/Financial-Times/people-rw-neo4j`

or update:

`go get -u github.com/Financial-Times/people-rw-neo4j`

## Running

`$GOPATH/bin/people-rw-neo4j --neo-url={neo4jUrl} --port={port} --batchSize=50 --graphiteTCPAddress=graphite.ft.com:2003 --graphitePrefix=content.{env}.people.rw.neo4j.{hostname} --logMetrics=false

All arguments are optional, they default to a local Neo4j install on the default port (7474), application running on port 8080, batchSize of 1024, graphiteTCPAddress of "" (meaning metrics won't be written to Graphite), graphitePrefix of "" and logMetrics false.

NB: the default batchSize is much higher than the throughput the instance data ingester currently can cope with.

## Updating the model
Use gojson against a transformer endpoint to create a person struct and update the person/model.go file. NB: we DO need a separate identifier struct

`curl http://ftaps35629-law1a-eu-t:8080/transformers/people/ad60f5b2-4306-349d-92d8-cf9d9572a6f6 | gojson -name=person`

## Endpoints

/people/{uuid}


### PUT
The only mandatory field is the uuid, and the alternativeIdentifier uuids (because the uuid is also listed in the alternativeIdentifier uuids list). The uuid in the body must match the one used on the path.

Every request results in an attempt to update that person: unlike with GraphDB there is no check on whether the person already exists and whether there are any changes between what's there and what's being written. We just do a MERGE which is Neo4j for create if not there, update if it is there.

A successful PUT results in 200.

We run queries in batches. If a batch fails, all failing requests will get a 500 server error response.

Example PUT request:

    `curl -XPUT localhost:8080/people/3fa70485-3a57-3b9b-9449-774b001cd965 \
         -H "X-Request-Id: 123" \
         -H "Content-Type: application/json" \
         -d '{"uuid":"3fa70485-3a57-3b9b-9449-774b001cd965","birthYear":1974,"salutation":"Mr","name":"Robert W. Addington","prefLabel":"Robert Addington","twitterHandle":"@rwa","facebookProfile":"raddington","linkedinProfile":"robert-addington","description": "Some text","descriptionXML": "Some text containing <strong>markup</strong>","_imageUrl": "http://someimage.jpg","alternativeIdentifiers":{"TME":["MTE3-U3ViamVjdHM="],"uuids":["3fa70485-3a57-3b9b-9449-774b001cd965","6a2a0170-6afa-4bcc-b427-430268d2ac50"],"factsetIdentifier":"000BJG-E"},"type":"People"}'`

The type field is not currently validated - instead, the People Writer writes type People and its parent types (Thing, Concept) as labels for People.

Invalid json body input, or uuids that don't match between the path and the body will result in a 400 bad request response.

### GET
Thie internal read should return what got written (i.e., this isn't the public person read API)

If not found, you'll get a 404 response.

Empty fields are omitted from the response.
`curl -H "X-Request-Id: 123" localhost:8080/people/3fa70485-3a57-3b9b-9449-774b001cd965`

### DELETE
Will return 204 if successful, 404 if not found
`curl -XDELETE -H "X-Request-Id: 123" localhost:8080/people/3fa70485-3a57-3b9b-9449-774b001cd965`

### Admin endpoints
Healthchecks: [http://localhost:8080/__health](http://localhost:8080/__health)

Ping: [http://localhost:8080/ping](http://localhost:8080/ping) or [http://localhost:8080/__ping](http://localhost:8080/__ping)


### Logging
 the application uses logrus, the logfile is initialised in main.go.
 logging requires an env app parameter, for all environments  other than local logs are written to file
 when running locally logging is written to console (if you want to log locally to file you need to pass in an env parameter that is != local)
 NOTE: build-info end point is not logged as it is called every second from varnish and this information is not needed in  logs/splunk
