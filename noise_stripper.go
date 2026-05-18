package htmldistiller

import (
	"github.com/PuerkitoBio/goquery"
	"github.com/madmike/go-infra/telemetry"
)

func stripNoise(doc *goquery.Document, role ExtractionRole, aggressive bool, logger telemetry.Logger) {
	for _, tag := range []string{"script", "style", "noscript", "iframe", "canvas", "svg"} {
		doc.Find(tag).Remove()
	}

	for _, sel := range []string{
		"header", "footer", "nav", "aside",
		"[id*='header']", "[id*='footer']", "[id*='nav']",
		"[class*='header']", "[class*='footer']", "[class*='nav']",
		"#SITE_HEADER", "#SITE_FOOTER",
	} {
		doc.Find(sel).Remove()
	}

	for _, sel := range []string{
		"[class*='modal']", "[id*='modal']",
		"[class*='popup']", "[id*='popup']",
		"[class*='cookie']", "[id*='cookie']",
		"[class*='banner']", "[class*='overlay']",
		"[class*='consent']", "[class*='widget']", "[id*='widget']",
		"[class*='fixed']", "[class*='sticky']",
		"[class*='z-99']", "[class*='z-100']",
		".hidden", ".invisible",
		"[style*='display: none']", "[style*='display:none']",
		"[style*='visibility: hidden']",
		".t-records__modals", ".t-popup", ".t-newsletter",
		".t-sociallinks", ".t-footer", ".t-header", ".t-menu__container",
	} {
		doc.Find(sel).Remove()
	}

	if role == RoleEntity || role == RoleClassification || aggressive {
		for _, sel := range []string{
			"[id*='menu']", "[id*='sidebar']",
			"[class*='menu']", "[class*='sidebar']",
			"[class*='slider']", "[class*='carousel']",
			"[class*='social']", "[class*='contact-info']",
			"button", "[role='button']", ".controls", ".arrows",
			"[data-record-type='257']", "[data-record-type='456']",
			"[data-record-type='454']", "[data-record-type='450']",
			"[data-record-type='393']", "[data-record-type='204']",
			"[data-record-type='125']", "[data-record-type='230']",
			"[data-record-type='240']", "[data-record-type='312']",
			"[data-record-type='331']", "[data-record-type='381']",
			"[data-record-type='389']", "[data-record-type='410']",
			"[data-record-type='412']", "[data-record-type='437']",
			"[data-record-type='492']", "[data-record-type='532']",
		} {
			doc.Find(sel).Remove()
		}
	}

	logger.Debug("noise stripped",
		telemetry.String("role", string(role)),
		telemetry.Bool("aggressive", aggressive))
}
