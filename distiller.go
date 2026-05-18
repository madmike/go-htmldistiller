package htmldistiller

import (
	"bytes"
	"context"
	"fmt"
	"io"

	"github.com/PuerkitoBio/goquery"
	"github.com/madmike/go-infra/telemetry"
)

// Distiller orchestrates the HTML extraction pipeline.
type Distiller struct {
	cfg    Config
	logger telemetry.Logger
}

// New creates a new Distiller. Zero-value Config fields are replaced with defaults.
func New(cfg Config, logger telemetry.Logger) *Distiller {
	cfg.Defaults()
	return &Distiller{
		cfg:    cfg,
		logger: logger.WithModule("HTMLDistiller"),
	}
}

// Distill processes HTML bytes and returns structured content.
// Use RoleContent for simple ingestion; RoleDiscovery for full crawl metadata.
func (d *Distiller) Distill(ctx context.Context, html []byte, url string, role ExtractionRole) (*Result, error) {
	d.logger.Info("distilling",
		telemetry.String("url", url),
		telemetry.String("role", string(role)),
		telemetry.Int("html_bytes", len(html)))

	doc, err := goquery.NewDocumentFromReader(io.Reader(bytes.NewReader(html)))
	if err != nil {
		return nil, fmt.Errorf("parse html: %w", err)
	}

	// JSON-LD first (highest-signal structured data)
	jsonLD, hasJSONLD := extractJSONLD(doc, d.logger)

	// Metadata
	meta := extractMeta(doc)

	// Navigation links (only for discovery/classification roles)
	var links []Link
	if role == RoleDiscovery || role == RoleClassification || role == RoleEntity {
		links = extractNavigation(doc, d.cfg.MaxNavItems)
	}

	// Contact info BEFORE noise stripping (may live in headers/footers)
	var contacts ContactInfo
	if role != RoleContent {
		contacts = extractContacts(doc)
	}

	// Noise stripping
	stripNoise(doc, role, d.cfg.AggressivePruning, d.logger)

	// Main content detection
	mainContent := findMainContent(doc, d.cfg.MinTextDensity, d.logger)
	content := formatContextBlock(mainContent, d.cfg.MaxContentLength)

	method := "readability"
	if hasJSONLD {
		method = "json-ld"
	}

	d.logger.Info("distillation done",
		telemetry.String("url", url),
		telemetry.String("method", method),
		telemetry.Int("content_len", len(content)),
		telemetry.Int("link_count", len(links)))

	return &Result{
		Meta:     meta,
		Links:    links,
		Contacts: contacts,
		Content:  content,
		JSONLD:   jsonLD,
		Method:   method,
		Role:     role,
	}, nil
}

// GetJSONLDTypeFromMap extracts the @type field from a JSON-LD map.
func GetJSONLDTypeFromMap(data map[string]any) string {
	return getJSONLDType(data)
}
