package main

type person struct {
	BirthYear   int          `json:"birthYear"`
	Identifiers []identifier `json:"identifiers"`
	Name        string       `json:"name"`
	UUID        string       `json:"uuid"`
	Salutation  string       `json:"salutation"`
}

type identifier struct {
	Authority       string `json:"authority"`
	IdentifierValue string `json:"identifierValue"`
}
