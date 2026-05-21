package mcp

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"sort"
	"sync"
	"time"

	"github.com/bright-interaction/slab/internal/agent"
	authmw "github.com/bright-interaction/slab/internal/middleware"
	"github.com/bright-interaction/slab/internal/migration"
	"github.com/bright-interaction/slab/internal/shield"
	"github.com/bright-interaction/slab/internal/store"
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

// PreviewTokenMinter is the slice of *handlers.PreviewTokenManager the
// MCP server needs for preview_screenshot. Single-method interface keeps
// the import cycle clean (handlers imports mcp; mcp cannot import
// handlers).
type PreviewTokenMinter interface {
	Mint(siteID, pageID string) (string, error)
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

	// designReferencesDir points at the bundled MIT-licensed reference
	// repos that the search_design_corpus tool walks. Empty disables
	// the tool with a clean "not configured" error.
	designReferencesDir string

	// verifyJobMgr drives the async verify-live worker for migration
	// MCP tools (Sprint 4, 2026-05-06). Nil disables verify_migration_live
	// + cancel_verify_job with a clean "not configured" error so the
	// rest of the migration surface stays callable.
	verifyJobMgr *migration.JobManager

	// media is the MediaUploader the apply_migration tool injects into
	// the porter so imported images get re-hosted (no source-CDN
	// hot-linking). Nil makes apply_migration fall back to a noop
	// uploader which records each image as a warning - the migration
	// still applies, the human just gets to retry media later.
	media migration.MediaUploader

	// shieldStore + shieldKey wire LLM-boundary tokenization (Shield).
	// When shieldEnabled is true, tool responses are redacted on the way
	// out and tool-call arguments are unshielded on the way in. Off by
	// default; flipped on by WithShield() at server construction.
	//
	// shieldHintLevel dials how much value-derived metadata appears on
	// markers (HintFull preserves v1 behavior; HintMinimal/HintNone
	// hide identifying hints when the moat needs the agent to know
	// "same person across turns" but not "who that person is").
	shieldStore     shield.Store
	shieldMu        sync.RWMutex // guards shieldKey so /admin/reload-secrets can hot-swap
	shieldKey       []byte
	shieldEnabled   bool
	shieldTTL       time.Duration
	shieldHintLevel shield.HintLevel

	// previewTokens + loopbackPort wire the preview_screenshot tool: the
	// tool mints a one-shot token via previewTokens.Mint, drives chromedp
	// at http://127.0.0.1:{loopbackPort}/_preview/{token}, and the
	// loopback-only Serve handler swaps the token for the rendered draft
	// HTML so chromedp can capture the visual. Nil disables the tool with
	// a clean "not configured" error.
	previewTokens PreviewTokenMinter
	loopbackPort  int

	tools     map[string]Tool
	resources map[string]Resource
	prompts   map[string]Prompt
}

// WithPreviewTokens wires the preview-screenshot loopback path. port is
// the HTTP port the server is listening on (cfg.Port); the MCP tool
// builds http://127.0.0.1:{port}/_preview/{token} URLs and drives
// chromedp against them. Calling with a nil minter disables the
// preview_screenshot tool surface with a clean error message.
func (s *Server) WithPreviewTokens(m PreviewTokenMinter, port int) *Server {
	s.previewTokens = m
	s.loopbackPort = port
	return s
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

// WithDesignReferencesDir configures the corpus root for the
// search_design_corpus tool. Should be set to cfg.DesignReferencesDir.
func (s *Server) WithDesignReferencesDir(dir string) *Server {
	s.designReferencesDir = dir
	return s
}

// WithVerifyJobManager wires the async verify-live worker so the
// verify_migration_live + cancel_verify_job MCP tools can enqueue +
// cancel runs. Sprint 4 (2026-05-06).
func (s *Server) WithVerifyJobManager(m *migration.JobManager) *Server {
	s.verifyJobMgr = m
	return s
}

// WithMediaUploader wires the production media uploader the porter
// uses when apply_migration is invoked through MCP. Same instance the
// HTTP MigrationHandler uses; without it apply_migration falls back to
// a noop uploader and records warnings per image.
func (s *Server) WithMediaUploader(m migration.MediaUploader) *Server {
	s.media = m
	return s
}

// WithShield enables LLM-boundary tokenization. store and key are
// required; key must be 32 bytes (the same ATOMICSITE_SHIELD_KEY env
// var consumed at startup). ttl defaults to 30 minutes when zero.
//
// When enabled: every tool result + resource read passes through
// Session.ShieldJSON before reaching the agent (regex-based PII
// redaction as a safety net), and every tool-call argument blob passes
// through Session.UnshieldJSON before reaching the handler (token
// resolution). Tool handlers can fetch the per-request *shield.Session
// from ctx via shield.SessionFromContext for explicit struct-tag
// shielding of typed responses (preferred over the regex safety net
// when the response shape is known).
//
// Off by default so existing tests + dev installs without a
// SHIELD_KEY do not break.
//
// hintLevel defaults to HintFull when zero (the iota default). Pass
// HintMinimal/HintNone to harden the moat: with deterministic per-
// session token ids the agent still gets stable identity ("the same
// lead across this conversation"), but no identifying hints leak.
func (s *Server) WithShield(store shield.Store, key []byte, ttl time.Duration, hintLevel shield.HintLevel) *Server {
	if store == nil || len(key) != 32 {
		return s
	}
	s.shieldStore = store
	s.shieldMu.Lock()
	s.shieldKey = key
	s.shieldMu.Unlock()
	s.shieldEnabled = true
	s.shieldHintLevel = hintLevel
	if ttl > 0 {
		s.shieldTTL = ttl
	} else {
		// 24h default lets real Claude Code / agent conversations span
		// a full working day. Caller can override to a shorter TTL for
		// stricter deployments. The TTL is housekeeping, not auth;
		// X-Agent-Key validation upstream is the security boundary.
		s.shieldTTL = 24 * time.Hour
	}
	return s
}

// UpdateShieldKey hot-swaps the in-memory AES-256 key the MCP server
// hands to every new shield.Session. Safe for concurrent use; existing
// Sessions hold their own key by value (snapshot at NewSession time)
// and keep working until they expire, so PII tokenized with the prior
// key remains resolvable for that session's lifetime. Tokens are
// session-scoped so cross-session reads cannot occur.
//
// Returns an error when shield was never enabled, the new key is the
// wrong length, or the value is byte-identical to the current key. The
// rotation engine treats a returned error as a phase-1 failure and
// records it in the entry's last_rotation_error.
func (s *Server) UpdateShieldKey(newKey []byte) error {
	if !s.shieldEnabled {
		return fmt.Errorf("shield: not enabled, cannot rotate key")
	}
	if len(newKey) != 32 {
		return fmt.Errorf("shield: key must be 32 bytes, got %d", len(newKey))
	}
	s.shieldMu.Lock()
	defer s.shieldMu.Unlock()
	if len(s.shieldKey) == 32 && string(s.shieldKey) == string(newKey) {
		// Same-value updates are a no-op rather than an error so the
		// rotation engine's dual-phase delivery (open-grace then
		// close-grace, both sending the new value) lands cleanly. The
		// allowlist drops SHIELD_KEY_PREVIOUS, so phase-2 looks
		// identical to phase-1 by the time we see it.
		return nil
	}
	// Make a defensive copy so the caller cannot zero the buffer underneath
	// concurrent NewSession reads that capture this slice header.
	dup := make([]byte, 32)
	copy(dup, newKey)
	s.shieldKey = dup
	return nil
}

// ShieldEnabled reports whether tokenization is active. Used by the
// admin AgentSurface to surface the privacy invariant to operators.
func (s *Server) ShieldEnabled() bool { return s.shieldEnabled }

// beginShieldSession returns the per-request shield Session keyed by
// the Mcp-Session-Id header (preferred) or a stable hash of the
// X-Agent-Key when no MCP session id is present. Returns nil + nil
// when shield is disabled. Returns nil + err on driver / crypto
// failure; callers should fall back to passthrough on error so a
// shield outage does not block legitimate MCP traffic (the alternative
// is silently leaking PII, which is worse).
func (s *Server) beginShieldSession(r *http.Request) (*shield.Session, error) {
	if !s.shieldEnabled {
		return nil, nil
	}
	id := r.Header.Get("Mcp-Session-Id")
	if id == "" {
		// Fall back to a hash of the agent key so the same caller gets
		// a stable session across requests within the TTL window.
		// SHA-256 of the raw key keeps the lookup id non-reversible.
		key := r.Header.Get("X-Agent-Key")
		if key == "" {
			return nil, errors.New("shield: no Mcp-Session-Id and no X-Agent-Key")
		}
		sum := sha256.Sum256([]byte(key))
		id = "agent-" + hex.EncodeToString(sum[:8])
	}
	s.shieldMu.RLock()
	key := s.shieldKey
	s.shieldMu.RUnlock()
	return shield.NewSession(r.Context(), s.shieldStore, id, key, s.shieldTTL, s.shieldHintLevel)
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
	s.registerDesignTools()
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
		Instructions: "atomicsite MCP server. On session start: (1) read atomicsite://meta/design-playbook (the rubric the Inspector grades against: principles, archetypes, anti-patterns, vibe archetypes, materiality, slop terms, motion budget, copy voice, font system, hero graphics) BEFORE authoring any block, (2) read atomicsite://meta/capabilities for identity plus the live tools/resources/prompts surface, (3) read atomicsite://site/context for site state. Use tools/list to see writable surfaces. Visitor data plus identified-tier PII is intentionally NOT exposed via MCP.",
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

	ctx := r.Context()
	session, sessErr := s.beginShieldSession(r)
	if sessErr != nil {
		return ToolCallResult{
			IsError: true,
			Content: []Content{{Type: "text", Text: "shield session error: " + sessErr.Error()}},
		}, nil
	}
	if session != nil {
		ctx = shield.WithSession(ctx, session)
		// Resolve any tokens in the agent's tool-call arguments back to
		// plaintext before the handler runs against the store. A token
		// that doesn't exist in this session (replay or expiry) bubbles
		// up as an application error so the agent can recover.
		if len(args.Arguments) > 0 {
			resolved, err := session.UnshieldJSON(ctx, args.Arguments)
			if err != nil {
				return ToolCallResult{
					IsError: true,
					Content: []Content{{Type: "text", Text: "shield: failed to resolve tool arguments: " + err.Error()}},
				}, nil
			}
			args.Arguments = resolved
		}
	}

	body, err := tool.Handler(ctx, identity, args.Arguments)
	if err != nil {
		return ToolCallResult{
			IsError: true,
			Content: []Content{{Type: "text", Text: scrubError(ctx, session, err)}},
		}, nil
	}

	if session != nil && body != "" {
		shielded, sErr := session.ShieldJSON(ctx, []byte(body))
		if sErr == nil {
			body = string(shielded)
		}
		// On ShieldJSON parse error, fall through with the raw body.
		// This typically means the handler returned non-JSON (e.g. a
		// plain "ok"), which is safe because the redact pass was the
		// safety net; tagged-struct shielding is the primary path and
		// happens inside the handler.
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

	ctx := r.Context()
	session, _ := s.beginShieldSession(r)
	if session != nil {
		ctx = shield.WithSession(ctx, session)
	}

	body, err := res.Reader(ctx, identity)
	if err != nil {
		return nil, &ResponseError{Code: ErrInternal, Message: err.Error()}
	}

	if session != nil && body != "" {
		shielded, sErr := session.ShieldJSON(ctx, []byte(body))
		if sErr == nil {
			body = string(shielded)
		}
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

	ctx := r.Context()
	session, _ := s.beginShieldSession(r)
	if session != nil {
		ctx = shield.WithSession(ctx, session)
	}

	msgs, err := pr.Render(ctx, identity, args.Arguments)
	if err != nil {
		return nil, &ResponseError{Code: ErrInternal, Message: scrubError(ctx, session, err)}
	}

	if session != nil {
		// Prompt messages are free-form text the agent receives as
		// system/user context. Run the redact pass over each one so
		// any embedded PII (entity ids, emails inlined into a
		// playbook) becomes a marker before reaching the LLM.
		for i := range msgs {
			if msgs[i].Content.Text == "" {
				continue
			}
			redacted, rerr := session.RedactString(ctx, msgs[i].Content.Text)
			if rerr == nil {
				msgs[i].Content.Text = redacted
			}
		}
	}

	return map[string]any{
		"description": pr.Description,
		"messages":    msgs,
	}, nil
}

// scrubError runs an error message through the regex redact pass when
// shield is enabled, so a "lead 'anna@andersson-law.se' not found"
// surfaces to the agent as "lead '[shield:email:tok_X]' not found"
// rather than leaking the email. Returns the original message when
// shield is off or the redact pass itself errors.
func scrubError(ctx context.Context, session *shield.Session, err error) string {
	if err == nil {
		return ""
	}
	msg := err.Error()
	if session == nil {
		return msg
	}
	scrubbed, rerr := session.RedactString(ctx, msg)
	if rerr != nil {
		return msg
	}
	return scrubbed
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
