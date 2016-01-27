package people

type person struct {
	UUID        string       `json:"uuid"`
	BirthYear   int          `json:"birthYear,omitempty"`
	Identifiers []identifier `json:"identifiers,omitempty"`
	Name        string       `json:"name,omitempty"`
	Salutation  string       `json:"salutation,omitempty"`
	Aliases     []string     `json:"aliases,omitempty"`
}

type identifier struct {
	Authority       string `json:"authority"`
	IdentifierValue string `json:"identifierValue"`
}
