package people

import "sort"

type SortedIdentifiers []identifier

type person struct {
	UUID           string       `json:"uuid"`
	BirthYear      int          `json:"birthYear,omitempty"`
	Identifiers    []identifier `json:"identifiers,omitempty"`
	Name           string       `json:"name,omitempty"`
	Salutation     string       `json:"salutation,omitempty"`
	Aliases        []string     `json:"aliases,omitempty"`
	EmailAddress   string       `json:"emailAddress,omitempty"`
	TwitterHandle  string       `json:"twitterHandler,omitempty"`
	Description    string       `json:"description,omitempty"`
	DescriptionXML string       `json:"descriptionXML,omitempty"`
	ImageURL       string       `json:"_imageUrl"` // TODO this is a temporary thing - needs to be integrated into images properly
}

type identifier struct {
	Authority       string `json:"authority"`
	IdentifierValue string `json:"identifierValue"`
}

func sortIdentifiers(iden []identifier) {
	sort.Sort(SortedIdentifiers(iden))
}

// these three are the implementation of sort interface
func (si SortedIdentifiers) Len() int {
	return len(si)
}

func (si SortedIdentifiers) Swap(i, j int) {
	si[i], si[j] = si[j], si[i]
}

func (si SortedIdentifiers) Less(i, j int) bool {

	if si[i].Authority == si[j].Authority {
		return si[i].IdentifierValue < si[j].IdentifierValue
	} else {
		return si[i].Authority < si[j].Authority
	}
}
