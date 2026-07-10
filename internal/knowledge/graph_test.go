package knowledge

import (
	"strings"
	"testing"
)

// TestBuildGraphProducesNodesForEveryDoc asserts the loader's docs are
// all reachable as graph nodes. If a doc lands without a node, the
// agent's curriculum-by-graph traversal would be missing that chapter.
func TestBuildGraphProducesNodesForEveryDoc(t *testing.T) {
	g := BuildGraph(nil, nil, nil)
	docNodes := map[string]bool{}
	for _, n := range g.Nodes {
		if n.Kind == "doc" {
			docNodes[n.ID] = true
		}
	}
	for _, d := range All() {
		want := "doc:" + d.Slug
		if !docNodes[want] {
			t.Errorf("graph missing doc node %q", want)
		}
	}
}

// TestBuildGraphMentionEdgesAreFiltered guards against false-positive
// edges. A tool name only edges from a doc when the name appears in the
// known tool set passed in. Words in prose like "sort_order" must not
// produce a tool edge unless the caller registered "sort_order" as a
// tool.
func TestBuildGraphMentionEdgesAreFiltered(t *testing.T) {
	known := []string{"trigger_build", "upsert_css_class"}
	g := BuildGraph(known, nil, nil)
	for _, e := range g.Edges {
		if e.Kind != "mentions" {
			continue
		}
		if !strings.HasPrefix(e.To, "tool:") {
			continue
		}
		name := strings.TrimPrefix(e.To, "tool:")
		ok := false
		for _, k := range known {
			if k == name {
				ok = true
				break
			}
		}
		if !ok {
			t.Errorf("graph emitted edge to unregistered tool %q", name)
		}
	}
}

// TestBuildGraphResourceURIEdges verifies that doc bodies referencing
// slab:// URIs produce edges to the matching resource nodes when
// the URI is on the registered set.
func TestBuildGraphResourceURIEdges(t *testing.T) {
	knownResources := []string{
		"slab://site/security_posture",
		"slab://site/context",
		"slab://knowledge/typography-scale",
	}
	g := BuildGraph(nil, knownResources, nil)
	found := map[string]bool{}
	for _, e := range g.Edges {
		if e.Kind == "mentions" && strings.HasPrefix(e.To, "resource:") {
			found[e.To] = true
		}
	}
	// security-authoring doc references slab://site/security_posture.
	want := "resource:slab://site/security_posture"
	if !found[want] {
		t.Errorf("expected mentions edge to %q from security-authoring doc; got %v", want, found)
	}
}

// TestBuildGraphOrderedAfterChainsCurriculum verifies the curriculum
// reading-order chain so master_the_stack can walk docs deterministically.
func TestBuildGraphOrderedAfterChainsCurriculum(t *testing.T) {
	g := BuildGraph(nil, nil, nil)
	stackChain := 0
	uxChain := 0
	for _, e := range g.Edges {
		if e.Kind != "ordered_after" {
			continue
		}
		// from and to must both be docs
		if !strings.HasPrefix(e.From, "doc:") || !strings.HasPrefix(e.To, "doc:") {
			continue
		}
		fromSlug := strings.TrimPrefix(e.From, "doc:")
		toSlug := strings.TrimPrefix(e.To, "doc:")
		fromDoc, _ := Get(fromSlug)
		toDoc, _ := Get(toSlug)
		if fromDoc.Category != toDoc.Category {
			t.Errorf("ordered_after edge crosses categories: %s -> %s", fromDoc.Slug, toDoc.Slug)
		}
		if fromDoc.Order >= toDoc.Order {
			t.Errorf("ordered_after edge violates ordering: %s(%d) -> %s(%d)",
				fromDoc.Slug, fromDoc.Order, toDoc.Slug, toDoc.Order)
		}
		switch fromDoc.Category {
		case CategoryStack:
			stackChain++
		case CategoryUX:
			uxChain++
		}
	}
	// 11 stack docs (Sprint 4 added collection-design-patterns +
	// schema-org-per-collection-type; Phase 31.1.1 added
	// analytics-conversions-and-identify) and 10 ux docs; chain is n-1.
	if stackChain != 10 {
		t.Errorf("expected 10 stack ordered_after edges, got %d", stackChain)
	}
	if uxChain != 9 {
		t.Errorf("expected 9 ux ordered_after edges, got %d", uxChain)
	}
}

// TestBuildGraphTagEdgesClusterDocs guards the tag-cluster surface so
// the agent can answer "which docs relate to motion" with one resource
// fetch.
func TestBuildGraphTagEdgesClusterDocs(t *testing.T) {
	g := BuildGraph(nil, nil, nil)
	tagSeen := map[string]int{}
	for _, e := range g.Edges {
		if e.Kind != "tagged" {
			continue
		}
		tagSeen[e.To]++
	}
	// every category must appear as a tag with at least one connection
	if tagSeen["tag:stack"] == 0 {
		t.Error("expected tag:stack to have at least one edge")
	}
	if tagSeen["tag:ux"] == 0 {
		t.Error("expected tag:ux to have at least one edge")
	}
}

// TestBuildGraphStatsConsistent guards the small summary the agent reads
// alongside the full graph.
func TestBuildGraphStatsConsistent(t *testing.T) {
	g := BuildGraph([]string{"trigger_build"}, []string{"slab://site/context"}, nil)
	if g.Stats.NodeCount != len(g.Nodes) {
		t.Errorf("node_count %d does not match len(Nodes) %d", g.Stats.NodeCount, len(g.Nodes))
	}
	if g.Stats.EdgeCount != len(g.Edges) {
		t.Errorf("edge_count %d does not match len(Edges) %d", g.Stats.EdgeCount, len(g.Edges))
	}
	if g.Stats.NodesByKind["doc"] != len(All()) {
		t.Errorf("expected NodesByKind[doc] = %d, got %d", len(All()), g.Stats.NodesByKind["doc"])
	}
	if g.Stats.NodesByKind["tool"] != 1 || g.Stats.NodesByKind["resource"] != 1 {
		t.Errorf("expected one tool + one resource node, got %v", g.Stats.NodesByKind)
	}
}
