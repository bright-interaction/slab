package handlers

import (
	"fmt"
	"net/url"
	"strconv"
	"strings"
)

// validateSetting checks one {category, key, value} write against the
// constraints documented in the agent SettingsCatalog. Returns nil when
// the value is acceptable, or a user-facing error string when it isn't.
//
// Empty values are always accepted (they clear the row). Validation only
// fires on non-empty input. This matches the admin UI's behaviour: typing
// nothing into a field is the right way to revert to a default, and the
// pre-existing settings system has always treated "" as "unset".
//
// Validators here mirror the catalog in
// internal/agent/settings_catalog.go. When you add a new setting there,
// add the matching validator here so admin + agent writes both gate.
func validateSetting(category, key, value string) error {
	if value == "" {
		return nil
	}
	switch category {
	case "general":
		return validateGeneralSetting(key, value)
	case "seo":
		return validateSEOSetting(key, value)
	case "analytics":
		return validateAnalyticsSetting(key, value)
	case "security":
		return validateSecuritySetting(key, value)
	}
	// Unknown categories pass through; future-compat for whatever the next
	// phase introduces. The handler-level routing already guards write
	// access (agent_settings.go.agentWritableSettingsCategories).
	return nil
}

func validateGeneralSetting(key, value string) error {
	switch key {
	case "additional_langs", "domain_aliases":
		// CSV; reject empty entries between commas to surface typos.
		for _, raw := range strings.Split(value, ",") {
			if strings.TrimSpace(raw) == "" {
				return fmt.Errorf("general.%s contains an empty entry; check the comma list", key)
			}
		}
	}
	return nil
}

func validateSEOSetting(key, value string) error {
	switch key {
	case "hreflang_strategy":
		return enumIn("seo.hreflang_strategy", value, "path", "subdomain", "off")
	case "canonical_base":
		return mustBeAbsoluteURL("seo.canonical_base", value)
	case "sitemap_enabled":
		return boolValue("seo.sitemap_enabled", value)
	}
	return nil
}

func validateAnalyticsSetting(key, value string) error {
	switch key {
	case "cookieproof_enabled",
		"atomicsite_tracking_enabled",
		"ga4_enabled",
		"umami_enabled",
		"personalization_enabled":
		return boolValue("analytics."+key, value)
	case "ga4_id":
		// G-XXXXXXX. Permissive: accept any G- prefix followed by alnum.
		if !strings.HasPrefix(value, "G-") || len(value) < 4 {
			return fmt.Errorf("analytics.ga4_id must look like G-XXXXXXX")
		}
	case "umami_url", "crm_webhook_url":
		return mustBeAbsoluteURL("analytics."+key, value)
	case "identity_max_age_days":
		return intInRange("analytics.identity_max_age_days", value, 1, 3650)
	}
	return nil
}

func validateSecuritySetting(key, value string) error {
	switch key {
	case "hsts_enabled",
		"hsts_preload",
		"csp_enabled",
		"x_content_type_options",
		"https_redirect":
		return boolValue("security."+key, value)
	case "hsts_max_age":
		return intInRange("security.hsts_max_age", value, 0, 63072000)
	case "x_frame_options":
		return enumIn("security.x_frame_options", value, "DENY", "SAMEORIGIN")
	case "referrer_policy":
		return enumIn("security.referrer_policy", value,
			"no-referrer", "no-referrer-when-downgrade", "origin",
			"origin-when-cross-origin", "same-origin", "strict-origin",
			"strict-origin-when-cross-origin", "unsafe-url")
	case "coop":
		return enumIn("security.coop", value, "same-origin", "same-origin-allow-popups", "unsafe-none")
	case "corp":
		return enumIn("security.corp", value, "same-origin", "same-site", "cross-origin")
	case "coep":
		return enumIn("security.coep", value, "require-corp", "credentialless")
	}
	return nil
}

func enumIn(field, value string, allowed ...string) error {
	for _, a := range allowed {
		if value == a {
			return nil
		}
	}
	return fmt.Errorf("%s must be one of: %s", field, strings.Join(allowed, ", "))
}

func boolValue(field, value string) error {
	switch value {
	case "0", "1", "true", "false", "yes", "no", "on", "off":
		return nil
	}
	return fmt.Errorf("%s must be 0 or 1", field)
}

func intInRange(field, value string, min, max int64) error {
	n, err := strconv.ParseInt(value, 10, 64)
	if err != nil {
		return fmt.Errorf("%s must be an integer", field)
	}
	if n < min || n > max {
		return fmt.Errorf("%s must be between %d and %d (got %d)", field, min, max, n)
	}
	return nil
}

func mustBeAbsoluteURL(field, value string) error {
	u, err := url.Parse(value)
	if err != nil || u.Scheme == "" || u.Host == "" {
		return fmt.Errorf("%s must be an absolute URL with scheme and host", field)
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return fmt.Errorf("%s must use http or https", field)
	}
	return nil
}
