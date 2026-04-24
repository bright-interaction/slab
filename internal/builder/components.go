package builder

import (
	"context"
	"fmt"
	"path/filepath"

	"github.com/brightinteraction/atomicsite/internal/store"
)

// RenderComponents generates .astro component files from the components table.
func RenderComponents(ctx context.Context, queries *store.Queries, siteID string, wsDir string) error {
	components, err := queries.ListComponentsBySite(ctx, siteID)
	if err != nil {
		return fmt.Errorf("list components: %w", err)
	}

	for _, comp := range components {
		path := filepath.Join(wsDir, "src", "components", comp.Name+".astro")
		if err := WriteFile(path, comp.Template); err != nil {
			return fmt.Errorf("write component %s: %w", comp.Name, err)
		}
	}

	return nil
}
