package docs

// DocumentInfo is the metadata-only projection used by the docs index endpoint.
type DocumentInfo struct {
	Title       string                 `json:"title"`
	Description string                 `json:"description,omitempty"`
	Type        string                 `json:"type"` // "markdown" or "mdx"
	Path        string                 `json:"path"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
}

// DocumentSource is the raw file body plus parsed metadata. The MDX/Markdown
// compilation happens in the React client, so the server returns the source as-is.
type DocumentSource struct {
	Source   string       `json:"source"`
	Type     string       `json:"type"`
	Metadata DocumentInfo `json:"metadata"`
}
