package people

type person struct {
	UUID                   string                 `json:"uuid"`
	BirthYear              int                    `json:"birthYear,omitempty"`
	AlternativeIdentifiers alternativeIdentifiers `json:"alternativeIdentifiers"`
	Name                   string                 `json:"name,omitempty"`
	PrefLabel              string                 `json:"prefLabel"`
	Salutation             string                 `json:"salutation,omitempty"`
	Aliases                []string               `json:"aliases,omitempty"`
	EmailAddress           string                 `json:"emailAddress,omitempty"`
	TwitterHandle          string                 `json:"twitterHandle,omitempty"`
	FacebookProfile        string                 `json:"facebookProfile,omitempty"`
	LinkedinProfile        string                 `json:"linkedinProfile,omitempty"`
	Description            string                 `json:"description,omitempty"`
	DescriptionXML         string                 `json:"descriptionXML,omitempty"`
	ImageURL               string                 `json:"_imageUrl,omitempty"` // TODO this is a temporary thing - needs to be integrated into images properly
	Types                  []string               `json:"types,omitempty"`
}

type identifier struct {
	Authority       string `json:"authority"`
	IdentifierValue string `json:"identifierValue"`
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
