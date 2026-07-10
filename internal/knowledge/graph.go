package knowledge

import (
	"regexp"
	"sort"
	"strings"
)

// GraphNode is one entry in the knowledge / MCP-surface graph. Nodes
// carry just enough metadata for the agent to decide which doc / tool /
// resource to fetch next. Bodies are deliberately absent; the agent
// follows up with a resources/read or tools/list call when it picks a
// node.
type GraphNode struct {
	ID       string   `json:"id"`
	Kind     string   `json:"kind"` // "doc" | "resource" | "tool" | "prompt" | "tag"
	Title    string   `json:"title"`
	Category string   `json:"category,omitempty"`
	Tags     []string `json:"tags,omitempty"`
	Order    int      `json:"order,omitempty"`
}

// GraphEdge connects two nodes by ID. Edges are typed by Kind so the
// agent can filter ("which docs reference the upsert_css_class tool?")
// without parsing prose.
type GraphEdge struct {
	From string `json:"from"`
	To   string `json:"to"`
	Kind string `json:"kind"` // "mentions" | "tagged" | "ordered_after"
}

// Graph is the full payload behind slab://meta/knowledge-graph.
// Fields are JSON-stable so external clients can index them.
type Graph struct {
	Nodes []GraphNode `json:"nodes"`
	Edges []GraphEdge `json:"edges"`
	Stats GraphStats  `json:"stats"`
}

// GraphStats is a small summary clients can render without scanning
// the full edge list.
type GraphStats struct {
	NodeCount     int            `json:"node_count"`
	EdgeCount     int            `json:"edge_count"`
	NodesByKind   map[string]int `json:"nodes_by_kind"`
	EdgesByKind   map[string]int `json:"edges_by_kind"`
	TopReferenced []string       `json:"top_referenced"`
}

// resourceURIPattern matches every slab:// URI that may appear
// inside curriculum bodies. The agent uses these to navigate from a
// concept doc to a runtime resource (e.g. the security-authoring doc
// references slab://site/security_posture).
var resourceURIPattern = regexp.MustCompile(`slab://[a-z0-9_/-]+`)

// toolNamePattern matches snake_case tool names mentioned in body text.
// We only emit a "mentions" edge if the matched name is on the known
// tools list (registered by the MCP server), to avoid false positives
// like "sort_order" or "data_json".
var toolNamePattern = regexp.MustCompile(`\b[a-z][a-z0-9_]+[a-z0-9]\b`)

// BuildGraph computes the knowledge graph from the embedded curriculum
// plus the live MCP surface (tool names, resource URIs, prompt names
// the caller threads in). Pure function; no I/O. Cheap enough to run on
// every resources/read of slab://meta/knowledge-graph.
//
// Tool, resource, prompt name slices are passed in instead of being
// imported from the mcp package because internal/knowledge cannot
// depend on internal/mcp without creating a cycle (mcp imports
// knowledge for resource registration).
func BuildGraph(toolNames, resourceURIs, promptNames []string) Graph {
	docs := All()

	// Deterministic ordering throughout. JSON output stable.
	sort.Strings(toolNames)
	sort.Strings(resourceURIs)
	sort.Strings(promptNames)

	nodes := make([]GraphNode, 0, len(docs)+len(toolNames)+len(resourceURIs)+len(promptNames))
	for _, d := range docs {
		nodes = append(nodes, GraphNode{
			ID:       "doc:" + d.Slug,
			Kind:     "doc",
			Title:    d.Title,
			Category: string(d.Category),
			Tags:     extractTags(d),
			Order:    d.Order,
		})
	}
	for _, n := range toolNames {
		nodes = append(nodes, GraphNode{
			ID:    "tool:" + n,
			Kind:  "tool",
			Title: n,
		})
	}
	for _, uri := range resourceURIs {
		nodes = append(nodes, GraphNode{
			ID:    "resource:" + uri,
			Kind:  "resource",
			Title: uri,
		})
	}
	for _, p := range promptNames {
		nodes = append(nodes, GraphNode{
			ID:    "prompt:" + p,
			Kind:  "prompt",
			Title: p,
		})
	}

	// Edges: for each doc body, scan for resource URI mentions and known
	// tool / prompt names. Build "mentions" edges; deduplicate.
	toolSet := setOf(toolNames)
	resourceSet := setOf(resourceURIs)
	promptSet := setOf(promptNames)

	edgeKey := map[string]bool{}
	edges := make([]GraphEdge, 0, 256)
	addEdge := func(from, to, kind string) {
		k := from + "->" + to + ":" + kind
		if edgeKey[k] {
			return
		}
		edgeKey[k] = true
		edges = append(edges, GraphEdge{From: from, To: to, Kind: kind})
	}

	for _, d := range docs {
		from := "doc:" + d.Slug
		// Resource URIs.
		for _, uri := range uniqueMatches(resourceURIPattern, d.Body) {
			if resourceSet[uri] {
				addEdge(from, "resource:"+uri, "mentions")
			}
		}
		// Tool / prompt names. We only edge to names in the known sets to
		// avoid prose false positives.
		for _, name := range uniqueMatches(toolNamePattern, d.Body) {
			if toolSet[name] {
				addEdge(from, "tool:"+name, "mentions")
			}
			if promptSet[name] {
				addEdge(from, "prompt:"+name, "mentions")
			}
		}
	}

	// Tag edges (clusters in the graph). Tags come from the doc summary
	// and category, lowercased and tokenised into common UX/UI vocab.
	tagNodes := map[string]bool{}
	for _, d := range docs {
		from := "doc:" + d.Slug
		for _, t := range extractTags(d) {
			tagID := "tag:" + t
			if !tagNodes[tagID] {
				nodes = append(nodes, GraphNode{
					ID:    tagID,
					Kind:  "tag",
					Title: t,
				})
				tagNodes[tagID] = true
			}
			addEdge(from, tagID, "tagged")
		}
	}

	// Ordered-after edges within each category so the agent can walk the
	// curriculum in sequence by following the chain from the lowest
	// Order doc forward.
	stackDocs := []Doc{}
	uxDocs := []Doc{}
	for _, d := range docs {
		switch d.Category {
		case CategoryStack:
			stackDocs = append(stackDocs, d)
		case CategoryUX:
			uxDocs = append(uxDocs, d)
		}
	}
	for i := 1; i < len(stackDocs); i++ {
		addEdge("doc:"+stackDocs[i-1].Slug, "doc:"+stackDocs[i].Slug, "ordered_after")
	}
	for i := 1; i < len(uxDocs); i++ {
		addEdge("doc:"+uxDocs[i-1].Slug, "doc:"+uxDocs[i].Slug, "ordered_after")
	}

	stats := computeStats(nodes, edges)

	return Graph{
		Nodes: nodes,
		Edges: edges,
		Stats: stats,
	}
}

// extractTags derives a small set of conceptual tags from each doc, so
// the graph clusters by topic. Tags come from category + a hand-curated
// per-slug list. The agent uses tag edges to find related docs without
// parsing prose.
func extractTags(d Doc) []string {
	base := []string{string(d.Category)}
	switch d.Slug {
	case "astro-conventions":
		base = append(base, "astro", "build", "framework")
	case "typescript-strict":
		base = append(base, "typescript", "code-quality")
	case "css-variable-system":
		base = append(base, "css", "tokens", "build")
	case "block-renderer-patterns":
		base = append(base, "blocks", "composition")
	case "i18n-authoring":
		base = append(base, "i18n", "seo")
	case "security-authoring":
		base = append(base, "security", "csp", "headers")
	case "personalization":
		base = append(base, "personalization", "privacy")
	case "cookieproof-integration":
		base = append(base, "consent", "privacy", "gdpr")
	case "typography-scale":
		base = append(base, "typography", "tokens")
	case "color-system":
		base = append(base, "color", "tokens", "branding")
	case "spacing-rhythm":
		base = append(base, "spacing", "tokens", "rhythm")
	case "motion-curves":
		base = append(base, "motion", "performance")
	case "accessibility-patterns":
		base = append(base, "accessibility", "wcag")
	case "performance-budgets":
		base = append(base, "performance", "lcp", "budgets")
	case "forms-ux":
		base = append(base, "forms", "ux", "accessibility")
	case "nav-ux":
		base = append(base, "navigation", "ux", "accessibility")
	case "dark-mode":
		base = append(base, "dark-mode", "tokens", "branding")
	case "premium-design-principles":
		base = append(base, "premium", "design-discipline")
	}
	sort.Strings(base)
	return base
}

func setOf(xs []string) map[string]bool {
	m := make(map[string]bool, len(xs))
	for _, x := range xs {
		m[x] = true
	}
	return m
}

func uniqueMatches(re *regexp.Regexp, body string) []string {
	seen := map[string]bool{}
	out := []string{}
	for _, m := range re.FindAllString(body, -1) {
		m = strings.TrimRight(m, ".,;:)")
		if seen[m] {
			continue
		}
		seen[m] = true
		out = append(out, m)
	}
	return out
}

func computeStats(nodes []GraphNode, edges []GraphEdge) GraphStats {
	byKindNode := map[string]int{}
	for _, n := range nodes {
		byKindNode[n.Kind]++
	}
	byKindEdge := map[string]int{}
	indegree := map[string]int{}
	for _, e := range edges {
		byKindEdge[e.Kind]++
		indegree[e.To]++
	}

	type ref struct {
		id  string
		cnt int
	}
	refs := make([]ref, 0, len(indegree))
	for id, c := range indegree {
		refs = append(refs, ref{id, c})
	}
	sort.Slice(refs, func(i, j int) bool {
		if refs[i].cnt != refs[j].cnt {
			return refs[i].cnt > refs[j].cnt
		}
		return refs[i].id < refs[j].id
	})
	top := []string{}
	for i := 0; i < len(refs) && i < 10; i++ {
		top = append(top, refs[i].id)
	}

	return GraphStats{
		NodeCount:     len(nodes),
		EdgeCount:     len(edges),
		NodesByKind:   byKindNode,
		EdgesByKind:   byKindEdge,
		TopReferenced: top,
	}
}
