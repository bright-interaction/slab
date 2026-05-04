package builder

// JSON-LD generation for Collection items (Sprint 4, 2026-05-04).
//
// Drives schema.org structured data per item from the Collection's
// settings.schema_org_type. Field-name-to-property mapping is
// deliberately conservative: the agent or admin uses the conventional
// names (headline, author, datePublished for Article; name, price,
// brand for Product) and we map them. Unmapped fields fall through
// quietly so the JSON-LD stays valid.
//
// Supported types: Article, BlogPosting, Product, Person, Event,
// JobPosting, Recipe, FAQPage, Review. Any other value emits a Thing
// with name + description so the page still has *some* structured
// data rather than none.

import (
	"encoding/json"
	"strings"

	"github.com/brightinteraction/atomicsite/internal/store"
)

// buildJSONLD produces the JSON-LD body to embed in a
// <script type="application/ld+json"> on a Collection item page.
// Returns "" when the Collection has no schema_org_type (caller skips
// the script tag in that case).
func buildJSONLD(c store.Collection, fields []collectionFieldDef, item store.CollectionItem, data map[string]any, mediaByID map[string]store.Medium, site store.Site) string {
	var cs collectionSettings
	_ = json.Unmarshal([]byte(c.SettingsJson), &cs)
	t := strings.TrimSpace(cs.SchemaOrgType)
	if t == "" {
		return ""
	}

	out := map[string]any{
		"@context": "https://schema.org",
		"@type":    t,
		"name":     item.Title,
	}
	loc := normalizeLocale(item.Locale, site.Lang)
	rel := "/" + collectionItemPath(c.Slug, item.Slug, loc, site.Lang) + "/"
	if u := absoluteURL(site, rel); u != "" {
		out["url"] = u
	}

	switch t {
	case "Article", "BlogPosting", "NewsArticle":
		mapArticle(out, data, fields, mediaByID, site)
	case "Product":
		mapProduct(out, data, fields, mediaByID)
	case "Person":
		mapPerson(out, data, fields, mediaByID)
	case "Event":
		mapEvent(out, data, fields, mediaByID)
	case "JobPosting":
		mapJobPosting(out, data, fields, site)
	case "Recipe":
		mapRecipe(out, data, fields, mediaByID)
	case "FAQPage":
		mapFAQPage(out, data, fields)
	case "Review":
		mapReview(out, data, fields)
	default:
		// Generic Thing: name + description.
		if d := stringField(data, "description", "summary", "lead"); d != "" {
			out["description"] = d
		}
	}

	bytes, err := json.Marshal(out)
	if err != nil {
		return ""
	}
	return string(bytes)
}

func mapArticle(out, data map[string]any, fields []collectionFieldDef, mediaByID map[string]store.Medium, site store.Site) {
	if h := stringField(data, "headline", "title", "name"); h != "" {
		out["headline"] = h
	}
	if d := stringField(data, "description", "summary", "lead"); d != "" {
		out["description"] = d
	}
	if a := stringField(data, "author", "author_name", "byline"); a != "" {
		out["author"] = map[string]any{"@type": "Person", "name": a}
	}
	if dp := stringField(data, "date_published", "datePublished", "published_at"); dp != "" {
		out["datePublished"] = dp
	}
	if dm := stringField(data, "date_modified", "dateModified", "updated_at"); dm != "" {
		out["dateModified"] = dm
	}
	if img := mediaURLFromField(data, mediaByID, site, "image", "image_id", "cover", "featured_image"); img != "" {
		out["image"] = img
	}
	if site.Name != "" {
		out["publisher"] = map[string]any{"@type": "Organization", "name": site.Name}
	}
}

func mapProduct(out, data map[string]any, fields []collectionFieldDef, mediaByID map[string]store.Medium) {
	if d := stringField(data, "description", "summary"); d != "" {
		out["description"] = d
	}
	if b := stringField(data, "brand"); b != "" {
		out["brand"] = map[string]any{"@type": "Brand", "name": b}
	}
	if img := mediaURLFromField(data, mediaByID, store.Site{}, "image", "image_id", "main_image"); img != "" {
		out["image"] = img
	}
	if sku := stringField(data, "sku"); sku != "" {
		out["sku"] = sku
	}
	if price := numberField(data, "price"); price != 0 {
		offer := map[string]any{
			"@type":         "Offer",
			"price":         price,
			"priceCurrency": stringFieldOr(data, "EUR", "currency", "price_currency"),
			"availability":  "https://schema.org/InStock",
		}
		out["offers"] = offer
	}
}

func mapPerson(out, data map[string]any, fields []collectionFieldDef, mediaByID map[string]store.Medium) {
	if jt := stringField(data, "job_title", "jobTitle", "title", "role"); jt != "" {
		out["jobTitle"] = jt
	}
	if email := stringField(data, "email"); email != "" {
		out["email"] = email
	}
	if img := mediaURLFromField(data, mediaByID, store.Site{}, "image", "image_id", "photo", "headshot"); img != "" {
		out["image"] = img
	}
	if u := stringField(data, "url", "website", "linkedin"); u != "" {
		out["sameAs"] = []string{u}
	}
}

func mapEvent(out, data map[string]any, fields []collectionFieldDef, mediaByID map[string]store.Medium) {
	if sd := stringField(data, "start_date", "startDate", "starts_at"); sd != "" {
		out["startDate"] = sd
	}
	if ed := stringField(data, "end_date", "endDate", "ends_at"); ed != "" {
		out["endDate"] = ed
	}
	if loc := stringField(data, "location", "venue"); loc != "" {
		out["location"] = map[string]any{"@type": "Place", "name": loc}
	}
	if img := mediaURLFromField(data, mediaByID, store.Site{}, "image", "image_id", "cover"); img != "" {
		out["image"] = img
	}
}

func mapJobPosting(out, data map[string]any, fields []collectionFieldDef, site store.Site) {
	if t := stringField(data, "title", "job_title"); t != "" {
		out["title"] = t
	}
	if d := stringField(data, "description", "summary"); d != "" {
		out["description"] = d
	}
	if dp := stringField(data, "date_posted", "datePosted", "published_at"); dp != "" {
		out["datePosted"] = dp
	}
	if loc := stringField(data, "location", "city"); loc != "" {
		out["jobLocation"] = map[string]any{"@type": "Place", "address": loc}
	}
	if site.Name != "" {
		out["hiringOrganization"] = map[string]any{"@type": "Organization", "name": site.Name}
	}
}

func mapRecipe(out, data map[string]any, fields []collectionFieldDef, mediaByID map[string]store.Medium) {
	if img := mediaURLFromField(data, mediaByID, store.Site{}, "image", "image_id", "photo"); img != "" {
		out["image"] = img
	}
	if ing, ok := data["ingredients"].([]any); ok {
		out["recipeIngredient"] = ing
	}
	if inst, ok := data["instructions"].([]any); ok {
		out["recipeInstructions"] = inst
	}
}

func mapFAQPage(out, data map[string]any, fields []collectionFieldDef) {
	q := stringField(data, "question")
	a := stringField(data, "answer")
	if q == "" || a == "" {
		return
	}
	out["mainEntity"] = []any{
		map[string]any{
			"@type": "Question",
			"name":  q,
			"acceptedAnswer": map[string]any{
				"@type": "Answer",
				"text":  a,
			},
		},
	}
}

func mapReview(out, data map[string]any, fields []collectionFieldDef) {
	if r := stringField(data, "reviewed_item", "item_name", "subject"); r != "" {
		out["itemReviewed"] = map[string]any{"@type": "Thing", "name": r}
	}
	if rating := numberField(data, "rating", "score"); rating != 0 {
		out["reviewRating"] = map[string]any{
			"@type":       "Rating",
			"ratingValue": rating,
			"bestRating":  numberFieldOr(data, 5, "best_rating", "max_rating"),
		}
	}
	if author := stringField(data, "author", "reviewer"); author != "" {
		out["author"] = map[string]any{"@type": "Person", "name": author}
	}
	if body := stringField(data, "review_body", "body", "content"); body != "" {
		out["reviewBody"] = body
	}
}

func mediaURLFromField(data map[string]any, mediaByID map[string]store.Medium, site store.Site, keys ...string) string {
	for _, k := range keys {
		v, ok := data[k]
		if !ok {
			continue
		}
		s, ok := v.(string)
		if !ok || s == "" {
			continue
		}
		if m, ok := mediaByID[s]; ok {
			rel := "/media/" + m.ID + "/" + m.Filename
			if site.Domain != "" {
				return absoluteURL(site, rel)
			}
			return rel
		}
		// Direct URL in the field (fallback when admin pasted a URL
		// instead of selecting a media id).
		if strings.HasPrefix(s, "http") {
			return s
		}
	}
	return ""
}

func numberField(data map[string]any, keys ...string) float64 {
	for _, k := range keys {
		if v, ok := data[k]; ok {
			switch n := v.(type) {
			case float64:
				return n
			case int:
				return float64(n)
			case int64:
				return float64(n)
			}
		}
	}
	return 0
}

func numberFieldOr(data map[string]any, fallback float64, keys ...string) float64 {
	if n := numberField(data, keys...); n != 0 {
		return n
	}
	return fallback
}

func stringFieldOr(data map[string]any, fallback string, keys ...string) string {
	if s := stringField(data, keys...); s != "" {
		return s
	}
	return fallback
}
