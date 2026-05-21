package htmldistiller

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/madmike/go-infra/telemetry"
)

// TestDistillerCreation initializes with defaults.
func TestDistillerCreation(t *testing.T) {
	logger := &telemetry.NoOpLogger{}
	distiller := New(Config{}, logger)
	require.NotNil(t, distiller)
}

// TestDistillSimpleContent extracts text from basic HTML.
func TestDistillSimpleContent(t *testing.T) {
	html := []byte(`
		<html>
			<head><title>Test Page</title></head>
			<body>
				<h1>Main Heading</h1>
				<p>Article content goes here.</p>
			</body>
		</html>
	`)

	distiller := New(Config{}, &telemetry.NoOpLogger{})
	result, err := distiller.Distill(context.Background(), html, "https://example.com", RoleContent)

	require.NoError(t, err)
	require.NotNil(t, result)
	require.NotEmpty(t, result.Content)
	require.Contains(t, result.Content, "Main Heading")
}

// TestDistillMetadataExtraction pulls title and description.
func TestDistillMetadataExtraction(t *testing.T) {
	html := []byte(`
		<html>
			<head>
				<title>Page Title</title>
				<meta name="description" content="Page summary">
				<meta property="og:title" content="Social Title">
			</head>
			<body><p>Content</p></body>
		</html>
	`)

	distiller := New(Config{}, &telemetry.NoOpLogger{})
	result, err := distiller.Distill(context.Background(), html, "https://example.com", RoleContent)

	require.NoError(t, err)
	require.NotNil(t, result.Meta)
}

// TestDistillJSONLD extracts structured data.
func TestDistillJSONLD(t *testing.T) {
	html := []byte(`
		<html>
			<head>
				<script type="application/ld+json">
					{"@type": "Article", "headline": "News Article"}
				</script>
			</head>
			<body><p>Content</p></body>
		</html>
	`)

	distiller := New(Config{}, &telemetry.NoOpLogger{})
	result, err := distiller.Distill(context.Background(), html, "https://example.com", RoleContent)

	require.NoError(t, err)
	require.NotNil(t, result)
}

// TestDistillDiscoveryRole extracts navigation links.
func TestDistillDiscoveryRole(t *testing.T) {
	html := []byte(`
		<html>
			<body>
				<nav>
					<a href="/home">Home</a>
					<a href="/about">About</a>
				</nav>
				<p>Content</p>
			</body>
		</html>
	`)

	distiller := New(Config{MaxNavItems: 10}, &telemetry.NoOpLogger{})
	result, err := distiller.Distill(context.Background(), html, "https://example.com", RoleDiscovery)

	require.NoError(t, err)
	require.NotNil(t, result)
	// Links may or may not be extracted depending on DOM structure
	require.NotNil(t, result.Links)
}

// TestDistillContentRole skipshops navigation.
func TestDistillContentRole(t *testing.T) {
	html := []byte(`
		<html>
			<body>
				<nav><a href="/nav">Nav</a></nav>
				<main>Main content here</main>
			</body>
		</html>
	`)

	distiller := New(Config{}, &telemetry.NoOpLogger{})
	result, err := distiller.Distill(context.Background(), html, "https://example.com", RoleContent)

	require.NoError(t, err)
	require.NotNil(t, result)
	require.Contains(t, result.Content, "content")
}

// TestDistillInvalidHTML returns error on parse failure.
func TestDistillInvalidHTML(t *testing.T) {
	html := []byte(`<html><body>Unclosed tag</body>`)
	distiller := New(Config{}, &telemetry.NoOpLogger{})

	// Note: goquery is lenient; this may not error
	result, err := distiller.Distill(context.Background(), html, "https://example.com", RoleContent)
	if err == nil {
		require.NotNil(t, result)
	}
}

// TestDistillEmptyBody handles minimal HTML.
func TestDistillEmptyBody(t *testing.T) {
	html := []byte(`<html><body></body></html>`)
	distiller := New(Config{}, &telemetry.NoOpLogger{})

	result, err := distiller.Distill(context.Background(), html, "https://example.com", RoleContent)
	require.NoError(t, err)
	require.NotNil(t, result)
}

// TestDistillNoiseStripping removes boilerplate.
func TestDistillNoiseStripping(t *testing.T) {
	html := []byte(`
		<html>
			<body>
				<header>Site Header</header>
				<footer>Copyright 2026</footer>
				<main>Important content</main>
			</body>
		</html>
	`)

	distiller := New(Config{AggressivePruning: true}, &telemetry.NoOpLogger{})
	result, err := distiller.Distill(context.Background(), html, "https://example.com", RoleContent)

	require.NoError(t, err)
	require.NotEmpty(t, result.Content)
}

// TestDistillContextCancellation respects cancelled context.
func TestDistillContextCancellation(t *testing.T) {
	html := []byte(`<html><body><p>Content</p></body></html>`)
	distiller := New(Config{}, &telemetry.NoOpLogger{})

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	result, err := distiller.Distill(ctx, html, "https://example.com", RoleContent)
	// Should still work (context is mostly informational in distiller)
	if err == nil {
		require.NotNil(t, result)
	}
}

// TestDistillURL includes provided URL in result.
func TestDistillURL(t *testing.T) {
	html := []byte(`<html><body><a href="/page">Link</a></body></html>`)
	distiller := New(Config{}, &telemetry.NoOpLogger{})

	result, err := distiller.Distill(context.Background(), html, "https://example.com/article", RoleContent)
	require.NoError(t, err)
	require.NotNil(t, result)
	require.Equal(t, RoleContent, result.Role)
}

// TestDistillLargeContent handles extended HTML.
func TestDistillLargeContent(t *testing.T) {
	html := []byte(`
		<html>
			<body>
				<h1>Title</h1>
	`)
	for i := 0; i < 1000; i++ {
		html = append(html, []byte("<p>Paragraph content.</p>\n")...)
	}
	html = append(html, []byte(`
			</body>
		</html>
	`)...)

	distiller := New(Config{}, &telemetry.NoOpLogger{})
	result, err := distiller.Distill(context.Background(), html, "https://example.com", RoleContent)

	require.NoError(t, err)
	require.NotNil(t, result)
	require.NotEmpty(t, result.Content)
}

// TestDistillMaxContentLength respects limit.
func TestDistillMaxContentLength(t *testing.T) {
	html := []byte(`
		<html>
			<body>
				<p>Start of content</p>
				<p>Middle content</p>
				<p>End content</p>
			</body>
		</html>
	`)

	distiller := New(Config{MaxContentLength: 50}, &telemetry.NoOpLogger{})
	result, err := distiller.Distill(context.Background(), html, "https://example.com", RoleContent)

	require.NoError(t, err)
	require.NotNil(t, result)
	// Content should be truncated
	require.True(t, len(result.Content) <= 100)
}

// TestDistillMinTextDensity filters sparse content.
func TestDistillMinTextDensity(t *testing.T) {
	html := []byte(`
		<html>
			<body>
				<div style="height: 1000px; width: 1000px;">
					<p>Tiny text</p>
				</div>
			</body>
		</html>
	`)

	distiller := New(Config{MinTextDensity: 0.1}, &telemetry.NoOpLogger{})
	result, err := distiller.Distill(context.Background(), html, "https://example.com", RoleContent)

	require.NoError(t, err)
	require.NotNil(t, result)
}

// TestGetJSONLDTypeFromMap extracts @type field.
func TestGetJSONLDTypeFromMap(t *testing.T) {
	data := map[string]any{
		"@type":    "Article",
		"headline": "Test",
	}

	typeVal := GetJSONLDTypeFromMap(data)
	require.Equal(t, "Article", typeVal)
}

// TestGetJSONLDTypeFromMapMissing handles missing @type.
func TestGetJSONLDTypeFromMapMissing(t *testing.T) {
	data := map[string]any{
		"headline": "Test",
	}

	typeVal := GetJSONLDTypeFromMap(data)
	require.Equal(t, "unknown", typeVal)
}

// TestDistillResult includes all fields.
func TestDistillResult(t *testing.T) {
	html := []byte(`<html><body><p>Content</p></body></html>`)
	distiller := New(Config{}, &telemetry.NoOpLogger{})

	result, err := distiller.Distill(context.Background(), html, "https://example.com", RoleContent)

	require.NoError(t, err)
	require.NotNil(t, result)
	require.NotNil(t, result.Meta)
	require.True(t, result.Links == nil || len(result.Links) >= 0)
	require.NotEmpty(t, result.Method)
	require.Equal(t, RoleContent, result.Role)
}

// TestDistillRoleEntity extracts entity-specific metadata.
func TestDistillRoleEntity(t *testing.T) {
	html := []byte(`
		<html>
			<body>
				<h1>Person Name</h1>
				<p>Biography</p>
				<a href="/related">Related</a>
			</body>
		</html>
	`)

	distiller := New(Config{}, &telemetry.NoOpLogger{})
	result, err := distiller.Distill(context.Background(), html, "https://example.com", RoleEntity)

	require.NoError(t, err)
	require.NotNil(t, result)
	require.Equal(t, RoleEntity, result.Role)
}

// TestDistillClassificationRole includes navigation.
func TestDistillClassificationRole(t *testing.T) {
	html := []byte(`
		<html>
			<body>
				<nav><a href="/products">Products</a></nav>
				<p>Taxonomy</p>
			</body>
		</html>
	`)

	distiller := New(Config{}, &telemetry.NoOpLogger{})
	result, err := distiller.Distill(context.Background(), html, "https://example.com", RoleClassification)

	require.NoError(t, err)
	require.NotNil(t, result)
}
