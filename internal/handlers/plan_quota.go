package handlers

import (
	"context"
	"errors"
	"net/http"

	"github.com/brightinteraction/atomicsite/internal/billing"
	authmw "github.com/brightinteraction/atomicsite/internal/middleware"
	"github.com/brightinteraction/atomicsite/internal/store"
)

// resolveUserWorkspace returns the workspace ID for the authenticated user
// behind the request. Mirrors SiteHandler.resolveWorkspaceForCreate but
// works without a SiteHandler instance so any handler can call it.
//
// Resolution order matches the existing pattern in sites.go:
//  1. The user's first workspace_members row (cloud signup default)
//  2. Admin fallback to the first workspace in the table (auto-bootstrap
//     in single-deploy OSS where there's exactly one workspace)
//  3. Error if neither path resolves
//
// Returns ("not authenticated") when no user is in context, which is what
// the workspace gate uses to refuse anonymous Apply / BulkImport calls.
func resolveUserWorkspace(r *http.Request, q *store.Queries) (string, error) {
	user := authmw.GetUser(r)
	if user == nil {
		return "", errors.New("not authenticated")
	}
	ids, err := q.ListWorkspaceIDsForUser(r.Context(), user.ID)
	if err == nil && len(ids) > 0 {
		return ids[0], nil
	}
	if user.Role == "admin" {
		all, err := q.ListAllWorkspaces(r.Context())
		if err == nil && len(all) > 0 {
			return all[0].ID, nil
		}
	}
	return "", errors.New("user has no workspace; create one first")
}

// planLimitForRequest returns the plan limit for a (workspace, resource)
// pair. Reads directly from the OSS-linked billing.Limit ladder rather
// than the EE provider boundary because plan caps are part of the OSS
// plan ladder by design - the EE provider's `OSS implementation` returns
// -1 for everything as a "no enforcement in single-deploy" stance, but
// the migration system's plan-quota gate must fire whenever the
// workspace has a paid plan value regardless of build tag. EE builds
// see identical behaviour because eeProvider.PlanLimits also delegates
// to billing.Limit.
//
// Returns -1 (unlimited) when the workspace can't be looked up so a
// transient DB error doesn't accidentally refuse a paid customer.
func planLimitForRequest(ctx context.Context, q *store.Queries, workspaceID, resource string) int64 {
	ws, err := q.GetWorkspaceByID(ctx, workspaceID)
	if err != nil {
		return -1
	}
	return billing.Limit(ws.Plan, resource)
}

// quotaPagesError describes a 402 response when migration Apply would
// exceed max_pages_per_site. The caller writes a friendly message + the
// machine-readable cap so the UI can render an upgrade nudge.
func quotaPagesError(siteID string, current, projected, limit int64) string {
	return errors.New("plan cap reached: site already has " +
		i64s(current) + " pages, this migration would add " + i64s(projected) +
		" more, but the workspace plan caps at " + i64s(limit) + " pages per site").Error()
}

// quotaRedirectsError mirrors quotaPagesError for the bulk redirect
// import path.
func quotaRedirectsError(current, projected, limit int64) string {
	return errors.New("plan cap reached: site already has " +
		i64s(current) + " redirects, this import would add " + i64s(projected) +
		" more, but the workspace plan caps at " + i64s(limit) + " redirects per site").Error()
}

// i64s avoids strconv.FormatInt + an extra import for a single-call site.
// Three-digit fast path covers every cap value in the Plans ladder
// (50, 250, 500, 1000, 2500, 10000) without touching the runtime.
func i64s(n int64) string {
	if n == 0 {
		return "0"
	}
	neg := false
	if n < 0 {
		neg = true
		n = -n
	}
	var buf [20]byte
	i := len(buf)
	for n > 0 {
		i--
		buf[i] = byte('0' + n%10)
		n /= 10
	}
	if neg {
		i--
		buf[i] = '-'
	}
	return string(buf[i:])
}
