// Package builder - feeds.go.
//
// Emits an Atom 1.0 feed per opted-in collection at
// public/{collection.slug}/feed.xml. Atom (RFC 4287) over RSS 2.0
// because: it has a real spec, mandatory unique IDs and timestamps,
// no ambiguous string-vs-CDATA bodies, and every modern reader
// (Feedly, Inoreader, NetNewsWire, Mozilla, Apple Mail) speaks it
// natively. Discovery is a separate concern; the per-page <link
// rel="alternate" type="application/atom+xml"> tag lives wherever
// page <head> is composed.
//
// Collection settings (stored in collections.settings_json):
//
//   feed_enabled      *bool  // explicit on/off; defaults to true
//                            // when schema_org_type is one of
//                            // {BlogPosting, NewsArticle, Article}
//   feed_title        string // overrides Collection.Name in <title>
//   feed_description  string // <subtitle>; falls back to a
//                            // generic "Latest from {Collection}".
//   feed_max_entries  int    // cap on entries; defaults to 50
//
// Items render in published_at DESC order; non-published items are
// excluded. Locales: every locale's items are merged into one feed
// (per collection, not per locale) so a Swedish + English bilingual
// blog produces one /blog/feed.xml with both languages tagged via
// the per-entry xml:lang attribute. Most readers handle that fine
// and a single feed avoids the user needing to subscribe twice.

package builder

import (
	"context"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"log/slog"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/bright-interaction/slab/internal/store"
)

// collectionFeedSettings is the subset of settings_json relevant to
// the feed builder. Held separately from collectionSettings so the
// JSON shape is unambiguous about pointer-bool semantics.
type collectionFeedSettings struct {
	Enabled     *bool  `json:"feed_enabled,omitempty"`
	Title       string `json:"feed_title,omitempty"`
	Description string `json:"feed_description,omitempty"`
	MaxEntries  int    `json:"feed_max_entries,omitempty"`
}

// articleSchemas is the set of schema.org types that auto-enable the
// feed when feed_enabled is unset. Operators with a "Recipes" or
// "Products" collection would set feed_enabled=false explicitly.
var articleSchemas = map[string]bool{
	"BlogPosting":    true,
	"NewsArticle":    true,
	"Article":        true,
	"Posting":        true,
	"TechArticle":    true,
	"AnalysisReport": true,
}

// defaultFeedMaxEntries caps the number of entries in a feed. Most
// readers (and most users' attention) cap out around 50.
const defaultFeedMaxEntries = 50

// RenderCollectionFeeds writes one Atom feed file per opted-in
// collection. Returns the count of feeds written. Wired into Build()
// after RenderCollectionPages so item URLs already match what the
// pages emit.
func RenderCollectionFeeds(ctx context.Context, queries *store.Queries, siteID, wsDir string) (int, error) {
	collections, err := queries.ListCollectionsBySite(ctx, siteID)
	if err != nil {
		return 0, fmt.Errorf("list collections: %w", err)
	}
	if len(collections) == 0 {
		return 0, nil
	}

	site, err := queries.GetSiteByID(ctx, siteID)
	if err != nil {
		return 0, fmt.Errorf("get site: %w", err)
	}

	siteBaseURL := canonicalBaseURL(site)
	feedsEmitted := 0

	for _, c := range collections {
		var cs collectionSettings
		_ = json.Unmarshal([]byte(c.SettingsJson), &cs)
		var fs collectionFeedSettings
		_ = json.Unmarshal([]byte(c.SettingsJson), &fs)

		if !shouldEmitFeed(cs.SchemaOrgType, fs) {
			continue
		}
		if !collectionSlugSafe(c.Slug) {
			slog.Warn("feeds: skipping unsafe slug", "site_id", siteID, "slug", c.Slug)
			continue
		}

		items, err := queries.ListPublishedItemsByCollection(ctx, c.ID)
		if err != nil {
			return feedsEmitted, fmt.Errorf("list items %s: %w", c.Slug, err)
		}
		if len(items) == 0 {
			// Empty feed is still valid Atom; emit it anyway so the
			// route exists for readers that subscribe ahead of
			// content. Saves a 404 nuisance.
		}

		// Sort by published_at DESC; fall back to created_at to keep
		// items with empty published_at at the bottom rather than
		// dropping them entirely.
		sort.SliceStable(items, func(i, j int) bool {
			return feedItemTimestamp(items[i]) > feedItemTimestamp(items[j])
		})
		max := fs.MaxEntries
		if max <= 0 {
			max = defaultFeedMaxEntries
		}
		if len(items) > max {
			items = items[:max]
		}

		feedXML, err := renderAtomFeed(siteBaseURL, site, c, fs, items)
		if err != nil {
			return feedsEmitted, fmt.Errorf("render feed %s: %w", c.Slug, err)
		}
		path := filepath.Join(wsDir, "public", c.Slug, "feed.xml")
		if err := WriteFile(path, feedXML); err != nil {
			return feedsEmitted, fmt.Errorf("write feed %s: %w", c.Slug, err)
		}
		feedsEmitted++
	}
	return feedsEmitted, nil
}

// shouldEmitFeed encodes the explicit-vs-auto-default policy. Explicit
// settings always win; auto-default ON only when the schema is
// article-like.
func shouldEmitFeed(schemaOrgType string, fs collectionFeedSettings) bool {
	if fs.Enabled != nil {
		return *fs.Enabled
	}
	return articleSchemas[strings.TrimSpace(schemaOrgType)]
}

// feedItemTimestamp returns a sortable timestamp string for an item.
// published_at takes precedence; created_at is the fallback.
func feedItemTimestamp(it store.CollectionItem) string {
	if strings.TrimSpace(it.PublishedAt) != "" {
		return it.PublishedAt
	}
	return it.CreatedAt
}

// canonicalBaseURL derives the absolute origin used to construct
// feed-entry URLs. Falls back to the site's domain field; final
// fallback to "/" so relative links still parse if neither is set.
// The builder package's existing canonical helpers are page-scoped;
// this is the simpler "site origin" reduction.
func canonicalBaseURL(site store.Site) string {
	d := strings.TrimSpace(site.Domain)
	if d == "" {
		return ""
	}
	if !strings.Contains(d, "://") {
		d = "https://" + d
	}
	return strings.TrimRight(d, "/")
}

// atomFeed and atomEntry mirror the relevant subset of RFC 4287.
// encoding/xml handles all the escaping; we don't hand-format
// elements with user content.
type atomFeed struct {
	XMLName  xml.Name    `xml:"feed"`
	Xmlns    string      `xml:"xmlns,attr"`
	XMLLang  string      `xml:"xml:lang,attr,omitempty"`
	ID       string      `xml:"id"`
	Title    string      `xml:"title"`
	Subtitle string      `xml:"subtitle,omitempty"`
	Updated  string      `xml:"updated"`
	Links    []atomLink  `xml:"link"`
	Author   *atomAuthor `xml:"author,omitempty"`
	Entries  []atomEntry `xml:"entry"`
}

type atomLink struct {
	Rel  string `xml:"rel,attr,omitempty"`
	Type string `xml:"type,attr,omitempty"`
	Href string `xml:"href,attr"`
}

type atomAuthor struct {
	Name string `xml:"name"`
}

type atomEntry struct {
	XMLLang   string      `xml:"xml:lang,attr,omitempty"`
	ID        string      `xml:"id"`
	Title     string      `xml:"title"`
	Updated   string      `xml:"updated"`
	Published string      `xml:"published,omitempty"`
	Links     []atomLink  `xml:"link"`
	Summary   *atomText   `xml:"summary,omitempty"`
	Content   *atomText   `xml:"content,omitempty"`
	Author    *atomAuthor `xml:"author,omitempty"`
}

// atomText carries either plain text or HTML payload. type="html"
// means the value is escaped HTML; encoding/xml handles the escape.
type atomText struct {
	Type string `xml:"type,attr,omitempty"`
	Body string `xml:",chardata"`
}

func renderAtomFeed(baseURL string, site store.Site, c store.Collection, fs collectionFeedSettings, items []store.CollectionItem) (string, error) {
	title := strings.TrimSpace(fs.Title)
	if title == "" {
		title = c.Name
	}
	subtitle := strings.TrimSpace(fs.Description)
	if subtitle == "" {
		subtitle = "Latest from " + c.Name
	}
	feedURL := strings.TrimRight(baseURL, "/") + "/" + c.Slug + "/feed.xml"
	htmlURL := strings.TrimRight(baseURL, "/") + "/" + c.Slug + "/"

	updated := feedUpdatedAt(items)
	feed := atomFeed{
		Xmlns:    "http://www.w3.org/2005/Atom",
		XMLLang:  site.Lang,
		ID:       feedURL,
		Title:    title,
		Subtitle: subtitle,
		Updated:  updated,
		Links: []atomLink{
			{Rel: "self", Type: "application/atom+xml", Href: feedURL},
			{Rel: "alternate", Type: "text/html", Href: htmlURL},
		},
	}
	if site.Name != "" {
		feed.Author = &atomAuthor{Name: site.Name}
	}

	for _, it := range items {
		entry := buildAtomEntry(baseURL, site, c, it)
		feed.Entries = append(feed.Entries, entry)
	}

	body, err := xml.MarshalIndent(feed, "", "  ")
	if err != nil {
		return "", fmt.Errorf("marshal atom: %w", err)
	}
	return xml.Header + string(body) + "\n", nil
}

func buildAtomEntry(baseURL string, site store.Site, c store.Collection, it store.CollectionItem) atomEntry {
	loc := normalizeLocale(it.Locale, site.Lang)
	rel := collectionItemPath(c.Slug, it.Slug, loc, site.Lang)
	itemURL := strings.TrimRight(baseURL, "/") + "/" + strings.TrimLeft(rel, "/") + "/"
	updated := feedEntryTimestamp(it)
	entry := atomEntry{
		XMLLang: loc,
		ID:      itemURL,
		Title:   it.Title,
		Updated: updated,
		Links: []atomLink{
			{Rel: "alternate", Type: "text/html", Href: itemURL},
		},
	}
	if pub := strings.TrimSpace(it.PublishedAt); pub != "" {
		entry.Published = pub
	}
	if summary := extractFeedSummary(it.DataJson); summary != "" {
		entry.Summary = &atomText{Type: "text", Body: summary}
	}
	if site.Name != "" {
		entry.Author = &atomAuthor{Name: site.Name}
	}
	return entry
}

// feedUpdatedAt returns the feed's <updated>: the most recent
// timestamp across entries. Empty feed gets the current time so
// readers don't choke.
func feedUpdatedAt(items []store.CollectionItem) string {
	best := ""
	for _, it := range items {
		t := feedEntryTimestamp(it)
		if t > best {
			best = t
		}
	}
	if best == "" {
		return time.Now().UTC().Format(time.RFC3339)
	}
	return best
}

// feedEntryTimestamp returns the entry's updated timestamp in
// RFC3339-ish form. SQLite stores datetime('now') as
// "YYYY-MM-DD HH:MM:SS" without a timezone marker; Atom needs the
// 'T' separator and a Z suffix for UTC. We translate inline.
func feedEntryTimestamp(it store.CollectionItem) string {
	raw := strings.TrimSpace(it.UpdatedAt)
	if raw == "" {
		raw = strings.TrimSpace(it.PublishedAt)
	}
	if raw == "" {
		raw = strings.TrimSpace(it.CreatedAt)
	}
	if raw == "" {
		return time.Now().UTC().Format(time.RFC3339)
	}
	// SQLite's "2026-05-09 10:00:00" → "2026-05-09T10:00:00Z".
	if len(raw) == 19 && raw[10] == ' ' {
		return raw[:10] + "T" + raw[11:] + "Z"
	}
	// Already RFC3339 (ISO-8601) or some custom format; pass through.
	return raw
}

// extractFeedSummary pulls a short text summary from the item's
// data_json. The CMS schema is per-collection so we can't rely on a
// single field name; we peek at common ones in priority order.
func extractFeedSummary(dataJSON string) string {
	if strings.TrimSpace(dataJSON) == "" {
		return ""
	}
	var data map[string]any
	if err := json.Unmarshal([]byte(dataJSON), &data); err != nil {
		return ""
	}
	for _, key := range []string{"summary", "excerpt", "description", "subtitle", "lead", "intro"} {
		if v, ok := data[key].(string); ok {
			s := strings.TrimSpace(v)
			if s != "" {
				return s
			}
		}
	}
	return ""
}
