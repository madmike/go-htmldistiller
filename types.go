// Package htmldistiller provides HTML→clean-text extraction:
// main content detection (Readability heuristic), noise stripping, JSON-LD parsing,
// contact extraction, and structured markdown output.
// Used by crawler-service (full pipeline) and ingestion-service (single-page extraction).
package htmldistiller

// ExtractionRole controls the aggressiveness of noise stripping.
type ExtractionRole string

const (
	// RoleContent extracts main content only, suitable for ingestion.
	RoleContent ExtractionRole = "content"
	// RoleDiscovery is a full extraction for site crawling (navigation + contacts).
	RoleDiscovery ExtractionRole = "site_discovery"
	// RoleClassification strips aggressively for business classification.
	RoleClassification ExtractionRole = "business_classification"
	// RoleEntity strips aggressively for entity extraction.
	RoleEntity ExtractionRole = "entity_extraction"
)

// Config controls distillation behaviour.
type Config struct {
	MinTextDensity   float64 // minimum text density to accept a block (default 10.0)
	MaxContentLength int     // maximum content length in bytes (default 50000)
	MaxNavItems      int     // maximum navigation links to extract (default 100)
	AggressivePruning bool   // remove menus/sidebars/sliders
}

// Defaults fills in zero-value config fields with sensible defaults.
func (c *Config) Defaults() {
	if c.MinTextDensity == 0 {
		c.MinTextDensity = 10.0
	}
	if c.MaxContentLength == 0 {
		c.MaxContentLength = 50_000
	}
	if c.MaxNavItems == 0 {
		c.MaxNavItems = 100
	}
}

// Result is the structured output of extraction.
type Result struct {
	Meta    Meta
	Links   []Link
	Contacts ContactInfo
	Content string
	JSONLD  map[string]any
	Method  string         // "json-ld" | "readability" | "fallback"
	Role    ExtractionRole
}

// Meta holds document-level metadata extracted from <head>.
type Meta struct {
	Title       string
	Description string
	Language    string
	Author      string
	Keywords    []string
}

// Link is a navigation or in-content link.
type Link struct {
	Text string
	URL  string
}

// ContactInfo holds all contact data extracted from the page.
type ContactInfo struct {
	Emails      []string          `json:"emails"`
	Phones      []string          `json:"phones"`
	SocialLinks map[string]string `json:"social_links"`
	Addresses   []Address         `json:"addresses"`
}

// Address is a physical address extracted from structured markup or text.
type Address struct {
	Text       string `json:"text"`
	Normalized string `json:"normalized"`
	IsMain     bool   `json:"is_main"`
}
