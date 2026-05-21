package critique

import "testing"

// TestInspirationsFor_HeroReturnsThreeCuratedReferences locks the
// contract that hero authors get 3 references (the playbook's
// "consult 3 references before writing" guidance maps to this number).
func TestInspirationsFor_HeroReturnsThreeCuratedReferences(t *testing.T) {
	got := InspirationsFor("hero")
	if len(got) != 3 {
		t.Fatalf("hero inspirations = %d; want 3", len(got))
	}
	for _, ins := range got {
		if ins.Repo == "" || ins.Path == "" || ins.Pattern == "" || ins.WhyRead == "" {
			t.Errorf("incomplete inspiration: %+v", ins)
		}
	}
}

// TestInspirationsFor_PricingReturnsAtLeastOne confirms the canonical
// AstroWind 3-tier reference is in the list. If this regresses, the
// pricing authoring loop loses its anchor reference.
func TestInspirationsFor_PricingReturnsAtLeastOne(t *testing.T) {
	got := InspirationsFor("pricing")
	if len(got) == 0 {
		t.Fatalf("pricing inspirations empty")
	}
	found := false
	for _, ins := range got {
		if ins.Repo == "astrowind" && ins.Path == "astrowind/src/components/widgets/Pricing.astro" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected astrowind Pricing.astro in pricing inspirations; got %+v", got)
	}
}

// TestInspirationsFor_UnknownTypeReturnsNil belts the "no false-positive
// suggestions" rule: an unknown block_type returns nil, not a generic
// fallback that would distract the agent.
func TestInspirationsFor_UnknownTypeReturnsNil(t *testing.T) {
	got := InspirationsFor("not_a_real_block")
	if got != nil {
		t.Errorf("expected nil for unknown block_type; got %v", got)
	}
}

// TestInspirationsFor_CoversCanonicalBlockTypes spot-checks the most
// common atomicsite block types so the curated map doesn't lose its
// minimum-viable coverage. New block_types added to renderer should
// also get an entry here.
func TestInspirationsFor_CoversCanonicalBlockTypes(t *testing.T) {
	for _, bt := range []string{
		"hero",
		"split_hero",
		"pricing",
		"accordion_faq",
		"stat_grid",
		"feature_grid",
		"process_steps",
		"cta",
		"about_split",
		"logo_strip",
	} {
		if len(InspirationsFor(bt)) == 0 {
			t.Errorf("block_type %q has no curated inspirations", bt)
		}
	}
}
