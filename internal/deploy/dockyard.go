package deploy

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"
)

// dockyardHTTPTimeout caps a single HTTP call to the Dockyard admin API.
// Caller's context still wins if it's tighter.
const dockyardHTTPTimeout = 30 * time.Second

// uuidPattern is a permissive UUID-ish matcher: hex segments separated by
// dashes. Dockyard server IDs are UUIDs, but we don't insist on the exact
// 8-4-4-4-12 layout in case the format ever shifts.
var uuidPattern = regexp.MustCompile(`^[a-fA-F0-9]{8}-?[a-fA-F0-9]{4}-?[a-fA-F0-9]{4}-?[a-fA-F0-9]{4}-?[a-fA-F0-9]{12}$`)

// DockyardDeployer pushes a built site to a Dockyard-managed server. It runs
// in two stages: rsync the dist tree to the host filesystem, then register a
// Caddy route via the Dockyard admin API so traffic for the site's domain is
// served from that path. Idempotent: re-running on the same target = no-op
// rsync + 409-on-route-already-exists, both treated as success.
type DockyardDeployer struct {
	httpClient *http.Client
	rsync      *RsyncDeployer
	client     dockyardClient
}

// NewDockyardDeployer returns a configured DockyardDeployer with sensible
// defaults. Tests can swap dockyardClient via SetClient.
func NewDockyardDeployer() *DockyardDeployer {
	hc := &http.Client{Timeout: dockyardHTTPTimeout}
	return &DockyardDeployer{
		httpClient: hc,
		rsync:      NewRsyncDeployer(),
	}
}

// Kind reports the deployer kind.
func (d *DockyardDeployer) Kind() string { return "dockyard" }

// SetClient overrides the Dockyard API client. Used by tests.
func (d *DockyardDeployer) SetClient(c dockyardClient) { d.client = c }

// dockyardConfig is the parsed view of target.Config for dockyard targets.
type dockyardConfig struct {
	DockyardURL   string
	APIKey        string
	ServerID      string
	Domain        string
	Host          string
	User          string
	Port          int
	PrivateKeyPEM string
	HostPath      string
}

// dockyardClient is the narrow surface the deployer uses to talk to Dockyard.
// Everything routes through here so tests can fake responses without wiring
// up a real httptest server.
type dockyardClient interface {
	PutProxyRoute(ctx context.Context, serverID string, body proxyRouteRequest) error
	ApplyProxy(ctx context.Context, serverID string) error
}

// proxyRouteRequest mirrors the body Dockyard's POST /api/servers/{id}/proxy/routes
// expects. Dockyard has no native "static files" upstream kind, so we emit a
// snippet of Caddy custom_config that does `root` + `file_server` against the
// host path the rsync step just wrote to. UpstreamTarget still has to be set
// since Dockyard validates it as required; the file_server directive in
// custom_config takes precedence at request time.
type proxyRouteRequest struct {
	Domain          string `json:"domain"`
	UpstreamTarget  string `json:"upstream_target"`
	SSLMode         string `json:"ssl_mode"`
	ListenInterface string `json:"listen_interface"`
	CustomConfig    string `json:"custom_config"`
}

// parseDockyardConfig pulls the dockyard-specific fields from a Target.Config
// map. Permissive about port type to match RsyncDeployer.
func parseDockyardConfig(cfg map[string]any) (dockyardConfig, error) {
	out := dockyardConfig{Port: 22}
	if cfg == nil {
		return out, errors.New("dockyard deploy: config is empty")
	}

	out.DockyardURL = strings.TrimSpace(stringFrom(cfg["dockyard_url"]))
	out.APIKey = stringFrom(cfg["api_key"])
	out.ServerID = strings.TrimSpace(stringFrom(cfg["server_id"]))
	out.Domain = strings.TrimSpace(stringFrom(cfg["domain"]))
	out.Host = strings.TrimSpace(stringFrom(cfg["host"]))
	out.User = strings.TrimSpace(stringFrom(cfg["user"]))
	out.PrivateKeyPEM = stringFrom(cfg["private_key_pem"])
	out.HostPath = strings.TrimSpace(stringFrom(cfg["host_path"]))

	if v, ok := cfg["port"]; ok && v != nil {
		switch n := v.(type) {
		case float64:
			out.Port = int(n)
		case int:
			out.Port = n
		case int64:
			out.Port = int(n)
		case string:
			if n == "" {
				break
			}
			parsed, err := parsePort(n)
			if err != nil {
				return out, err
			}
			out.Port = parsed
		default:
			return out, fmt.Errorf("dockyard deploy: port has unsupported type %T", v)
		}
	}

	return out, nil
}

// rsyncTargetFor builds a synthetic rsync Target so the embedded
// RsyncDeployer can be reused without re-implementing Validate/Deploy.
func rsyncTargetFor(target Target, cfg dockyardConfig) Target {
	return Target{
		ID:     target.ID,
		SiteID: target.SiteID,
		Name:   target.Name,
		Kind:   "rsync",
		Config: map[string]any{
			"host":            cfg.Host,
			"user":            cfg.User,
			"port":            cfg.Port,
			"path":            cfg.HostPath,
			"private_key_pem": cfg.PrivateKeyPEM,
			"public_url":      "https://" + cfg.Domain,
		},
	}
}

// Validate enforces structural rules on a dockyard target before any network
// call. Delegates the SSH/rsync subset to RsyncDeployer so the rules stay in
// one place.
func (d *DockyardDeployer) Validate(target Target) error {
	cfg, err := parseDockyardConfig(target.Config)
	if err != nil {
		return err
	}

	if cfg.DockyardURL == "" {
		return errors.New("dockyard deploy: dockyard_url is required")
	}
	u, err := url.Parse(cfg.DockyardURL)
	if err != nil {
		return fmt.Errorf("dockyard deploy: dockyard_url %q is invalid: %w", cfg.DockyardURL, err)
	}
	if u.Scheme != "https" {
		return fmt.Errorf("dockyard deploy: dockyard_url must use https, got %q", u.Scheme)
	}
	if u.Host == "" {
		return errors.New("dockyard deploy: dockyard_url has no host")
	}
	if cfg.APIKey == "" {
		return errors.New("dockyard deploy: api_key is required")
	}
	if cfg.ServerID == "" {
		return errors.New("dockyard deploy: server_id is required")
	}
	if !uuidPattern.MatchString(cfg.ServerID) {
		return fmt.Errorf("dockyard deploy: server_id %q is not a valid UUID", cfg.ServerID)
	}
	if cfg.Domain == "" {
		return errors.New("dockyard deploy: domain is required")
	}
	if !hostnamePattern.MatchString(cfg.Domain) {
		return fmt.Errorf("dockyard deploy: domain %q is not a valid hostname", cfg.Domain)
	}
	if cfg.HostPath == "" {
		return errors.New("dockyard deploy: host_path is required")
	}
	if !strings.HasPrefix(cfg.HostPath, "/") {
		return fmt.Errorf("dockyard deploy: host_path %q must be absolute", cfg.HostPath)
	}

	// Reuse rsync validation for the SSH subset.
	rsyncT := rsyncTargetFor(target, cfg)
	if err := d.rsyncValidator().Validate(rsyncT); err != nil {
		return err
	}
	return nil
}

// rsyncValidator returns the embedded RsyncDeployer, lazily constructing one
// if NewDockyardDeployer wasn't used (e.g. via deploy.New).
func (d *DockyardDeployer) rsyncValidator() *RsyncDeployer {
	if d.rsync == nil {
		d.rsync = NewRsyncDeployer()
	}
	return d.rsync
}

// Deploy syncs the dist tree to the Dockyard-managed host then registers a
// Caddy proxy route so the configured domain serves those files.
//
// Order matters: rsync first, then route. If the route call fails after a
// successful sync the dist is up but unreachable; we surface that as an error
// so the caller can retry. Retry is safe because rsync re-syncs idempotently
// and a duplicate route returns 409, which we treat as success.
func (d *DockyardDeployer) Deploy(ctx context.Context, distDir string, target Target) (Result, error) {
	if err := d.Validate(target); err != nil {
		return Result{}, err
	}

	cfg, err := parseDockyardConfig(target.Config)
	if err != nil {
		return Result{}, err
	}

	// Step 1: rsync the dist tree.
	rsyncT := rsyncTargetFor(target, cfg)
	rsyncResult, err := d.rsyncValidator().Deploy(ctx, distDir, rsyncT)
	if err != nil {
		return Result{}, fmt.Errorf("dockyard deploy: rsync: %w", err)
	}

	// Step 2: register the Caddy proxy route + apply.
	client := d.dockyardClientFor(cfg)

	customConfig := fmt.Sprintf("root * %s\nfile_server", strings.TrimRight(cfg.HostPath, "/"))
	body := proxyRouteRequest{
		Domain:          cfg.Domain,
		UpstreamTarget:  "static-files",
		SSLMode:         "auto",
		ListenInterface: "public",
		CustomConfig:    customConfig,
	}

	if err := client.PutProxyRoute(ctx, cfg.ServerID, body); err != nil {
		slog.Warn("dockyard deploy: rsync succeeded but proxy route registration failed",
			"target_id", target.ID,
			"site_id", target.SiteID,
			"server_id", cfg.ServerID,
			"domain", cfg.Domain,
			"err", err,
		)
		return Result{}, fmt.Errorf("dockyard deploy: register proxy route: %w", err)
	}

	if err := client.ApplyProxy(ctx, cfg.ServerID); err != nil {
		slog.Warn("dockyard deploy: proxy route registered but apply failed",
			"target_id", target.ID,
			"site_id", target.SiteID,
			"server_id", cfg.ServerID,
			"err", err,
		)
		return Result{}, fmt.Errorf("dockyard deploy: apply proxy: %w", err)
	}

	return Result{
		URL:        "https://" + cfg.Domain,
		DeployedAt: time.Now(),
		SizeBytes:  rsyncResult.SizeBytes,
		FileCount:  rsyncResult.FileCount,
	}, nil
}

// dockyardClientFor returns the configured client or a default HTTP-backed
// one bound to this target's URL + API key.
func (d *DockyardDeployer) dockyardClientFor(cfg dockyardConfig) dockyardClient {
	if d.client != nil {
		return d.client
	}
	hc := d.httpClient
	if hc == nil {
		hc = &http.Client{Timeout: dockyardHTTPTimeout}
	}
	return &httpDockyardClient{
		base:   strings.TrimRight(cfg.DockyardURL, "/"),
		apiKey: cfg.APIKey,
		hc:     hc,
	}
}

// httpDockyardClient is the production implementation of dockyardClient.
type httpDockyardClient struct {
	base   string
	apiKey string
	hc     *http.Client
}

// PutProxyRoute POSTs to /api/servers/{serverID}/proxy/routes. 200/201/409
// are treated as success (409 means the route already exists).
func (c *httpDockyardClient) PutProxyRoute(ctx context.Context, serverID string, body proxyRouteRequest) error {
	endpoint := fmt.Sprintf("%s/api/servers/%s/proxy/routes", c.base, url.PathEscape(serverID))
	payload, err := json.Marshal(body)
	if err != nil {
		return fmt.Errorf("encode body: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(payload))
	if err != nil {
		return fmt.Errorf("build request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-API-Key", c.apiKey)

	resp, err := c.hc.Do(req)
	if err != nil {
		return fmt.Errorf("post proxy route: %w", err)
	}
	defer resp.Body.Close()

	switch resp.StatusCode {
	case http.StatusOK, http.StatusCreated, http.StatusConflict:
		return nil
	default:
		respBody, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return fmt.Errorf("post proxy route: status %d: %s",
			resp.StatusCode, strings.TrimSpace(string(respBody)))
	}
}

// ApplyProxy POSTs to /api/servers/{serverID}/proxy/apply.
func (c *httpDockyardClient) ApplyProxy(ctx context.Context, serverID string) error {
	endpoint := fmt.Sprintf("%s/api/servers/%s/proxy/apply", c.base, url.PathEscape(serverID))
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, nil)
	if err != nil {
		return fmt.Errorf("build request: %w", err)
	}
	req.Header.Set("X-API-Key", c.apiKey)

	resp, err := c.hc.Do(req)
	if err != nil {
		return fmt.Errorf("post proxy apply: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		return nil
	}
	respBody, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
	return fmt.Errorf("post proxy apply: status %d: %s",
		resp.StatusCode, strings.TrimSpace(string(respBody)))
}
