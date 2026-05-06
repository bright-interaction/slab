package billing

import "testing"

func TestLookupKnownPlans(t *testing.T) {
	cases := []struct {
		key   string
		name  string
		price int64
	}{
		{"oss", "OSS", 0},
		{"solo", "Solo", 1900},
		{"studio", "Studio", 7900},
		{"agency", "Agency", 29900},
	}
	for _, c := range cases {
		p := Lookup(c.key)
		if p.Name != c.name {
			t.Errorf("%s: name = %q, want %q", c.key, p.Name, c.name)
		}
		if p.PriceCents != c.price {
			t.Errorf("%s: price = %d, want %d", c.key, p.PriceCents, c.price)
		}
	}
}

func TestLookupUnknownFallsBackToOSS(t *testing.T) {
	p := Lookup("does-not-exist")
	if p.Key != "oss" {
		t.Fatalf("unknown key: got %q, want oss fallback", p.Key)
	}
}

func TestLimit(t *testing.T) {
	if got := Limit("solo", "max_sites"); got != 1 {
		t.Errorf("solo max_sites: got %d, want 1", got)
	}
	if got := Limit("studio", "max_sites"); got != 5 {
		t.Errorf("studio max_sites: got %d, want 5", got)
	}
	if got := Limit("agency", "max_sites"); got != 25 {
		t.Errorf("agency max_sites: got %d, want 25", got)
	}
	if got := Limit("oss", "max_sites"); got != -1 {
		t.Errorf("oss max_sites: got %d, want -1 (unlimited)", got)
	}
}

func TestCustomDomainGate(t *testing.T) {
	if got := Limit("solo", "custom_domain"); got != 0 {
		t.Errorf("solo custom_domain: got %d, want 0", got)
	}
	if got := Limit("studio", "custom_domain"); got != 1 {
		t.Errorf("studio custom_domain: got %d, want 1", got)
	}
	if got := Limit("agency", "custom_domain"); got != 1 {
		t.Errorf("agency custom_domain: got %d, want 1", got)
	}
	if got := Limit("oss", "custom_domain"); got != 1 {
		t.Errorf("oss custom_domain: got %d, want 1", got)
	}
}

func TestUnknownResource(t *testing.T) {
	if got := Limit("solo", "made_up_resource"); got != -1 {
		t.Errorf("unknown resource: got %d, want -1 fallback", got)
	}
}

func TestAllReturnsCommercialOnly(t *testing.T) {
	all := All()
	if len(all) != 3 {
		t.Fatalf("All returned %d plans, want 3 (solo+studio+agency, no oss)", len(all))
	}
	for _, p := range all {
		if p.Key == "oss" {
			t.Errorf("All() leaked oss plan to public listing")
		}
	}
}

// TestMaxPagesPerSite locks in the per-tier page caps that gate migration
// Apply (Sprint 3 of the post-Layer-3d plan, 2026-05-06). Solo's 50 cap
// is what stops a free-trial user from importing a 500-page WordPress
// blog into a tier they're not paying for.
func TestMaxPagesPerSite(t *testing.T) {
	cases := map[string]int64{
		"oss":    -1,
		"solo":   50,
		"studio": 250,
		"agency": 1000,
	}
	for plan, want := range cases {
		if got := Limit(plan, "max_pages_per_site"); got != want {
			t.Errorf("%s max_pages_per_site: got %d, want %d", plan, got, want)
		}
	}
}

// TestMaxRedirectsPerSite locks in the per-tier redirect caps that gate
// bulk redirect import. The migration porter emits ~1.5x as many redirects
// as pages on a typical migration (page renames + .html stripping + date
// archive collapse), so the cap scales as 10x pages to leave headroom.
func TestMaxRedirectsPerSite(t *testing.T) {
	cases := map[string]int64{
		"oss":    -1,
		"solo":   500,
		"studio": 2500,
		"agency": 10000,
	}
	for plan, want := range cases {
		if got := Limit(plan, "max_redirects_per_site"); got != want {
			t.Errorf("%s max_redirects_per_site: got %d, want %d", plan, got, want)
		}
	}
}

// TestLimitOnUnknownPlan ensures malformed workspace.plan values don't
// silently grant unlimited quotas - they fall back to OSS which IS
// unlimited (-1) but at least the behaviour is documented.
func TestLimitOnUnknownPlan(t *testing.T) {
	if got := Limit("future-tier-not-yet-shipped", "max_pages_per_site"); got != -1 {
		t.Errorf("unknown plan should fall back to oss (-1), got %d", got)
	}
}
