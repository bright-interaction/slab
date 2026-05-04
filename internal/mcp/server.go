package mcp

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"sort"

	"github.com/brightinteraction/atomicsite/internal/agent"
	authmw "github.com/brightinteraction/atomicsite/internal/middleware"
	"github.com/brightinteraction/atomicsite/internal/store"
)

// BuildTrigger is the small interface the MCP server depends on for the
// async build pipeline. Satisfied by *handlers.BuildHandler. Defined as
// an interface here to avoid an import cycle (handlers imports mcp via
// the route wiring in server.go).
type BuildTrigger interface {
	// StartBuild creates a deployment record + kicks off an async build
	// for the given site. Returns build_id; status is polled separately.
	StartBuild(ctx context.Context, siteID string) (string, error)
	// GetBuildState returns the in-memory state plus the persisted
	// deployment row. The MCP wrapper exposes both.
	GetBuildState(ctx context.Context, buildID, siteID string) (any, error)
}

// Server is the MCP server. It owns the registries (tools, resources,
// prompts) and the JSON-RPC dispatch loop. One instance is constructed at
// startup and mounted under /mcp behind the existing AgentAuthMiddleware.
type Server struct {
	queries *store.Queries
	context *agent.ContextBuilder
	builds  BuildTrigger

	// fontsDir is the on-disk root the upload_font tool writes woff2
	// files to, mirroring the FontsHandler's storage path. Optional;
	// when empty, upload_font returns a not-configured error so unit
	// tests can construct Servers without filesystem coupling.
	fontsDir string

	// baseURL is the admin's external base URL, used by tools that
	// need to point users at admin pages (e.g. get_figma_import_url
	// returns BaseURL + /sites/{siteID}/settings/design/figma).
	baseURL string

	tools     map[string]Tool
	resources map[string]Resource
	prompts   map[string]Prompt
}

// WithFontsDir configures the on-disk root for woff2 uploads. Call once
// after NewServer with the same value handlers use (cfg.FontsDir).
func (s *Server) WithFontsDir(dir string) *Server {
	s.fontsDir = dir
	return s
}

// WithBaseURL configures the admin base URL used by tools that need to
// link the user back to the admin (e.g. Figma import).
func (s *Server) WithBaseURL(u string) *Server {
	s.baseURL = u
	return s
}

// NewServer constructs an MCP server with every registered tool /
// resource / prompt wired against the given store. Registries are
// populated from tools.go, resources.go, prompts.go via the package-level
// helpers; calling NewServer multiple times is safe but pointless (the
// registries are static).
//
// builds may be nil for unit tests — the build-trigger / status / eval
// tools handle nil with a clean "not wired" error so the rest of the
// surface stays callable.
func NewServer(queries *store.Queries, builds BuildTrigger) *Server {
	s := &Server{
		queries:   queries,
		context:   agent.NewContextBuilder(queries),
		builds:    builds,
		tools:     map[string]Tool{},
		resources: map[string]Resource{},
		prompts:   map[string]Prompt{},
	}
	s.registerTools()
	s.registerResources()
	s.registerPrompts()
	return s
}

// Handler returns the http.Handler that the router mounts at /mcp.
//
// Method routing:
//
//	POST   /mcp  client-to-server JSON-RPC; we answer inline
//	GET    /mcp  reserved for SSE upgrade in the Streamable HTTP spec;
//	             we return 405 today (everything completes synchronously)
//
// The handler depends on the AgentAuthMiddleware running first so the
// AgentIdentity is in context. /mcp without a valid X-Agent-Key returns
// 401 from the middleware before reaching this handler.
func (s *Server) Handler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodPost:
			s.handlePost(w, r)
		case http.MethodGet, http.MethodOptions:
			// Streamable HTTP spec leaves room for SSE-pull on GET. Not
			// needed yet; everything resolves synchronously today.
			http.Error(w, "MCP server does not stream; POST JSON-RPC requests instead", http.StatusMethodNotAllowed)
		default:
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		}
	})
}

// handlePost decodes one JSON-RPC request envelope, dispatches it to the
// matching method handler, and writes the response. Notifications (id
// missing) get a 204 with no body, per spec.
func (s *Server) handlePost(w http.ResponseWriter, r *http.Request) {
	identity := authmw.GetAgent(r)
	if identity == nil {
		// AgentAuthMiddleware should have caught this, but defend in
		// depth so a misconfigured route doesn't expose unauth tool
		// calls.
		writeJSONRPCError(w, nil, ErrInvalidRequest, "agent not authenticated", nil)
		return
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		writeJSONRPCError(w, nil, ErrParse, "failed to read body", nil)
		return
	}
	var req Request
	if err := json.Unmarshal(body, &req); err != nil {
		writeJSONRPCError(w, nil, ErrParse, "invalid JSON: "+err.Error(), nil)
		return
	}
	if req.JSONRPC != JSONRPCVersion {
		writeJSONRPCError(w, req.ID, ErrInvalidRequest, "jsonrpc must be \"2.0\"", nil)
		return
	}

	// Notifications carry no id; method handlers that have no useful
	// response (initialized, ping when sent as notification) can be
	// silently ack'd with 204.
	if req.ID == nil {
		s.handleNotification(req, identity)
		w.WriteHeader(http.StatusNoContent)
		return
	}

	result, rpcErr := s.dispatch(r, req, identity)
	resp := Response{JSONRPC: JSONRPCVersion, ID: req.ID}
	if rpcErr != nil {
		resp.Error = rpcErr
	} else {
		resp.Result = result
	}
	writeJSON(w, http.StatusOK, resp)
}

// dispatch routes one method to its handler. Returns either a result body
// or a JSON-RPC error envelope.
func (s *Server) dispatch(r *http.Request, req Request, identity *authmw.AgentIdentity) (any, *ResponseError) {
	switch req.Method {
	case "initialize":
		return s.handleInitialize(req)
	case "ping":
		return map[string]any{}, nil
	case "tools/list":
		return s.handleToolsList(), nil
	case "tools/call":
		return s.handleToolsCall(r, identity, req.Params)
	case "resources/list":
		return s.handleResourcesList(), nil
	case "resources/read":
		return s.handleResourcesRead(r, identity, req.Params)
	case "prompts/list":
		return s.handlePromptsList(), nil
	case "prompts/get":
		return s.handlePromptsGet(r, identity, req.Params)
	}
	return nil, &ResponseError{Code: ErrMethodNotFound, Message: "method not found: " + req.Method}
}

// handleNotification is the silent-ack path for non-response messages.
// "notifications/initialized" is the canonical example: client sends it
// after receiving the initialize result; no response expected.
func (s *Server) handleNotification(req Request, _ *authmw.AgentIdentity) {
	// No-op today. Hook here if we ever need to track per-session state
	// (warm caches on initialized, log shutdown intent on cancelled).
}

// --- initialize -----------------------------------------------------------

func (s *Server) handleInitialize(req Request) (any, *ResponseError) {
	var args struct {
		ProtocolVersion string `json:"protocolVersion"`
	}
	if len(req.Params) > 0 {
		_ = json.Unmarshal(req.Params, &args)
	}
	version := Protocol
	if args.ProtocolVersion != "" {
		// Echo the client's offered version. Hosts that pin to an older
		// spec keep working because the methods we use are stable
		// across the 2024-11-05 → 2025-03-26 jump.
		version = args.ProtocolVersion
	}
	return InitializeResult{
		ProtocolVersion: version,
		ServerInfo: ServerInfo{
			Name:    "atomicsite",
			Version: "1.0.0",
		},
		Capabilities: Capabilities{
			Tools:     &struct{}{},
			Resources: &struct{}{},
			Prompts:   &struct{}{},
		},
		Instructions: "atomicsite MCP server. Read resources/list to see live site context (settings, security posture, i18n, pages). Use tools/list to see writable surfaces. Visitor data + identified-tier PII is intentionally NOT exposed via MCP.",
	}, nil
}

// --- tools ----------------------------------------------------------------

type toolsListResult struct {
	Tools []Tool `json:"tools"`
}

func (s *Server) handleToolsList() any {
	out := make([]Tool, 0, len(s.tools))
	for _, t := range s.tools {
		out = append(out, t)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return toolsListResult{Tools: out}
}

func (s *Server) handleToolsCall(r *http.Request, identity *authmw.AgentIdentity, params json.RawMessage) (any, *ResponseError) {
	var args struct {
		Name      string          `json:"name"`
		Arguments json.RawMessage `json:"arguments"`
	}
	if err := json.Unmarshal(params, &args); err != nil {
		return nil, &ResponseError{Code: ErrInvalidParams, Message: "invalid params: " + err.Error()}
	}
	tool, ok := s.tools[args.Name]
	if !ok {
		// Unknown tool name returns an application-level error, not a
		// JSON-RPC error. Hosts surface this to the model so it can
		// pick a different tool or ask the user for clarification.
		return ToolCallResult{
			IsError: true,
			Content: []Content{{Type: "text", Text: "unknown tool: " + args.Name + " (call tools/list to see what's available)"}},
		}, nil
	}
	if tool.RequiresWrite && !agent.ValidateCapability(identity.Capabilities, "write") {
		return ToolCallResult{
			IsError: true,
			Content: []Content{{Type: "text", Text: "tool " + args.Name + " requires the 'write' capability; ask the human to mint a write-enabled key"}},
		}, nil
	}
	body, err := tool.Handler(r.Context(), identity, args.Arguments)
	if err != nil {
		return ToolCallResult{
			IsError: true,
			Content: []Content{{Type: "text", Text: err.Error()}},
		}, nil
	}
	return ToolCallResult{
		Content: []Content{{Type: "text", Text: body}},
	}, nil
}

// --- resources ------------------------------------------------------------

type resourcesListResult struct {
	Resources []Resource `json:"resources"`
}

func (s *Server) handleResourcesList() any {
	out := make([]Resource, 0, len(s.resources))
	for _, r := range s.resources {
		out = append(out, r)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].URI < out[j].URI })
	return resourcesListResult{Resources: out}
}

func (s *Server) handleResourcesRead(r *http.Request, identity *authmw.AgentIdentity, params json.RawMessage) (any, *ResponseError) {
	var args struct {
		URI string `json:"uri"`
	}
	if err := json.Unmarshal(params, &args); err != nil {
		return nil, &ResponseError{Code: ErrInvalidParams, Message: "invalid params: " + err.Error()}
	}
	res, ok := s.resources[args.URI]
	if !ok {
		return nil, &ResponseError{Code: ErrInvalidParams, Message: "unknown resource URI: " + args.URI}
	}
	body, err := res.Reader(r.Context(), identity)
	if err != nil {
		return nil, &ResponseError{Code: ErrInternal, Message: err.Error()}
	}
	mime := res.MimeType
	if mime == "" {
		mime = "application/json"
	}
	return map[string]any{
		"contents": []map[string]any{
			{"uri": args.URI, "mimeType": mime, "text": body},
		},
	}, nil
}

// --- prompts --------------------------------------------------------------

type promptsListResult struct {
	Prompts []Prompt `json:"prompts"`
}

func (s *Server) handlePromptsList() any {
	out := make([]Prompt, 0, len(s.prompts))
	for _, p := range s.prompts {
		out = append(out, p)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return promptsListResult{Prompts: out}
}

func (s *Server) handlePromptsGet(r *http.Request, identity *authmw.AgentIdentity, params json.RawMessage) (any, *ResponseError) {
	var args struct {
		Name      string            `json:"name"`
		Arguments map[string]string `json:"arguments"`
	}
	if err := json.Unmarshal(params, &args); err != nil {
		return nil, &ResponseError{Code: ErrInvalidParams, Message: "invalid params: " + err.Error()}
	}
	pr, ok := s.prompts[args.Name]
	if !ok {
		return nil, &ResponseError{Code: ErrInvalidParams, Message: "unknown prompt: " + args.Name}
	}
	msgs, err := pr.Render(r.Context(), identity, args.Arguments)
	if err != nil {
		return nil, &ResponseError{Code: ErrInternal, Message: err.Error()}
	}
	return map[string]any{
		"description": pr.Description,
		"messages":    msgs,
	}, nil
}

// writeJSONRPCError sends a JSON-RPC error envelope with the given fields.
// Used when a request can't even be dispatched (parse failure, unknown
// method); application errors ride inside successful results.
func writeJSONRPCError(w http.ResponseWriter, id any, code int, msg string, data any) {
	resp := Response{
		JSONRPC: JSONRPCVersion,
		ID:      id,
		Error:   &ResponseError{Code: code, Message: msg, Data: data},
	}
	writeJSON(w, http.StatusOK, resp)
}

// errMustBeString is used when a JSON arg expected a string but got
// something else; centralised so errors stay consistent.
var errMustBeString = errors.New("argument must be a string")
