package api

// CredentialRefresh is stored only inside the encrypted task payload. Stable
// resource identity and recipient are checked again immediately before delivery.
type CredentialRefresh struct {
	Version        int               `json:"version"`
	Kind           string            `json:"kind"`
	OrganizationID string            `json:"organization_id"`
	ResourceID     string            `json:"resource_id"`
	Recipient      string            `json:"recipient"`
	Scope          map[string]string `json:"scope,omitempty"`
	Fields         []CredentialField `json:"fields"`
	Current        string            `json:"-"`
}

type CredentialField struct {
	Path         []string `json:"path"`
	Original     string   `json:"original"`
	Encoding     string   `json:"encoding,omitempty"`
	DocumentPath []string `json:"document_path,omitempty"`
}

// CredentialRefreshRecorder is optional so legacy task producers keep working.
type CredentialRefreshRecorder interface {
	RecordCredentialRefresh(CredentialRefresh)
}
