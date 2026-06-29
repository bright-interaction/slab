package builder

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// secretEnvNames are server secrets that must never be inherited by a build
// subprocess (bun install / astro build run tenant-influenced JS and dependency
// scripts). Kept alongside a suffix heuristic so a future secret-shaped var is
// stripped by default.
var secretEnvNames = map[string]bool{
	"ADMIN_PASSWORD": true, "JWT_SECRET": true, "VAULT_KEY": true,
	"ATOMICSITE_SHIELD_KEY": true, "ANALYTICS_SALT": true,
	"MOLLIE_API_KEY": true, "MAILERSEND_API_TOKEN": true,
	"ATOMICSITE_CLOUDFLARE_TOKEN": true, "CLOUDFLARE_TOKEN": true,
	"SESSION_SECRET": true, "DATABASE_URL": true, "DB_PATH": true,
}

func isSecretEnvName(name string) bool {
	if secretEnvNames[name] {
		return true
	}
	u := strings.ToUpper(name)
	for _, suffix := range []string{"_SECRET", "_TOKEN", "_API_KEY", "_APIKEY", "_PASSWORD", "_PASS", "_PRIVATE_KEY", "_CREDENTIAL", "_CREDENTIALS"} {
		if strings.HasSuffix(u, suffix) {
			return true
		}
	}
	return false
}

// buildEnv returns the server environment with secrets stripped, so the build
// (and any transitive dependency script) cannot read server secrets to
// exfiltrate them. A denylist preserves the toolchain variables the build
// legitimately needs (PATH/HOME/caches/locale) while removing anything
// secret-shaped.
func buildEnv() []string {
	src := os.Environ()
	env := make([]string, 0, len(src))
	for _, kv := range src {
		name := kv
		if i := strings.IndexByte(kv, '='); i >= 0 {
			name = kv[:i]
		}
		if isSecretEnvName(name) {
			continue
		}
		env = append(env, kv)
	}
	return env
}

// BuildTimeout caps a single end-to-end Compile() (bun install + bun run
// build). 10 minutes is generous for a fresh install on a multi-page site
// with image processing and tight enough that a deadlocked bun process
// doesn't pin a goroutine forever (audit finding H4).
//
// The timeout is enforced via exec.CommandContext: when the deadline hits,
// the bun process gets a SIGKILL and the goroutine returns with an error.
// Without this, a single hung bun could pile up unbounded goroutines and
// memory across repeated build retries.
const BuildTimeout = 10 * time.Minute

// MaxBuildLogBytes caps the captured build log so a particularly verbose
// bun error doesn't blow up memory or the deployments.build_log column
// (audit finding L2). 1 MB covers any plausible Astro / Vite stack trace
// while bounding the worst case.
const MaxBuildLogBytes = 1024 * 1024

// CompileResult holds the output of a build.
type CompileResult struct {
	DistDir    string
	BuildLog   string
	DurationMs int64
	Success    bool
	Error      string
}

// Compile runs bun install + bun run build in the workspace directory.
// The supplied ctx bounds the operation: cancel ctx and the bun child
// processes get killed via exec.CommandContext. If ctx has no deadline
// of its own, we add BuildTimeout so a forgotten context.Background()
// caller still gets the safety net.
func Compile(ctx context.Context, wsDir string) *CompileResult {
	if ctx == nil {
		ctx = context.Background()
	}
	if _, hasDeadline := ctx.Deadline(); !hasDeadline {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, BuildTimeout)
		defer cancel()
	}

	start := time.Now()
	var logBuf bytes.Buffer

	// Step 1: bun install. Try frozen-lockfile first; fall back without
	// when the workspace has no bun.lock yet (first build of a site).
	slog.Info("build: installing dependencies", "workspace", wsDir)
	installCmd := exec.CommandContext(ctx, "bun", "install", "--frozen-lockfile")
	installCmd.Dir = wsDir
	installCmd.Env = buildEnv()
	installOut, err := installCmd.CombinedOutput()
	appendLogSection(&logBuf, "bun install", installOut)

	if err != nil {
		if ctxErr := ctx.Err(); ctxErr != nil {
			return failureResult(&logBuf, start, fmt.Sprintf("bun install timed out or cancelled: %v", ctxErr))
		}
		installCmd = exec.CommandContext(ctx, "bun", "install")
		installCmd.Dir = wsDir
		installCmd.Env = buildEnv()
		installOut, err = installCmd.CombinedOutput()
		appendLogSection(&logBuf, "bun install (no lockfile)", installOut)
		if err != nil {
			if ctxErr := ctx.Err(); ctxErr != nil {
				return failureResult(&logBuf, start, fmt.Sprintf("bun install timed out or cancelled: %v", ctxErr))
			}
			return failureResult(&logBuf, start, fmt.Sprintf("bun install failed: %v", err))
		}
	}

	// Step 2: bun run build. Honour the same deadline.
	slog.Info("build: compiling astro project", "workspace", wsDir)
	buildCmd := exec.CommandContext(ctx, "bun", "run", "build")
	buildCmd.Dir = wsDir
	buildCmd.Env = buildEnv()
	buildOut, err := buildCmd.CombinedOutput()
	appendLogSection(&logBuf, "bun run build", buildOut)

	if err != nil {
		if ctxErr := ctx.Err(); ctxErr != nil {
			return failureResult(&logBuf, start, fmt.Sprintf("astro build timed out or cancelled: %v", ctxErr))
		}
		return failureResult(&logBuf, start, fmt.Sprintf("astro build failed: %v", err))
	}

	distDir := filepath.Join(wsDir, "dist")
	duration := time.Since(start).Milliseconds()

	slog.Info("build: complete", "duration_ms", duration, "dist", distDir)

	return &CompileResult{
		DistDir:    distDir,
		BuildLog:   capLog(logBuf.String()),
		DurationMs: duration,
		Success:    true,
	}
}

// failureResult assembles a CompileResult for the error path. Caps the
// log size; otherwise a misbehaving bun could blow up memory.
func failureResult(logBuf *bytes.Buffer, start time.Time, msg string) *CompileResult {
	return &CompileResult{
		BuildLog:   capLog(logBuf.String()),
		DurationMs: time.Since(start).Milliseconds(),
		Error:      msg,
	}
}

// appendLogSection writes a labelled section to the running buffer.
// Truncates each section's contribution so a runaway log line can't blow
// past MaxBuildLogBytes on its own.
func appendLogSection(buf *bytes.Buffer, label string, out []byte) {
	buf.WriteString("=== ")
	buf.WriteString(label)
	buf.WriteString(" ===\n")
	if len(out) > MaxBuildLogBytes {
		buf.Write(out[:MaxBuildLogBytes])
		buf.WriteString("\n...[truncated]\n")
	} else {
		buf.Write(out)
	}
	buf.WriteString("\n")
}

// capLog clamps the final log string to MaxBuildLogBytes. The truncation
// keeps the head + tail (most useful debugging context) and drops the
// middle when a build produces a multi-MB log.
func capLog(s string) string {
	if len(s) <= MaxBuildLogBytes {
		return s
	}
	half := MaxBuildLogBytes / 2
	return s[:half] + "\n...[truncated middle]...\n" + s[len(s)-half:]
}

// ErrBuildCancelled is returned via CompileResult.Error when the supplied
// context was cancelled (timeout, parent-context kill). Callers can match
// on this prefix to distinguish "build legitimately failed" from "build
// got killed by us".
var ErrBuildCancelled = errors.New("build cancelled")

// PagefindTimeout caps a single pagefind index pass. Sites with 1000+
// pages take a few seconds; 5 minutes is comfortably above the worst
// case we'd want to wait on a build.
const PagefindTimeout = 5 * time.Minute

// RunPagefind generates the static search index for the built site by
// invoking pagefind against dist/. Sprint 5 quick-win (2026-05-24).
// The output lands in dist/pagefind/ and the search_box block's
// runtime loader points at it. Opt-in: callers gate this behind the
// search.pagefind_enabled setting.
//
// pagefind is declared in the workspace skeleton's devDependencies so
// `bun install` in Compile() already fetched it. We invoke via bunx
// to pick up the local copy; no global install required on the
// builder host.
//
// Returns the captured log lines so the renderer can append them to
// the deployment's build_log column, and a single error when pagefind
// itself returns non-zero. Cancellation flows through ctx.
func RunPagefind(ctx context.Context, wsDir string) (string, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	if _, hasDeadline := ctx.Deadline(); !hasDeadline {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, PagefindTimeout)
		defer cancel()
	}
	slog.Info("build: running pagefind index", "workspace", wsDir)
	cmd := exec.CommandContext(ctx, "bunx", "pagefind", "--site", "dist", "--quiet")
	cmd.Dir = wsDir
	out, err := cmd.CombinedOutput()
	if err != nil {
		if ctxErr := ctx.Err(); ctxErr != nil {
			return string(out), fmt.Errorf("pagefind timed out or cancelled: %w", ctxErr)
		}
		return string(out), fmt.Errorf("pagefind failed: %w", err)
	}
	return string(out), nil
}
