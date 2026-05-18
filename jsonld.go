package htmldistiller

import (
	"encoding/json"
	"strings"

	"github.com/PuerkitoBio/goquery"
	"github.com/madmike/go-infra/telemetry"
)

func extractJSONLD(doc *goquery.Document, logger telemetry.Logger) (map[string]any, bool) {
	var result map[string]any

	doc.Find("script[type='application/ld+json']").EachWithBreak(func(i int, s *goquery.Selection) bool {
		jsonText := strings.TrimSpace(s.Text())
		if jsonText == "" {
			return true
		}
		var data map[string]any
		if err := json.Unmarshal([]byte(jsonText), &data); err != nil {
			logger.Debug("json-ld parse failed", telemetry.Int("index", i), telemetry.Err(err))
			return true
		}
		result = data
		return false
	})

	return result, result != nil
}

func getJSONLDType(data map[string]any) string {
	if typeVal, ok := data["@type"]; ok {
		if typeStr, ok := typeVal.(string); ok {
			return typeStr
		}
	}
	return "unknown"
}
