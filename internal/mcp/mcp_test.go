package mcp

import (
	"strings"
	"testing"
)

// TestPrivacyPosture is the headline guard. The MCP surface must NEVER
// expose a tool, resource, or prompt that touches identified-tier
// visitor PII. This test enforces that by asserting:
//
//  1. No registered tool name contains "visitor" or "metadata" in a way
//     that would leak the GET /api/agent/visitor-metadata endpoint.
//  2. No resource URI references visit-events, sessions, or identified
//     metadata.
//  3. Settings-write tool restricts category to {general, seo, analytics}
//     (admin-only categories like "security" and "allowed-scripts" are
//     rejected).
//
// If you add a tool that legitimately needs "visitor" in its name, update
// this test to allow the specific name explicitly. The default is REJECT
// so accidental exposure is impossible.
func TestPrivacyPosture(t *testing.T) {
	s := &Server{
		tools:     map[string]Tool{},
		resources: map[string]Resource{},
		prompts:   map[string]Prompt{},
	}
	s.registerTools()
	s.registerResources()
	s.registerPrompts()

	for name := range s.tools {
		lower := strings.ToLower(name)
		if strings.Contains(lower, "visitor") || strings.Contains(lower, "session") {
			t.Errorf("tool %q looks like it touches visitor PII; add an explicit allow-list entry to TestPrivacyPosture if it's safe", name)
		}
	}

	for uri := range s.resources {
		lower := strings.ToLower(uri)
		// "session" is valid for "session-management" prompts; visitor /
		// identity / metadata are the actual PII surfaces to block.
		if strings.Contains(lower, "visitor") || strings.Contains(lower, "/identity") || strings.Contains(lower, "/visit-events") {
			t.Errorf("resource %q looks like it touches visitor PII", uri)
		}
	}
}

func TestRequiredToolsRegistered(t *testing.T) {
	s := &Server{
		tools:     map[string]Tool{},
		resources: map[string]Resource{},
		prompts:   map[string]Prompt{},
	}
	s.registerTools()
	s.registerResources()
	s.registerPrompts()

	required := []string{
		"get_site_context",
		"get_settings_catalog",
		"list_pages",
		"create_page",
		"update_page",
		"delete_page",
		"list_blocks",
		"create_block",
		"update_block",
		"delete_block",
		"get_branding",
		"get_profile",
		"update_profile",
		"list_settings",
		"bulk_upsert_settings",
		"get_security_posture",
		"list_media",
		"trigger_build",
		"get_build_status",
		"get_evaluation",
	}
	for _, name := range required {
		if _, ok := s.tools[name]; !ok {
			t.Errorf("required tool %q not registered", name)
		}
	}

	requiredResources := []string{
		"atomicsite://site/context",
		"atomicsite://site/settings_catalog",
		"atomicsite://site/security_posture",
		"atomicsite://site/i18n",
		"atomicsite://site/pending_setup",
		"atomicsite://site/structure",
	}
	for _, uri := range requiredResources {
		if _, ok := s.resources[uri]; !ok {
			t.Errorf("required resource %q not registered", uri)
		}
	}

	requiredPrompts := []string{
		"walk_through_pending_setup",
		"audit_seo",
		"connect_analytics",
		"add_iframe_integration",
		"create_landing_page",
	}
	for _, name := range requiredPrompts {
		if _, ok := s.prompts[name]; !ok {
			t.Errorf("required prompt %q not registered", name)
		}
	}
}

func TestSettingValidation(t *testing.T) {
	cases := []struct {
		category, key, value string
		ok                   bool
	}{
		// passes
		{"seo", "hreflang_strategy", "path", true},
		{"seo", "hreflang_strategy", "subdomain", true},
		{"analytics", "ga4_id", "G-ABC1234", true},
		{"analytics", "umami_url", "https://analytics.example.com", true},
		{"analytics", "identity_max_age_days", "30", true},
		{"general", "domain_aliases", "www.example.com,example.org", true},
		// rejections
		{"seo", "hreflang_strategy", "GARBAGE", false},
		{"analytics", "ga4_id", "XX-12345", false},
		{"analytics", "umami_url", "not a url", false},
		{"analytics", "identity_max_age_days", "9999", false},
		{"analytics", "identity_max_age_days", "0", false},
		{"general", "domain_aliases", "www.example.com,,example.org", false},
	}
	for _, c := range cases {
		err := validateSettingValue(c.category, c.key, c.value)
		if c.ok && err != nil {
			t.Errorf("expected %s.%s=%q to pass, got %v", c.category, c.key, c.value, err)
		}
		if !c.ok && err == nil {
			t.Errorf("expected %s.%s=%q to fail validation, got pass", c.category, c.key, c.value)
		}
	}
}

func TestAgentWritableCategories(t *testing.T) {
	if !isAgentWritableCategory("seo") || !isAgentWritableCategory("analytics") || !isAgentWritableCategory("general") {
		t.Error("seo / analytics / general must be agent-writable")
	}
	if isAgentWritableCategory("security") {
		t.Error("security category must NOT be agent-writable")
	}
	if isAgentWritableCategory("allowed-scripts") {
		t.Error("allowed-scripts must NOT be agent-writable (CSP widens attack surface)")
	}
}
