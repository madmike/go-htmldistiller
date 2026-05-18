package htmldistiller

import (
	"regexp"
	"strings"

	"github.com/PuerkitoBio/goquery"
)

func extractContacts(doc *goquery.Document) ContactInfo {
	return ContactInfo{
		Emails:      extractEmails(doc),
		Phones:      extractPhones(doc),
		SocialLinks: extractSocialLinks(doc),
		Addresses:   extractAddresses(doc),
	}
}

func extractEmails(doc *goquery.Document) []string {
	seen := make(map[string]bool)
	var emails []string
	emailRE := regexp.MustCompile(`[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}`)

	doc.Find("a[href^='mailto:']").Each(func(i int, s *goquery.Selection) {
		href, _ := s.Attr("href")
		email := strings.Split(strings.TrimPrefix(href, "mailto:"), "?")[0]
		if emailRE.MatchString(email) && !seen[email] {
			emails = append(emails, email)
			seen[email] = true
		}
	})

	for _, email := range emailRE.FindAllString(doc.Find("body").Text(), -1) {
		if !seen[email] {
			emails = append(emails, email)
			seen[email] = true
		}
	}
	return emails
}

func extractPhones(doc *goquery.Document) []string {
	seen := make(map[string]bool)
	var phones []string

	doc.Find("a[href^='tel:']").Each(func(i int, s *goquery.Selection) {
		href, _ := s.Attr("href")
		if p := normalizePhone(strings.TrimPrefix(href, "tel:")); p != "" && !seen[p] {
			phones = append(phones, p)
			seen[p] = true
		}
	})

	patterns := []*regexp.Regexp{
		regexp.MustCompile(`\+\d{1,3}[\s\-\(\)]?\d{1,4}[\s\-\(\)]?\d{1,4}[\s\-]?\d{1,4}[\s\-]?\d{1,4}`),
		regexp.MustCompile(`\(\d{3}\)[\s\-]?\d{3}[\s\-]?\d{4}`),
		regexp.MustCompile(`\d{3}[\s\-]\d{3}[\s\-]\d{4}`),
	}
	body := doc.Find("body").Text()
	for _, re := range patterns {
		for _, m := range re.FindAllString(body, -1) {
			if p := normalizePhone(m); p != "" && !seen[p] {
				phones = append(phones, p)
				seen[p] = true
			}
		}
	}
	return phones
}

func normalizePhone(phone string) string {
	for _, r := range []string{" ", "-", "(", ")"} {
		phone = strings.ReplaceAll(phone, r, "")
	}
	if len(phone) < 7 {
		return ""
	}
	return phone
}

func extractSocialLinks(doc *goquery.Document) map[string]string {
	social := make(map[string]string)
	platforms := map[string][]string{
		"facebook":   {"facebook.com", "fb.com", "fb.me"},
		"instagram":  {"instagram.com", "instagr.am"},
		"twitter":    {"twitter.com", "x.com"},
		"linkedin":   {"linkedin.com"},
		"youtube":    {"youtube.com", "youtu.be"},
		"tiktok":     {"tiktok.com"},
		"pinterest":  {"pinterest.com", "pin.it"},
		"snapchat":   {"snapchat.com"},
		"whatsapp":   {"wa.me", "whatsapp.com"},
		"telegram":   {"t.me", "telegram.me"},
		"vk":         {"vk.com", "vkontakte.ru"},
		"ok":         {"ok.ru", "odnoklassniki.ru"},
		"discord":    {"discord.gg", "discord.com"},
		"reddit":     {"reddit.com"},
		"github":     {"github.com"},
		"twitch":     {"twitch.tv"},
	}

	doc.Find("a[href]").Each(func(i int, s *goquery.Selection) {
		href, _ := s.Attr("href")
		href = strings.ToLower(href)
		for platform, domains := range platforms {
			if _, exists := social[platform]; exists {
				continue
			}
			for _, domain := range domains {
				if strings.Contains(href, domain) {
					social[platform] = href
					break
				}
			}
		}
	})
	return social
}

func extractAddresses(doc *goquery.Document) []Address {
	var addrs []Address
	seen := make(map[string]bool)

	collect := func(s *goquery.Selection) {
		txt := strings.TrimSpace(s.Text())
		if txt != "" && !seen[txt] {
			addrs = append(addrs, Address{Text: txt})
			seen[txt] = true
		}
	}

	doc.Find("address").Each(func(i int, s *goquery.Selection) { collect(s) })
	doc.Find("[itemprop='address']").Each(func(i int, s *goquery.Selection) { collect(s) })
	return addrs
}
