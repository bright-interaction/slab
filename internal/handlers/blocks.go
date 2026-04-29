package handlers

import (
	"database/sql"
	"encoding/json"
	"net/http"

	"github.com/brightinteraction/atomicsite/internal/builder"
	"github.com/brightinteraction/atomicsite/internal/config"
	"github.com/brightinteraction/atomicsite/internal/store"
)

type BlockHandler struct {
	cfg     *config.Config
	queries *store.Queries
	db      *sql.DB
}

func NewBlockHandler(cfg *config.Config, queries *store.Queries, db *sql.DB) *BlockHandler {
	return &BlockHandler{cfg: cfg, queries: queries, db: db}
}

func (h *BlockHandler) List(w http.ResponseWriter, r *http.Request) {
	pageID := urlParam(r, "pageID")
	blocks, err := h.queries.ListBlocksByPage(r.Context(), pageID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to list blocks")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"blocks": blocks})
}

func (h *BlockHandler) Get(w http.ResponseWriter, r *http.Request) {
	block, err := h.queries.GetBlockByID(r.Context(), urlParam(r, "blockID"))
	if err != nil {
		writeError(w, http.StatusNotFound, "Block not found")
		return
	}
	writeJSON(w, http.StatusOK, block)
}

func (h *BlockHandler) Create(w http.ResponseWriter, r *http.Request) {
	pageID := urlParam(r, "pageID")

	var req struct {
		BlockType string          `json:"block_type"`
		Data      json.RawMessage `json:"data"`
		Style     json.RawMessage `json:"style"`
		SortOrder *int64          `json:"sort_order"`
	}
	if err := parseJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid request body")
		return
	}
	if req.BlockType == "" {
		writeError(w, http.StatusBadRequest, "block_type is required")
		return
	}

	dataJSON := "{}"
	if len(req.Data) > 0 {
		dataJSON = string(req.Data)
	}
	styleJSON := "{}"
	if len(req.Style) > 0 {
		styleJSON = string(req.Style)
	}

	sortOrder := int64(0)
	if req.SortOrder != nil {
		sortOrder = *req.SortOrder
	}

	id := newID()
	if err := h.queries.CreateBlock(r.Context(), store.CreateBlockParams{
		ID:        id,
		PageID:    pageID,
		BlockType: req.BlockType,
		SortOrder: sortOrder,
		DataJson:  dataJSON,
		StyleJson: styleJSON,
		IsVisible: 1,
	}); err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to create block")
		return
	}

	block, _ := h.queries.GetBlockByID(r.Context(), id)
	writeJSON(w, http.StatusCreated, block)
}

func (h *BlockHandler) Update(w http.ResponseWriter, r *http.Request) {
	blockID := urlParam(r, "blockID")
	existing, err := h.queries.GetBlockByID(r.Context(), blockID)
	if err != nil {
		writeError(w, http.StatusNotFound, "Block not found")
		return
	}

	var req struct {
		BlockType *string          `json:"block_type"`
		Data      *json.RawMessage `json:"data"`
		Style     *json.RawMessage `json:"style"`
		SortOrder *int64           `json:"sort_order"`
		IsVisible *bool            `json:"is_visible"`
	}
	if err := parseJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	params := store.UpdateBlockParams{
		BlockType: existing.BlockType,
		SortOrder: existing.SortOrder,
		DataJson:  existing.DataJson,
		StyleJson: existing.StyleJson,
		IsVisible: existing.IsVisible,
		ID:        blockID,
	}

	if req.BlockType != nil {
		params.BlockType = *req.BlockType
	}
	if req.Data != nil {
		params.DataJson = string(*req.Data)
	}
	if req.Style != nil {
		params.StyleJson = string(*req.Style)
	}
	if req.SortOrder != nil {
		params.SortOrder = *req.SortOrder
	}
	if req.IsVisible != nil {
		if *req.IsVisible {
			params.IsVisible = 1
		} else {
			params.IsVisible = 0
		}
	}

	if err := h.queries.UpdateBlock(r.Context(), params); err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to update block")
		return
	}

	block, _ := h.queries.GetBlockByID(r.Context(), blockID)
	writeJSON(w, http.StatusOK, block)
}

func (h *BlockHandler) Delete(w http.ResponseWriter, r *http.Request) {
	blockID := urlParam(r, "blockID")
	if _, err := h.queries.GetBlockByID(r.Context(), blockID); err != nil {
		writeError(w, http.StatusNotFound, "Block not found")
		return
	}

	if err := h.queries.DeleteBlock(r.Context(), blockID); err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to delete block")
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "deleted"})
}

func (h *BlockHandler) Reorder(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Order []struct {
			ID        string `json:"id"`
			SortOrder int64  `json:"sort_order"`
		} `json:"order"`
	}
	if err := parseJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	for _, item := range req.Order {
		_ = h.queries.UpdateBlockOrder(r.Context(), store.UpdateBlockOrderParams{
			SortOrder: item.SortOrder,
			ID:        item.ID,
		})
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

// Preview returns the rendered Astro source for a single block. Used by
// the per-block "</>" code toggle in the admin so developers can see
// exactly what each block produces without triggering a full build.
// Defends against cross-site fetches by checking the block's page belongs
// to the requested site.
func (h *BlockHandler) Preview(w http.ResponseWriter, r *http.Request) {
	siteID := urlParam(r, "siteID")
	blockID := urlParam(r, "blockID")

	block, err := h.queries.GetBlockByID(r.Context(), blockID)
	if err != nil {
		writeError(w, http.StatusNotFound, "Block not found")
		return
	}
	page, err := h.queries.GetPageByID(r.Context(), block.PageID)
	if err != nil || page.SiteID != siteID {
		writeError(w, http.StatusNotFound, "Block not found")
		return
	}

	src, err := builder.RenderSingleBlock(r.Context(), h.queries, siteID, blockID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to render block")
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"block_id":   blockID,
		"block_type": block.BlockType,
		"page_id":    block.PageID,
		"astro":      src,
	})
}

// BulkSave atomically replaces all blocks on a page.
func (h *BlockHandler) BulkSave(w http.ResponseWriter, r *http.Request) {
	pageID := urlParam(r, "pageID")

	var req struct {
		Blocks []struct {
			ID        string          `json:"id"`
			BlockType string          `json:"block_type"`
			SortOrder int64           `json:"sort_order"`
			Data      json.RawMessage `json:"data"`
			Style     json.RawMessage `json:"style"`
			IsVisible bool            `json:"is_visible"`
		} `json:"blocks"`
	}
	if err := parseJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	tx, err := h.db.Begin()
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to begin transaction")
		return
	}
	defer tx.Rollback()

	qtx := h.queries.WithTx(tx)

	// Delete existing blocks
	if err := qtx.DeleteBlocksByPage(r.Context(), pageID); err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to clear blocks")
		return
	}

	// Insert new blocks
	for _, b := range req.Blocks {
		id := b.ID
		if id == "" {
			id = newID()
		}

		dataJSON := "{}"
		if len(b.Data) > 0 {
			dataJSON = string(b.Data)
		}
		styleJSON := "{}"
		if len(b.Style) > 0 {
			styleJSON = string(b.Style)
		}

		visible := int64(0)
		if b.IsVisible {
			visible = 1
		}

		if err := qtx.CreateBlock(r.Context(), store.CreateBlockParams{
			ID:        id,
			PageID:    pageID,
			BlockType: b.BlockType,
			SortOrder: b.SortOrder,
			DataJson:  dataJSON,
			StyleJson: styleJSON,
			IsVisible: visible,
		}); err != nil {
			writeError(w, http.StatusInternalServerError, "Failed to create block")
			return
		}
	}

	if err := tx.Commit(); err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to commit")
		return
	}

	blocks, _ := h.queries.ListBlocksByPage(r.Context(), pageID)
	writeJSON(w, http.StatusOK, map[string]any{"blocks": blocks})
}
