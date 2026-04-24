package builder

import (
	"bytes"
	"fmt"
	"log/slog"
	"os/exec"
	"path/filepath"
	"time"
)

// CompileResult holds the output of a build.
type CompileResult struct {
	DistDir    string
	BuildLog   string
	DurationMs int64
	Success    bool
	Error      string
}

// Compile runs bun install + bun run build in the workspace directory.
func Compile(wsDir string) *CompileResult {
	start := time.Now()
	var logBuf bytes.Buffer

	// Step 1: bun install
	slog.Info("build: installing dependencies", "workspace", wsDir)
	installCmd := exec.Command("bun", "install", "--frozen-lockfile")
	installCmd.Dir = wsDir
	installOut, err := installCmd.CombinedOutput()
	logBuf.WriteString("=== bun install ===\n")
	logBuf.Write(installOut)
	logBuf.WriteString("\n")

	if err != nil {
		// Retry without frozen lockfile (first build has no lockfile)
		installCmd = exec.Command("bun", "install")
		installCmd.Dir = wsDir
		installOut, err = installCmd.CombinedOutput()
		logBuf.WriteString("=== bun install (no lockfile) ===\n")
		logBuf.Write(installOut)
		logBuf.WriteString("\n")
		if err != nil {
			return &CompileResult{
				BuildLog:   logBuf.String(),
				DurationMs: time.Since(start).Milliseconds(),
				Error:      fmt.Sprintf("bun install failed: %v", err),
			}
		}
	}

	// Step 2: bun run build
	slog.Info("build: compiling astro project", "workspace", wsDir)
	buildCmd := exec.Command("bun", "run", "build")
	buildCmd.Dir = wsDir
	buildOut, err := buildCmd.CombinedOutput()
	logBuf.WriteString("=== bun run build ===\n")
	logBuf.Write(buildOut)
	logBuf.WriteString("\n")

	if err != nil {
		return &CompileResult{
			BuildLog:   logBuf.String(),
			DurationMs: time.Since(start).Milliseconds(),
			Error:      fmt.Sprintf("astro build failed: %v", err),
		}
	}

	distDir := filepath.Join(wsDir, "dist")
	duration := time.Since(start).Milliseconds()

	slog.Info("build: complete", "duration_ms", duration, "dist", distDir)

	return &CompileResult{
		DistDir:    distDir,
		BuildLog:   logBuf.String(),
		DurationMs: duration,
		Success:    true,
	}
}
