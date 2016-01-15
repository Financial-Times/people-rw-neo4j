package people

type person struct {
	UUID        string       `json:"uuid"`
	BirthYear   int          `json:"birthYear,omitempty"`
	Identifiers []identifier `json:"identifiers,omitempty"`
	Name        string       `json:"name,omitempty"`
	Salutation  string       `json:"salutation,omitempty"`
}

type identifier struct {
	Authority       string `json:"authority"`
	IdentifierValue string `json:"identifierValue"`
}
