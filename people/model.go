package people

type person struct {
	UUID                   string                 `json:"uuid"`
	BirthYear              int                    `json:"birthYear,omitempty"`
	Name                   string                 `json:"name,omitempty"`
	Salutation             string                 `json:"salutation,omitempty"`
	Aliases                []string               `json:"aliases,omitempty"`
	AlternativeIdentifiers alternativeIdentifiers `json:"alternativeIdentifiers"`
	Types                  []string               `json:"types,omitempty"`
}

type alternativeIdentifiers struct {
	TME               []string `json:"TME,omitempty"`
	UUIDS             []string `json:"uuids"`
	FactsetIdentifier string   `json:"factsetIdentifier,omitempty"`
}

const (
	tmeIdentifierLabel     = "TMEIdentifier"
	uppIdentifierLabel     = "UPPIdentifier"
	factsetIdentifierLabel = "FactsetIdentifier"
)
