package manifest

type FieldType string

const (
	FieldTypeString      FieldType = "string"
	FieldTypeNumber      FieldType = "number"
	FieldTypeBoolean     FieldType = "boolean"
	FieldTypeMap         FieldType = "map"          // arbitrary key-value pairs
	FieldTypeSelect      FieldType = "select"       // single select dropdown
	FieldTypeMultiSelect FieldType = "multi-select" // multiple selection
	FieldTypeTextarea    FieldType = "textarea"     // multi-line text
	FieldTypeArray       FieldType = "array"        // array of values
	FieldTypeObject      FieldType = "object"       // structured object with nested fields
	FieldTypeResource    FieldType = "resource"     // integration resource reference
)

type Validation struct {
	Min        *int32 `json:"min,omitempty"`        // Minimum value/length
	Max        *int32 `json:"max,omitempty"`        // Maximum value/length
	Pattern    string `json:"pattern,omitempty"`    // Regex pattern
	MinLength  *int32 `json:"minLength,omitempty"`  // Minimum string length
	MaxLength  *int32 `json:"maxLength,omitempty"`  // Maximum string length
	CustomRule string `json:"customRule,omitempty"` // Custom validation identifier
}

type Option struct {
	Value       string `json:"value"`
	Label       string `json:"label"`
	Description string `json:"description,omitempty"`
}

type FieldManifest struct {
	Name         string          `json:"name"`
	DisplayName  string          `json:"displayName"`
	Type         FieldType       `json:"type"`
	Required     bool            `json:"required"`
	Description  string          `json:"description,omitempty"`
	Options      []Option        `json:"options,omitempty"`
	ResourceType string          `json:"resourceType,omitempty"` // Resource type for FieldTypeResource
	Placeholder  string          `json:"placeholder,omitempty"`
	Default      any             `json:"default,omitempty"`
	Validation   *Validation     `json:"validation,omitempty"`
	DependsOn    string          `json:"dependsOn,omitempty"`
	Hidden       bool            `json:"hidden,omitempty"`
	Fields       []FieldManifest `json:"fields,omitempty"`
	ItemType     FieldType       `json:"itemType,omitempty"`
}

type TypeManifest struct {
	Type            string          `json:"type"`
	DisplayName     string          `json:"displayName"`
	Description     string          `json:"description"`
	Category        string          `json:"category"`
	IntegrationType string          `json:"integrationType,omitempty"`
	Icon            string          `json:"icon,omitempty"`
	Fields          []FieldManifest `json:"fields"`
}

type ManifestProvider interface {
	Manifest() *TypeManifest
}

type Registry struct {
	ExecutorManifests    map[string]*TypeManifest
	EventSourceManifests map[string]*TypeManifest
}

func NewRegistry() *Registry {
	return &Registry{
		ExecutorManifests:    make(map[string]*TypeManifest),
		EventSourceManifests: make(map[string]*TypeManifest),
	}
}

func (r *Registry) RegisterExecutor(manifest *TypeManifest) {
	if manifest != nil {
		r.ExecutorManifests[manifest.Type] = manifest
	}
}

func (r *Registry) RegisterEventSource(manifest *TypeManifest) {
	if manifest != nil {
		r.EventSourceManifests[manifest.Type] = manifest
	}
}

func (r *Registry) GetExecutorManifest(executorType string) *TypeManifest {
	return r.ExecutorManifests[executorType]
}

func (r *Registry) GetEventSourceManifest(sourceType string) *TypeManifest {
	return r.EventSourceManifests[sourceType]
}

func (r *Registry) GetAllExecutorManifests() []*TypeManifest {
	manifests := make([]*TypeManifest, 0, len(r.ExecutorManifests))
	for _, m := range r.ExecutorManifests {
		manifests = append(manifests, m)
	}
	return manifests
}

func (r *Registry) GetAllEventSourceManifests() []*TypeManifest {
	manifests := make([]*TypeManifest, 0, len(r.EventSourceManifests))
	for _, m := range r.EventSourceManifests {
		manifests = append(manifests, m)
	}
	return manifests
}
