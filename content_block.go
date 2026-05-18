package htmldistiller

import (
	"fmt"
	"strings"

	"github.com/PuerkitoBio/goquery"
	"golang.org/x/net/html"
)

func extractMeta(doc *goquery.Document) Meta {
	m := Meta{Keywords: []string{}}
	m.Title = strings.TrimSpace(doc.Find("title").Text())
	if lang, ok := doc.Find("html").Attr("lang"); ok {
		m.Language = lang
	}
	doc.Find("meta").Each(func(i int, s *goquery.Selection) {
		name, _ := s.Attr("name")
		prop, _ := s.Attr("property")
		content, _ := s.Attr("content")
		key := name
		if key == "" {
			key = prop
		}
		if key == "" || content == "" {
			return
		}
		switch strings.ToLower(key) {
		case "description", "og:description":
			if m.Description == "" {
				m.Description = content
			}
		case "author":
			m.Author = content
		case "keywords":
			parts := strings.Split(content, ",")
			for i := range parts {
				parts[i] = strings.TrimSpace(parts[i])
			}
			m.Keywords = parts
		}
	})
	return m
}

func extractNavigation(doc *goquery.Document, max int) []Link {
	var links []Link
	seen := make(map[string]bool)

	selectors := []string{
		"nav a", "header a", "footer a",
		"[role='navigation'] a", "[role='menubar'] a",
		".t-menu__container a", ".t199__menu a", ".t228 a", ".t450 a",
		"#SITE_HEADER a", "#SITE_FOOTER a",
		"[class*='nav'] a", "[class*='menu'] a",
		"[id*='nav'] a", "[id*='menu'] a",
		"li a", ".dropdown a",
	}

	for _, sel := range selectors {
		doc.Find(sel).Each(func(i int, s *goquery.Selection) {
			if len(links) >= max {
				return
			}
			var sb strings.Builder
			extractText(s.Get(0), &sb)
			text := strings.TrimSpace(sb.String())
			href, _ := s.Attr("href")
			if text == "" || len(text) > 150 || len(text) < 2 || href == "" {
				return
			}
			key := text + "|" + href
			if !seen[key] {
				seen[key] = true
				links = append(links, Link{Text: text, URL: href})
			}
		})
	}
	return links
}

func formatContextBlock(sel *goquery.Selection, maxLen int) string {
	if sel == nil || sel.Length() == 0 {
		return ""
	}
	var sb strings.Builder
	for _, n := range sel.Nodes {
		extractStructuredText(n, &sb, 0)
	}
	content := collapseNewlines(sb.String())
	if len(content) > maxLen {
		content = content[:maxLen] + "..."
	}
	return strings.TrimSpace(content)
}

func extractText(n *html.Node, sb *strings.Builder) {
	if n == nil {
		return
	}
	if n.Type == html.TextNode {
		t := normalizeWS(n.Data)
		if t != "" {
			if sb.Len() > 0 {
				sb.WriteByte(' ')
			}
			sb.WriteString(t)
		}
	}
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		extractText(c, sb)
	}
}

func extractStructuredText(n *html.Node, sb *strings.Builder, depth int) {
	if n.Type == html.TextNode {
		t := normalizeWS(n.Data)
		if t != "" {
			curr := sb.String()
			if curr != "" && !strings.HasSuffix(curr, " ") && !strings.HasSuffix(curr, "\n") {
				sb.WriteByte(' ')
			}
			sb.WriteString(t)
		}
		return
	}

	if n.Type != html.ElementNode {
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			extractStructuredText(c, sb, depth)
		}
		return
	}

	tag := strings.ToLower(n.Data)

	if isControl(tag) {
		token := controlToken(n, tag)
		if token != "" {
			if curr := sb.String(); curr != "" && !strings.HasSuffix(curr, "\n") {
				sb.WriteByte('\n')
			}
			sb.WriteString(token)
			sb.WriteByte('\n')
		}
		if tag == "input" || tag == "textarea" || tag == "button" {
			return
		}
	}

	if tag == "a" {
		href := nodeAttr(n, "href")
		txt := nodeText(n)
		if href != "" && txt != "" && len([]rune(txt)) <= 100 {
			if curr := sb.String(); curr != "" && !strings.HasSuffix(curr, " ") && !strings.HasSuffix(curr, "\n") {
				sb.WriteByte(' ')
			}
			fmt.Fprintf(sb, "[%s](%s)", txt, href)
			return
		}
	}

	if isHeading(tag) {
		sb.WriteString("\n\n")
		sb.WriteString(strings.Repeat("#", int(tag[1]-'0')) + " ")
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			extractStructuredText(c, sb, depth+1)
		}
		sb.WriteByte('\n')
		return
	}

	if tag == "li" {
		sb.WriteString("\n- ")
	}

	block := isBlock(tag)
	if block {
		if curr := sb.String(); curr != "" && !strings.HasSuffix(curr, "\n") {
			sb.WriteByte('\n')
		}
	}

	for c := n.FirstChild; c != nil; c = c.NextSibling {
		extractStructuredText(c, sb, depth+1)
	}

	if block || isHeading(tag) {
		if curr := sb.String(); curr != "" && !strings.HasSuffix(curr, "\n") {
			sb.WriteByte('\n')
		}
	}
}

func isHeading(tag string) bool {
	return len(tag) == 2 && tag[0] == 'h' && tag[1] >= '1' && tag[1] <= '6'
}

func isBlock(tag string) bool {
	switch tag {
	case "div", "section", "article", "p", "ul", "ol", "li", "table", "tr", "br":
		return true
	}
	return false
}

func isControl(tag string) bool {
	switch tag {
	case "input", "select", "textarea", "button", "option", "label":
		return true
	}
	return false
}

func controlToken(n *html.Node, tag string) string {
	switch tag {
	case "input":
		t := firstNonEmpty(nodeAttr(n, "type"), "text")
		name := firstNonEmpty(nodeAttr(n, "name"), nodeAttr(n, "id"), nodeAttr(n, "aria-label"), nodeAttr(n, "placeholder"))
		if name == "" {
			return fmt.Sprintf("<input type=%s>", t)
		}
		return fmt.Sprintf("<input type=%s name=%s>", t, cleanInline(name))
	case "select":
		name := firstNonEmpty(nodeAttr(n, "name"), nodeAttr(n, "id"), nodeAttr(n, "aria-label"))
		if name == "" {
			return "<select>"
		}
		return fmt.Sprintf("<select name=%s>", cleanInline(name))
	case "textarea":
		name := firstNonEmpty(nodeAttr(n, "name"), nodeAttr(n, "id"), nodeAttr(n, "aria-label"))
		if name == "" {
			return "<textarea>"
		}
		return fmt.Sprintf("<textarea name=%s>", cleanInline(name))
	case "button":
		if txt := nodeText(n); txt != "" {
			return fmt.Sprintf("<button>%s</button>", cleanInline(txt))
		}
		return "<button>"
	case "label":
		if txt := nodeText(n); txt != "" {
			return fmt.Sprintf("<label>%s</label>", cleanInline(txt))
		}
		return "<label>"
	case "option":
		if txt := nodeText(n); txt != "" {
			return fmt.Sprintf("<option>%s</option>", cleanInline(txt))
		}
		return ""
	}
	return ""
}

func nodeAttr(n *html.Node, key string) string {
	for _, a := range n.Attr {
		if strings.EqualFold(a.Key, key) {
			return strings.TrimSpace(a.Val)
		}
	}
	return ""
}

func nodeText(n *html.Node) string {
	var sb strings.Builder
	var walk func(*html.Node)
	walk = func(cur *html.Node) {
		if cur == nil {
			return
		}
		if cur.Type == html.TextNode {
			t := normalizeWS(cur.Data)
			if t != "" {
				if sb.Len() > 0 {
					sb.WriteByte(' ')
				}
				sb.WriteString(t)
			}
		}
		for c := cur.FirstChild; c != nil; c = c.NextSibling {
			walk(c)
		}
	}
	walk(n)
	return strings.TrimSpace(sb.String())
}

func cleanInline(s string) string {
	return strings.TrimSpace(strings.Join(strings.Fields(s), " "))
}

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if strings.TrimSpace(v) != "" {
			return v
		}
	}
	return ""
}

func normalizeWS(s string) string {
	return strings.Join(strings.Fields(s), " ")
}

func collapseNewlines(text string) string {
	var sb strings.Builder
	empty := 0
	for _, line := range strings.Split(text, "\n") {
		t := strings.TrimSpace(line)
		if t == "" {
			empty++
			if empty <= 1 {
				sb.WriteByte('\n')
			}
		} else {
			empty = 0
			sb.WriteString(t)
			sb.WriteByte('\n')
		}
	}
	return strings.TrimSpace(sb.String())
}
