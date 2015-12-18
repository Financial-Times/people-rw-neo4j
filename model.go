package main

type person struct {
	Identifiers []identifier `json:"identifiers"`
	Name        string       `json:"name"`
	UUID        string       `json:"uuid"`
}

type identifier struct {
	Authority       string `json:"authority"`
	IdentifierValue string `json:"identifierValue"`
}
