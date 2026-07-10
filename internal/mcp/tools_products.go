package mcp

import (
	"context"
	"encoding/json"
	"errors"
	"strings"

	"github.com/bright-interaction/atomicsite/internal/handlers"
	authmw "github.com/bright-interaction/atomicsite/internal/middleware"
	"github.com/bright-interaction/atomicsite/internal/store"
)

// registerProductTools wires the catalog MCP surface (Sprint 2 slice A
// of the WP/Webflow replacement roadmap):
//
//   - list_products: enumerate the site's catalog
//   - get_product: fetch a single product + its variants
//   - create_product: scaffold a draft product
//   - update_product: edit name/slug/price/status/images
//   - delete_product: remove (cascades to variants)
//   - create_variant: add a SKU under a product
//   - update_variant: edit price/inventory/attributes
//   - delete_variant: remove
//   - set_inventory: write absolute count OR a delta, with reason
//
// All write tools are RequiresWrite. Nil handler disables the tools
// with a clean "not configured" error.
func (s *Server) registerProductTools() {
	register := func(t Tool) { s.tools[t.Name] = t }
	if s.products == nil {
		// Without the handler the tools cannot operate; the tools/list
		// surface still mentions them with a configured-elsewhere error
		// so the agent's planning loop is informed.
	}

	register(Tool{
		Name:        "list_products",
		Description: "Lists the site's catalog (Sprint 2 / slice A of the WP/Webflow roadmap). Returns id, name, slug, status (draft|active|archived), base_price_cents, currency, category, sort_order. Use get_product to fetch a single product plus its variants.",
		InputSchema: schema(`{
			"type":"object",
			"properties":{}
		}`),
		Handler: func(ctx context.Context, agent *authmw.AgentIdentity, raw json.RawMessage) (string, error) {
			if s.products == nil {
				return "", errors.New("list_products not configured on this server")
			}
			rows, err := s.queries.ListProductsBySite(ctx, store.ListProductsBySiteParams{
				SiteID: agent.SiteID, Limit: 500,
			})
			if err != nil {
				return "", err
			}
			return mustJSON(map[string]any{"products": rows, "count": len(rows)}), nil
		},
	})

	register(Tool{
		Name:        "get_product",
		Description: "Returns a single product plus all variants. Required: product_id.",
		InputSchema: schema(`{
			"type":"object",
			"properties":{"product_id":{"type":"string"}},
			"required":["product_id"]
		}`),
		Handler: func(ctx context.Context, agent *authmw.AgentIdentity, raw json.RawMessage) (string, error) {
			if s.products == nil {
				return "", errors.New("get_product not configured")
			}
			var args struct {
				ProductID string `json:"product_id"`
			}
			if err := json.Unmarshal(raw, &args); err != nil {
				return "", err
			}
			p, err := s.queries.GetProductByID(ctx, store.GetProductByIDParams{
				ID: args.ProductID, SiteID: agent.SiteID,
			})
			if err != nil {
				return "", errors.New("product not found")
			}
			variants, _ := s.queries.ListProductVariants(ctx, args.ProductID)
			return mustJSON(map[string]any{"product": p, "variants": variants}), nil
		},
	})

	register(Tool{
		Name:        "create_product",
		Description: "Scaffolds a new product. Required: name (1-200 chars), slug (lowercase alphanumeric / dash / underscore, 1-80 chars). Optional: description, category, status ('draft' default, 'active' to publish, 'archived'), images (array of media IDs or URLs), base_price_cents (>=0), currency (ISO 4217, EUR default), requires_shipping (default true), tax_class (default 'standard'), weight_grams, sort_order.",
		InputSchema: schema(`{
			"type":"object",
			"properties":{
				"name":{"type":"string"},
				"slug":{"type":"string"},
				"description":{"type":"string"},
				"category":{"type":"string"},
				"status":{"type":"string","enum":["draft","active","archived"]},
				"images":{"type":"array","items":{"type":"string"}},
				"base_price_cents":{"type":"integer","minimum":0},
				"currency":{"type":"string"},
				"requires_shipping":{"type":"boolean"},
				"tax_class":{"type":"string"},
				"weight_grams":{"type":"integer","minimum":0},
				"sort_order":{"type":"integer"}
			},
			"required":["name","slug"]
		}`),
		RequiresWrite: true,
		Handler: func(ctx context.Context, agent *authmw.AgentIdentity, raw json.RawMessage) (string, error) {
			if s.products == nil {
				return "", errors.New("create_product not configured")
			}
			var in handlers.ProductInput
			if err := json.Unmarshal(raw, &in); err != nil {
				return "", err
			}
			created, err := s.products.CreateForAgent(ctx, agent.SiteID, in)
			if err != nil {
				return "", err
			}
			return mustJSON(map[string]any{"product": created}), nil
		},
	})

	register(Tool{
		Name:        "update_product",
		Description: "Updates a product. Required: product_id. Any of name/slug/description/category/status/images/base_price_cents/currency/requires_shipping/tax_class/weight_grams/sort_order can be omitted to leave unchanged.",
		InputSchema: schema(`{
			"type":"object",
			"properties":{
				"product_id":{"type":"string"},
				"name":{"type":"string"},
				"slug":{"type":"string"},
				"description":{"type":"string"},
				"category":{"type":"string"},
				"status":{"type":"string","enum":["draft","active","archived"]},
				"images":{"type":"array","items":{"type":"string"}},
				"base_price_cents":{"type":"integer","minimum":0},
				"currency":{"type":"string"},
				"requires_shipping":{"type":"boolean"},
				"tax_class":{"type":"string"},
				"weight_grams":{"type":"integer","minimum":0},
				"sort_order":{"type":"integer"}
			},
			"required":["product_id"]
		}`),
		RequiresWrite: true,
		Handler: func(ctx context.Context, agent *authmw.AgentIdentity, raw json.RawMessage) (string, error) {
			if s.products == nil {
				return "", errors.New("update_product not configured")
			}
			var args struct {
				ProductID string `json:"product_id"`
			}
			if err := json.Unmarshal(raw, &args); err != nil {
				return "", err
			}
			updated, err := s.products.UpdateForAgent(ctx, agent.SiteID, args.ProductID, raw)
			if err != nil {
				return "", err
			}
			return mustJSON(map[string]any{"product": updated}), nil
		},
	})

	register(Tool{
		Name:        "delete_product",
		Description: "Deletes a product. Cascades to variants and inventory adjustments. Required: product_id.",
		InputSchema: schema(`{
			"type":"object",
			"properties":{"product_id":{"type":"string"}},
			"required":["product_id"]
		}`),
		RequiresWrite: true,
		Handler: func(ctx context.Context, agent *authmw.AgentIdentity, raw json.RawMessage) (string, error) {
			if s.products == nil {
				return "", errors.New("delete_product not configured")
			}
			var args struct {
				ProductID string `json:"product_id"`
			}
			if err := json.Unmarshal(raw, &args); err != nil {
				return "", err
			}
			if err := s.products.DeleteForAgent(ctx, agent.SiteID, args.ProductID); err != nil {
				return "", err
			}
			return mustJSON(map[string]any{"status": "deleted", "product_id": args.ProductID}), nil
		},
	})

	register(Tool{
		Name:        "create_variant",
		Description: "Adds a variant under an existing product. Required: product_id. Optional: sku, name (e.g. 'Small / Black'), price_cents (>=0; overrides product.base_price_cents), compare_at_price_cents (for showing a strike-through), inventory_count, allow_backorder (boolean), sort_order, attributes (object map e.g. {size:'S', color:'black'}).",
		InputSchema: schema(`{
			"type":"object",
			"properties":{
				"product_id":{"type":"string"},
				"sku":{"type":"string"},
				"name":{"type":"string"},
				"price_cents":{"type":"integer","minimum":0},
				"compare_at_price_cents":{"type":"integer"},
				"inventory_count":{"type":"integer"},
				"allow_backorder":{"type":"boolean"},
				"sort_order":{"type":"integer"},
				"attributes":{"type":"object"}
			},
			"required":["product_id"]
		}`),
		RequiresWrite: true,
		Handler: func(ctx context.Context, agent *authmw.AgentIdentity, raw json.RawMessage) (string, error) {
			if s.products == nil {
				return "", errors.New("create_variant not configured")
			}
			var args struct {
				ProductID string `json:"product_id"`
			}
			if err := json.Unmarshal(raw, &args); err != nil {
				return "", err
			}
			created, err := s.products.CreateVariantForAgent(ctx, agent.SiteID, args.ProductID, raw)
			if err != nil {
				return "", err
			}
			return mustJSON(map[string]any{"variant": created}), nil
		},
	})

	register(Tool{
		Name:        "update_variant",
		Description: "Updates a variant. Required: variant_id. Any of sku/name/price_cents/compare_at_price_cents/inventory_count/allow_backorder/sort_order/attributes can be omitted to leave unchanged. Prefer set_inventory for inventory changes so the audit trail captures the reason.",
		InputSchema: schema(`{
			"type":"object",
			"properties":{
				"variant_id":{"type":"string"},
				"sku":{"type":"string"},
				"name":{"type":"string"},
				"price_cents":{"type":"integer","minimum":0},
				"compare_at_price_cents":{"type":"integer"},
				"inventory_count":{"type":"integer"},
				"allow_backorder":{"type":"boolean"},
				"sort_order":{"type":"integer"},
				"attributes":{"type":"object"}
			},
			"required":["variant_id"]
		}`),
		RequiresWrite: true,
		Handler: func(ctx context.Context, agent *authmw.AgentIdentity, raw json.RawMessage) (string, error) {
			if s.products == nil {
				return "", errors.New("update_variant not configured")
			}
			var args struct {
				VariantID string `json:"variant_id"`
			}
			if err := json.Unmarshal(raw, &args); err != nil {
				return "", err
			}
			updated, err := s.products.UpdateVariantForAgent(ctx, agent.SiteID, args.VariantID, raw)
			if err != nil {
				return "", err
			}
			return mustJSON(map[string]any{"variant": updated}), nil
		},
	})

	register(Tool{
		Name:        "delete_variant",
		Description: "Deletes a variant. Required: variant_id.",
		InputSchema: schema(`{
			"type":"object",
			"properties":{"variant_id":{"type":"string"}},
			"required":["variant_id"]
		}`),
		RequiresWrite: true,
		Handler: func(ctx context.Context, agent *authmw.AgentIdentity, raw json.RawMessage) (string, error) {
			if s.products == nil {
				return "", errors.New("delete_variant not configured")
			}
			var args struct {
				VariantID string `json:"variant_id"`
			}
			if err := json.Unmarshal(raw, &args); err != nil {
				return "", err
			}
			if err := s.products.DeleteVariantForAgent(ctx, agent.SiteID, args.VariantID); err != nil {
				return "", err
			}
			return mustJSON(map[string]any{"status": "deleted", "variant_id": args.VariantID}), nil
		},
	})

	register(Tool{
		Name:        "set_inventory",
		Description: "Adjusts a variant's inventory_count and writes an entry to the inventory audit log. Required: variant_id, plus EITHER new_count (absolute) OR delta (signed change). Optional: reason (restock|sale|manual|damage|return|correction, default 'manual'), note. Use 'restock' after a delivery, 'manual' for ad-hoc corrections, 'damage' for write-offs, 'correction' for cycle counts.",
		InputSchema: schema(`{
			"type":"object",
			"properties":{
				"variant_id":{"type":"string"},
				"new_count":{"type":"integer"},
				"delta":{"type":"integer"},
				"reason":{"type":"string","enum":["restock","sale","manual","damage","return","correction"]},
				"note":{"type":"string"}
			},
			"required":["variant_id"]
		}`),
		RequiresWrite: true,
		Handler: func(ctx context.Context, agent *authmw.AgentIdentity, raw json.RawMessage) (string, error) {
			if s.products == nil {
				return "", errors.New("set_inventory not configured")
			}
			var args struct {
				VariantID string `json:"variant_id"`
			}
			if err := json.Unmarshal(raw, &args); err != nil {
				return "", err
			}
			result, err := s.products.SetInventoryForAgent(ctx, agent.SiteID, args.VariantID, "agent:"+agent.KeyID, raw)
			if err != nil {
				return "", err
			}
			return mustJSON(result), nil
		},
	})
}

// registerDiscountCodeTools wires the discount-code MCP surface. The
// tenant uses these to spin up promo codes from the agent rather than
// the dashboard. Slice B's checkout flow validates + redeems codes.
func (s *Server) registerDiscountCodeTools() {
	register := func(t Tool) { s.tools[t.Name] = t }

	register(Tool{
		Name:        "list_discount_codes",
		Description: "Lists the site's discount codes (Sprint 2 slice A). Each row: id, code, kind (percent|fixed), value (basis points for percent, cents for fixed), min_subtotal_cents, max_uses, used_count, starts_at, ends_at, applies_to (all|categories|products), is_active.",
		InputSchema: schema(`{"type":"object","properties":{}}`),
		Handler: func(ctx context.Context, agent *authmw.AgentIdentity, raw json.RawMessage) (string, error) {
			if s.discounts == nil {
				return "", errors.New("list_discount_codes not configured")
			}
			rows, err := s.queries.ListDiscountCodesBySite(ctx, store.ListDiscountCodesBySiteParams{
				SiteID: agent.SiteID, Limit: 500,
			})
			if err != nil {
				return "", err
			}
			return mustJSON(map[string]any{"discount_codes": rows, "count": len(rows)}), nil
		},
	})

	register(Tool{
		Name:        "create_discount_code",
		Description: "Creates a discount code. Required: code (uppercase alphanumeric / dash / underscore, 2-50 chars), kind ('percent' or 'fixed'), value (for percent: basis points 1-10000 = 0.01%-100%; for fixed: cents). Optional: min_subtotal_cents, max_uses (0 = unlimited), starts_at + ends_at (ISO timestamps; empty = always), applies_to (all|categories|products, default 'all'), applies_to_ids (array of category names or product IDs), is_active (default false; flip true when ready to redeem).",
		InputSchema: schema(`{
			"type":"object",
			"properties":{
				"code":{"type":"string"},
				"kind":{"type":"string","enum":["percent","fixed"]},
				"value":{"type":"integer","minimum":1},
				"min_subtotal_cents":{"type":"integer","minimum":0},
				"max_uses":{"type":"integer","minimum":0},
				"starts_at":{"type":"string"},
				"ends_at":{"type":"string"},
				"applies_to":{"type":"string","enum":["all","categories","products"]},
				"applies_to_ids":{"type":"array","items":{"type":"string"}},
				"is_active":{"type":"boolean"}
			},
			"required":["code","kind","value"]
		}`),
		RequiresWrite: true,
		Handler: func(ctx context.Context, agent *authmw.AgentIdentity, raw json.RawMessage) (string, error) {
			if s.discounts == nil {
				return "", errors.New("create_discount_code not configured")
			}
			created, err := s.discounts.CreateForAgent(ctx, agent.SiteID, raw)
			if err != nil {
				return "", err
			}
			return mustJSON(map[string]any{"discount_code": created}), nil
		},
	})

	register(Tool{
		Name:        "update_discount_code",
		Description: "Updates a discount code. Required: code_id. Optional: any of code/kind/value/min_subtotal_cents/max_uses/starts_at/ends_at/applies_to/applies_to_ids/is_active.",
		InputSchema: schema(`{
			"type":"object",
			"properties":{
				"code_id":{"type":"string"},
				"code":{"type":"string"},
				"kind":{"type":"string","enum":["percent","fixed"]},
				"value":{"type":"integer"},
				"min_subtotal_cents":{"type":"integer","minimum":0},
				"max_uses":{"type":"integer","minimum":0},
				"starts_at":{"type":"string"},
				"ends_at":{"type":"string"},
				"applies_to":{"type":"string","enum":["all","categories","products"]},
				"applies_to_ids":{"type":"array","items":{"type":"string"}},
				"is_active":{"type":"boolean"}
			},
			"required":["code_id"]
		}`),
		RequiresWrite: true,
		Handler: func(ctx context.Context, agent *authmw.AgentIdentity, raw json.RawMessage) (string, error) {
			if s.discounts == nil {
				return "", errors.New("update_discount_code not configured")
			}
			var args struct {
				CodeID string `json:"code_id"`
			}
			if err := json.Unmarshal(raw, &args); err != nil {
				return "", err
			}
			args.CodeID = strings.TrimSpace(args.CodeID)
			updated, err := s.discounts.UpdateForAgent(ctx, agent.SiteID, args.CodeID, raw)
			if err != nil {
				return "", err
			}
			return mustJSON(map[string]any{"discount_code": updated}), nil
		},
	})

	register(Tool{
		Name:        "delete_discount_code",
		Description: "Deletes a discount code. Required: code_id.",
		InputSchema: schema(`{
			"type":"object",
			"properties":{"code_id":{"type":"string"}},
			"required":["code_id"]
		}`),
		RequiresWrite: true,
		Handler: func(ctx context.Context, agent *authmw.AgentIdentity, raw json.RawMessage) (string, error) {
			if s.discounts == nil {
				return "", errors.New("delete_discount_code not configured")
			}
			var args struct {
				CodeID string `json:"code_id"`
			}
			if err := json.Unmarshal(raw, &args); err != nil {
				return "", err
			}
			if err := s.discounts.DeleteForAgent(ctx, agent.SiteID, args.CodeID); err != nil {
				return "", err
			}
			return mustJSON(map[string]any{"status": "deleted", "code_id": args.CodeID}), nil
		},
	})
}
