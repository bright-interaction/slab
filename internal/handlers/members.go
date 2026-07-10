package handlers

import (
	"net/http"

	"github.com/bright-interaction/atomicsite/internal/config"
	authmw "github.com/bright-interaction/atomicsite/internal/middleware"
	"github.com/bright-interaction/atomicsite/internal/store"
)

// MembersHandler handles workspace member management. Single-tenant: every
// authenticated user belongs to the same implicit workspace. Roles: admin,
// editor. Only admins can mutate members or roles.
type MembersHandler struct {
	cfg     *config.Config
	queries *store.Queries
}

func NewMembersHandler(cfg *config.Config, queries *store.Queries) *MembersHandler {
	return &MembersHandler{cfg: cfg, queries: queries}
}

type memberView struct {
	ID        string `json:"id"`
	Email     string `json:"email"`
	Name      string `json:"name"`
	Role      string `json:"role"`
	CreatedAt string `json:"created_at"`
	IsSelf    bool   `json:"is_self"`
}

func (h *MembersHandler) List(w http.ResponseWriter, r *http.Request) {
	me := authmw.GetUser(r)
	rows, err := h.queries.ListUsers(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to list members")
		return
	}
	out := make([]memberView, 0, len(rows))
	for _, u := range rows {
		out = append(out, memberView{
			ID:        u.ID,
			Email:     u.Email,
			Name:      u.Name,
			Role:      u.Role,
			CreatedAt: u.CreatedAt,
			IsSelf:    me != nil && me.ID == u.ID,
		})
	}
	writeJSON(w, http.StatusOK, out)
}

func (h *MembersHandler) UpdateRole(w http.ResponseWriter, r *http.Request) {
	me := authmw.GetUser(r)
	id := urlParam(r, "userID")
	if me != nil && me.ID == id {
		writeError(w, http.StatusBadRequest, "Cannot change your own role")
		return
	}
	var req struct {
		Role string `json:"role"`
	}
	if err := parseJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid request body")
		return
	}
	if req.Role != "admin" && req.Role != "editor" {
		writeError(w, http.StatusBadRequest, "Role must be admin or editor")
		return
	}
	if _, err := h.queries.GetUserByID(r.Context(), id); err != nil {
		writeError(w, http.StatusNotFound, "Member not found")
		return
	}
	if err := h.queries.UpdateUserRole(r.Context(), store.UpdateUserRoleParams{
		Role: req.Role,
		ID:   id,
	}); err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to update role")
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (h *MembersHandler) Delete(w http.ResponseWriter, r *http.Request) {
	me := authmw.GetUser(r)
	id := urlParam(r, "userID")
	if me != nil && me.ID == id {
		writeError(w, http.StatusBadRequest, "Cannot delete your own account")
		return
	}
	target, err := h.queries.GetUserByID(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusNotFound, "Member not found")
		return
	}
	if err := h.queries.DeleteUser(r.Context(), id); err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to delete member")
		return
	}
	AuditLog(r.Context(), h.queries, r, "", AuditActionDelete, "member", id, map[string]any{
		"email": target.Email,
		"role":  target.Role,
		"name":  target.Name,
	})
	writeJSON(w, http.StatusOK, map[string]string{"status": "deleted"})
}

// UpdateProfile handles PATCH /api/auth/me - lets the current user rename
// themselves. Email and role changes go through different endpoints (or are
// admin-only).
func (h *MembersHandler) UpdateProfile(w http.ResponseWriter, r *http.Request) {
	me := authmw.GetUser(r)
	if me == nil {
		writeError(w, http.StatusUnauthorized, "Not authenticated")
		return
	}
	var req struct {
		Name string `json:"name"`
	}
	if err := parseJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid request body")
		return
	}
	if err := h.queries.UpdateUserName(r.Context(), store.UpdateUserNameParams{
		Name: req.Name,
		ID:   me.ID,
	}); err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to update profile")
		return
	}
	row, err := h.queries.GetUserByID(r.Context(), me.ID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to reload profile")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"user": map[string]any{
			"id":         row.ID,
			"email":      row.Email,
			"name":       row.Name,
			"role":       row.Role,
			"created_at": row.CreatedAt,
			"updated_at": row.UpdatedAt,
		},
	})
}
