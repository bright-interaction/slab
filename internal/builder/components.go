package builder

import (
	"context"
	"fmt"
	"log/slog"
	"path/filepath"
	"strings"

	"github.com/bright-interaction/atomicsite/internal/store"
)

// RenderComponents generates component files from the components table. The
// file extension is picked by sniffing the template body: Svelte 5 rune
// syntax (`$state(`, `$props(`, `$derived(`, `$effect(`) means a .svelte
// file; everything else stays .astro (the historical default). This keeps
// existing Astro components untouched while enabling Svelte components for
// any new template that uses runes. Adding the @astrojs/svelte integration
// in the skeleton is what makes Astro pick up the .svelte files at build.
func RenderComponents(ctx context.Context, queries *store.Queries, siteID string, wsDir string) error {
	components, err := queries.ListComponentsBySite(ctx, siteID)
	if err != nil {
		return fmt.Errorf("list components: %w", err)
	}

	for _, comp := range components {
		// Path-traversal guard: comp.Name becomes a filename under
		// src/components/. A name like "../pages/index" would escape the
		// directory and overwrite other workspace files, so skip anything that
		// is not a safe identifier (write tools reject these too; this is the
		// last-line guard at the actual file write).
		if !ValidComponentName(comp.Name) {
			slog.Warn("builder: skipping component with unsafe name", "site_id", siteID, "name", comp.Name)
			continue
		}
		ext := pickComponentExt(comp.Template)
		path := filepath.Join(wsDir, "src", "components", comp.Name+ext)
		if err := WriteFile(path, comp.Template); err != nil {
			return fmt.Errorf("write component %s: %w", comp.Name, err)
		}
	}

	return nil
}

// pickComponentExt returns ".svelte" if the template uses Svelte 5 rune
// syntax, ".astro" otherwise. The rune sigils are diagnostic: Astro
// frontmatter has no `$state(`, `$props(`, `$derived(`, `$effect(` calls
// (the `$` is reserved for Astro.glob and similar APIs but never preceded
// by these specific identifiers).
func pickComponentExt(template string) string {
	runes := []string{"$state(", "$props(", "$derived(", "$effect(", "$bindable(", "$inspect("}
	for _, r := range runes {
		if strings.Contains(template, r) {
			return ".svelte"
		}
	}
	return ".astro"
}
