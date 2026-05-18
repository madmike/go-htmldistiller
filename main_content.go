package htmldistiller

import (
	"strings"

	"github.com/PuerkitoBio/goquery"
	"github.com/madmike/go-infra/telemetry"
	"golang.org/x/net/html"
)

var prioritySelectors = []string{
	"main", "article", "[role='main']",
	"#content", "#main", ".content", ".main",
	"#allrecords", "#SITE_PAGES", ".sqs-layout",
	".entry-content", ".post-content", ".page-content",
	".post-body", ".post", "#container", ".container",
}

func findMainContent(doc *goquery.Document, minDensity float64, logger telemetry.Logger) *goquery.Selection {
	for _, sel := range prioritySelectors {
		s := doc.Find(sel)
		if s.Length() == 0 {
			continue
		}
		// Deduplicate nested matches
		final := s.FilterFunction(func(i int, node *goquery.Selection) bool {
			nested := false
			node.Parents().EachWithBreak(func(j int, p *goquery.Selection) bool {
				if s.Contains(p.Get(0)) {
					nested = true
					return false
				}
				return true
			})
			return !nested
		})
		logger.Debug("main content via selector", telemetry.String("sel", sel))
		return final
	}

	type candidate struct {
		sel     *goquery.Selection
		density float64
	}
	var candidates []candidate

	doc.Find("div, section, article").Each(func(i int, s *goquery.Selection) {
		d := calcDensity(s)
		if d >= minDensity && linkDensity(s) < 0.6 {
			candidates = append(candidates, candidate{s, d})
		}
	})

	if len(candidates) > 0 {
		var final *goquery.Selection
		for _, c := range candidates {
			nested := false
			c.sel.Parents().EachWithBreak(func(j int, p *goquery.Selection) bool {
				for _, o := range candidates {
					if p.Get(0) == o.sel.Get(0) {
						nested = true
						return false
					}
				}
				return true
			})
			if !nested {
				if final == nil {
					final = c.sel
				} else {
					final = final.AddSelection(c.sel)
				}
			}
		}
		if final != nil && final.Length() > 0 {
			logger.Debug("main content via density")
			return final
		}
	}

	logger.Debug("main content: body fallback")
	return doc.Find("body")
}

func calcDensity(s *goquery.Selection) float64 {
	var text, tags int
	s.Each(func(i int, sel *goquery.Selection) {
		for _, n := range sel.Nodes {
			walkCount(n, &text, &tags)
		}
	})
	if tags == 0 {
		return 0
	}
	return float64(text) / float64(tags)
}

func walkCount(n *html.Node, text, tags *int) {
	if n.Type == html.TextNode {
		*text += len(strings.TrimSpace(n.Data))
	} else if n.Type == html.ElementNode {
		*tags++
	}
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		walkCount(c, text, tags)
	}
}

func linkDensity(s *goquery.Selection) float64 {
	var link, total int
	s.Each(func(i int, sel *goquery.Selection) {
		for _, n := range sel.Nodes {
			walkLinks(n, &link, &total)
		}
	})
	if total == 0 {
		return 0
	}
	return float64(link) / float64(total)
}

func walkLinks(n *html.Node, link, total *int) {
	if n.Type == html.TextNode {
		l := len(strings.TrimSpace(n.Data))
		*total += l
		if insideLink(n) {
			*link += l
		}
	}
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		walkLinks(c, link, total)
	}
}

func insideLink(n *html.Node) bool {
	for cur := n.Parent; cur != nil; cur = cur.Parent {
		if cur.Type == html.ElementNode && strings.ToLower(cur.Data) == "a" {
			return true
		}
	}
	return false
}
