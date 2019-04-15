package component

import (
	"encoding/json"
	"fmt"

	"github.com/pkg/errors"
)

// ContentResponse is a a content response. It contains a
// title and one or more components.
type ContentResponse struct {
	Title      []TitleComponent `json:"title,omitempty"`
	Components []Component      `json:"viewComponents"`
}

// NewContentResponse creates an instance of ContentResponse.
func NewContentResponse(title []TitleComponent) *ContentResponse {
	return &ContentResponse{
		Title: title,
	}
}

// Add adds zero or more components to a content response.
func (c *ContentResponse) Add(components ...Component) {
	c.Components = append(c.Components, components...)
}

// UnmarshalJSON unarmshals a content response from JSON.
func (c *ContentResponse) UnmarshalJSON(data []byte) error {
	stage := struct {
		Title      []TypedObject `json:"title,omitempty"`
		Components []TypedObject `json:"viewComponents,omitempty"`
	}{}

	if err := json.Unmarshal(data, &stage); err != nil {
		return err
	}

	for _, t := range stage.Title {
		title, err := getTitleByUnmarshalInterface(t.Config)
		if err != nil {
			return err
		}

		c.Title = Title(NewText(title))
	}

	for _, to := range stage.Components {
		vc, err := to.ToComponent()
		if err != nil {
			return err
		}

		c.Components = append(c.Components, vc)
	}

	return nil
}

func getTitleByUnmarshalInterface(config json.RawMessage) (string, error) {
	var objmap map[string]interface{}
	if err := json.Unmarshal(config, &objmap); err != nil {
		return "", err
	}

	if value, ok := objmap["value"].(string); ok {
		return value, nil
	}

	return "", fmt.Errorf("title does not have a value")
}

type TypedObject struct {
	Config   json.RawMessage `json:"config,omitempty"`
	Metadata Metadata        `json:"metadata,omitempty"`
}

func (to *TypedObject) ToComponent() (Component, error) {
	o, err := unmarshal(*to)
	if err != nil {
		return nil, err
	}

	vc, ok := o.(Component)
	if !ok {
		return nil, errors.Errorf("unable to convert %T to Component",
			o)
	}

	return vc, nil
}

// Metadata collects common fields describing Components
type Metadata struct {
	Type     string           `json:"type"`
	Title    []TitleComponent `json:"title,omitempty"`
	Accessor string           `json:"accessor,omitempty"`
}

// SetTitleText sets the title using text components.
func (m *Metadata) SetTitleText(parts ...string) {
	var titleComponents []TitleComponent

	for _, part := range parts {
		titleComponents = append(titleComponents, NewText(part))
	}

	m.Title = titleComponents
}

func (m *Metadata) UnmarshalJSON(data []byte) error {
	x := struct {
		Type  string        `json:"type,omitempty"`
		Title []TypedObject `json:"title,omitempty"`
	}{}

	if err := json.Unmarshal(data, &x); err != nil {
		return err
	}

	m.Type = x.Type

	for _, title := range x.Title {
		vc, err := title.ToComponent()
		if err != nil {
			return errors.Wrap(err, "unmarshaling title")
		}

		tvc, ok := vc.(TitleComponent)
		if !ok {
			return errors.New("component in title isn't a title view component")
		}

		m.Title = append(m.Title, tvc)
	}

	return nil
}

// Component is a common interface for the data representation
// of visual components as rendered by the UI.
type Component interface {
	GetMetadata() Metadata
	SetAccessor(string)
	IsEmpty() bool
	String() string
}

// TitleComponent is a view component that can be used for a title.
type TitleComponent interface {
	Component

	SupportsTitle()
}

// Title is a convenience method for creating a title.
func Title(components ...TitleComponent) []TitleComponent {
	return components
}

// TitleFromString is a convenience methods for create a title from a string.
func TitleFromString(s string) []TitleComponent {
	return Title(NewText(s))
}