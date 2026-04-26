package analytics

import "strings"

// ParseUA returns (browser, os, device) for a User-Agent string. Pure
// heuristics matched against the User-Agent token zoo; no external deps.
//
// Goal: rough Umami-grade buckets, not pinpoint version detection. We label
// known crawlers as device="bot" so dashboards can exclude them with one
// filter.
//
// Order matters: bot patterns are tested first because many crawlers spoof
// a fake browser token; iOS / Android are tested before generic Safari /
// Chrome because mobile UAs include both.
func ParseUA(ua string) (browser, os, device string) {
	if ua == "" {
		return "", "", ""
	}
	low := strings.ToLower(ua)

	// Crawlers / bots first. Matches the Umami bot list plus our local
	// AI-bot blocklist.
	if isBot(low) {
		return botName(low), "Bot", "bot"
	}

	os = parseOS(low)
	device = parseDevice(low, os)
	browser = parseBrowser(low)

	return browser, os, device
}

func isBot(low string) bool {
	for _, b := range botMarkers {
		if strings.Contains(low, b) {
			return true
		}
	}
	return false
}

var botMarkers = []string{
	"bot", "crawler", "spider", "feedfetcher", "facebookexternalhit",
	"slurp", "duckduckgo", "applebot", "bingpreview", "linkedinbot",
	"telegrambot", "whatsapp", "discordbot", "skypeuripreview",
	"slackbot", "embedly", "quora link preview", "pinterest", "twitterbot",
	"redditbot", "yandex", "baidu", "petal", "ahrefs", "semrush",
	"mj12bot", "screaming frog", "lighthouse", "headlesschrome",
	"chrome-lighthouse", "googleother", "google-inspectiontool",
	"google-extended", "ccbot", "claude", "anthropic-ai", "gptbot",
	"chatgpt-user", "perplexitybot", "youbot",
}

// botName picks a friendly label for the most common bot families.
// Matches Site Inspector's privacy.go AI-bot list so the analytics page
// uses the same vocabulary as the security audit.
func botName(low string) string {
	switch {
	case strings.Contains(low, "googlebot"):
		return "Googlebot"
	case strings.Contains(low, "bingbot"):
		return "Bingbot"
	case strings.Contains(low, "duckduckbot"):
		return "DuckDuckBot"
	case strings.Contains(low, "applebot"):
		return "Applebot"
	case strings.Contains(low, "yandex"):
		return "YandexBot"
	case strings.Contains(low, "baidu"):
		return "Baiduspider"
	case strings.Contains(low, "facebookexternalhit"), strings.Contains(low, "meta-externalagent"):
		return "FacebookBot"
	case strings.Contains(low, "linkedinbot"):
		return "LinkedInBot"
	case strings.Contains(low, "twitterbot"):
		return "Twitterbot"
	case strings.Contains(low, "slackbot"):
		return "Slackbot"
	case strings.Contains(low, "discordbot"):
		return "Discordbot"
	case strings.Contains(low, "telegrambot"):
		return "Telegrambot"
	case strings.Contains(low, "whatsapp"):
		return "WhatsApp"
	case strings.Contains(low, "gptbot"):
		return "GPTBot"
	case strings.Contains(low, "chatgpt-user"):
		return "ChatGPT-User"
	case strings.Contains(low, "claude"), strings.Contains(low, "anthropic-ai"):
		return "ClaudeBot"
	case strings.Contains(low, "google-extended"):
		return "Google-Extended"
	case strings.Contains(low, "perplexitybot"):
		return "PerplexityBot"
	case strings.Contains(low, "ccbot"):
		return "CCBot"
	case strings.Contains(low, "ahrefs"):
		return "AhrefsBot"
	case strings.Contains(low, "semrush"):
		return "SemrushBot"
	case strings.Contains(low, "mj12bot"):
		return "MJ12Bot"
	case strings.Contains(low, "lighthouse"), strings.Contains(low, "headlesschrome"):
		return "Lighthouse"
	}
	return "Other Bot"
}

func parseOS(low string) string {
	switch {
	case strings.Contains(low, "windows nt"):
		return "Windows"
	case strings.Contains(low, "iphone os"), strings.Contains(low, "ios"):
		return "iOS"
	case strings.Contains(low, "ipad"):
		return "iPadOS"
	case strings.Contains(low, "android"):
		return "Android"
	case strings.Contains(low, "mac os x"), strings.Contains(low, "macintosh"):
		return "macOS"
	case strings.Contains(low, "cros"):
		return "ChromeOS"
	case strings.Contains(low, "ubuntu"):
		return "Ubuntu"
	case strings.Contains(low, "fedora"):
		return "Fedora"
	case strings.Contains(low, "linux"):
		return "Linux"
	case strings.Contains(low, "freebsd"):
		return "FreeBSD"
	}
	return "Other"
}

func parseDevice(low, os string) string {
	switch os {
	case "iOS", "Android":
		if strings.Contains(low, "ipad") || strings.Contains(low, "tablet") {
			return "tablet"
		}
		return "mobile"
	case "iPadOS":
		return "tablet"
	}
	if strings.Contains(low, "mobile") || strings.Contains(low, "phone") {
		return "mobile"
	}
	if strings.Contains(low, "tablet") {
		return "tablet"
	}
	return "desktop"
}

func parseBrowser(low string) string {
	// Order: more-specific tokens first. Edge / Opera / Vivaldi / Brave all
	// pretend to be Chrome, so they must be matched before generic Chrome.
	switch {
	case strings.Contains(low, "edg/"):
		return "Edge"
	case strings.Contains(low, "opr/"), strings.Contains(low, "opera"):
		return "Opera"
	case strings.Contains(low, "vivaldi"):
		return "Vivaldi"
	case strings.Contains(low, "brave"):
		return "Brave"
	case strings.Contains(low, "duckduckgo"):
		return "DuckDuckGo"
	case strings.Contains(low, "samsungbrowser"):
		return "Samsung Internet"
	case strings.Contains(low, "fxios"), strings.Contains(low, "firefox"):
		return "Firefox"
	case strings.Contains(low, "crios"), strings.Contains(low, "chrome"):
		return "Chrome"
	case strings.Contains(low, "safari"):
		return "Safari"
	}
	return "Other"
}

// ParsePrimaryLanguage returns the primary language tag from an
// Accept-Language header (e.g. "en", "sv-SE"). Empty string when no
// recognisable tag is present.
func ParsePrimaryLanguage(accLang string) string {
	if accLang == "" {
		return ""
	}
	// Take the first comma-separated entry, drop the q= weight.
	first := accLang
	if i := strings.Index(first, ","); i >= 0 {
		first = first[:i]
	}
	if i := strings.Index(first, ";"); i >= 0 {
		first = first[:i]
	}
	first = strings.TrimSpace(first)
	if first == "" || first == "*" {
		return ""
	}
	// Drop variant if present and keep the first 5 chars max ("sv-SE", "en").
	if len(first) > 5 {
		first = first[:5]
	}
	return first
}

// ParseUTMFromPath extracts utm_source / utm_medium / utm_campaign query
// params out of a request URI. Returns empty strings for missing params.
func ParseUTMFromPath(path string) (source, medium, campaign string) {
	q := strings.Index(path, "?")
	if q < 0 {
		return "", "", ""
	}
	for _, kv := range strings.Split(path[q+1:], "&") {
		eq := strings.Index(kv, "=")
		if eq < 0 {
			continue
		}
		key := kv[:eq]
		val := kv[eq+1:]
		switch strings.ToLower(key) {
		case "utm_source":
			source = limitString(val, 64)
		case "utm_medium":
			medium = limitString(val, 64)
		case "utm_campaign":
			campaign = limitString(val, 128)
		}
	}
	return source, medium, campaign
}

func limitString(s string, max int) string {
	if len(s) > max {
		return s[:max]
	}
	return s
}
