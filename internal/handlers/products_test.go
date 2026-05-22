package handlers

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/brightinteraction/atomicsite/internal/store"
)

// Sprint 2 slice A (WP/Webflow roadmap): catalog tests.

func TestProducts_CreateValidatesSlugAndStatus(t *testing.T) {
	_, q := setupDeployTestDB(t)
	siteID := "psite0000000000aaaa0001"
	seedSite(t, q, siteID)
	h := NewProductHandler(nil, q)
	ctx := context.Background()

	// Bad slug
	if _, err := h.CreateForAgent(ctx, siteID, ProductInput{
		Name: "p", Slug: "Has Space!", Status: "active",
	}); err == nil {
		t.Errorf("expected slug error")
	}
	// Bad status
	if _, err := h.CreateForAgent(ctx, siteID, ProductInput{
		Name: "p", Slug: "ok-slug", Status: "publish",
	}); err == nil {
		t.Errorf("expected status error")
	}
	// Happy path
	p, err := h.CreateForAgent(ctx, siteID, ProductInput{
		Name: "Hat", Slug: "hat", Status: "active",
		BasePriceCents: 2999, Currency: "eur",
	})
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if p.Status != "active" || p.Currency != "EUR" || p.BasePriceCents != 2999 {
		t.Errorf("got %#v", p)
	}
}

func TestProducts_UniqueSlugPerSite(t *testing.T) {
	_, q := setupDeployTestDB(t)
	siteID := "psite0000000000aaaa0002"
	seedSite(t, q, siteID)
	h := NewProductHandler(nil, q)
	ctx := context.Background()
	if _, err := h.CreateForAgent(ctx, siteID, ProductInput{Name: "a", Slug: "dup", Status: "draft"}); err != nil {
		t.Fatalf("first create: %v", err)
	}
	if _, err := h.CreateForAgent(ctx, siteID, ProductInput{Name: "b", Slug: "dup", Status: "draft"}); err == nil {
		t.Errorf("expected unique constraint error on second create with same slug")
	}
}

func TestProducts_VariantInventoryAuditLog(t *testing.T) {
	_, q := setupDeployTestDB(t)
	siteID := "psite0000000000aaaa0003"
	seedSite(t, q, siteID)
	h := NewProductHandler(nil, q)
	ctx := context.Background()

	p, err := h.CreateForAgent(ctx, siteID, ProductInput{Name: "Shirt", Slug: "shirt", Status: "active"})
	if err != nil {
		t.Fatalf("create product: %v", err)
	}
	raw, _ := json.Marshal(VariantInput{
		Sku: "SHIRT-S", Name: "Small", PriceCents: 1500, InventoryCount: 10,
	})
	v, err := h.CreateVariantForAgent(ctx, siteID, p.ID, raw)
	if err != nil {
		t.Fatalf("create variant: %v", err)
	}

	// Apply delta -3 (sale)
	rawAdj, _ := json.Marshal(map[string]any{"delta": -3, "reason": "sale", "note": "test sale"})
	result, err := h.SetInventoryForAgent(ctx, siteID, v.ID, "user:tester", rawAdj)
	if err != nil {
		t.Fatalf("set inventory: %v", err)
	}
	if result["delta"].(int64) != -3 {
		t.Errorf("delta = %v; want -3", result["delta"])
	}
	v2, _ := q.GetProductVariantByID(ctx, v.ID)
	if v2.InventoryCount != 7 {
		t.Errorf("inventory_count after sale delta = %d; want 7", v2.InventoryCount)
	}

	// Audit log row exists.
	adjs, err := q.ListInventoryAdjustmentsByVariant(ctx, store.ListInventoryAdjustmentsByVariantParams{
		VariantID: v.ID, Limit: 10,
	})
	if err != nil {
		t.Fatalf("list adjustments: %v", err)
	}
	if len(adjs) != 1 || adjs[0].Reason != "sale" || adjs[0].Delta != -3 || adjs[0].NewCount != 7 {
		t.Errorf("audit log = %#v", adjs)
	}
}

func TestProducts_CrossTenantVariantBlocked(t *testing.T) {
	_, q := setupDeployTestDB(t)
	siteA := "psite0000000000aaaa0004"
	siteB := "psite0000000000aaaa0005"
	seedSite(t, q, siteA)
	seedSite(t, q, siteB)
	h := NewProductHandler(nil, q)
	ctx := context.Background()

	// Create product + variant in site B.
	pB, err := h.CreateForAgent(ctx, siteB, ProductInput{Name: "B Hat", Slug: "b-hat", Status: "active"})
	if err != nil {
		t.Fatalf("create in B: %v", err)
	}
	rawV, _ := json.Marshal(VariantInput{Name: "x"})
	v, err := h.CreateVariantForAgent(ctx, siteB, pB.ID, rawV)
	if err != nil {
		t.Fatalf("create variant in B: %v", err)
	}

	// Attempt to mutate variant from site A.
	if err := h.DeleteVariantForAgent(ctx, siteA, v.ID); err == nil {
		t.Errorf("expected cross-tenant block on delete variant")
	}
	rawAdj, _ := json.Marshal(map[string]any{"new_count": 999})
	if _, err := h.SetInventoryForAgent(ctx, siteA, v.ID, "agent:a", rawAdj); err == nil {
		t.Errorf("expected cross-tenant block on set inventory")
	}
}

func TestDiscountCodes_CreateValidates(t *testing.T) {
	_, q := setupDeployTestDB(t)
	siteID := "psite0000000000aaaa0006"
	seedSite(t, q, siteID)
	h := NewDiscountCodeHandler(nil, q)
	ctx := context.Background()

	// Bad code
	raw, _ := json.Marshal(map[string]any{"code": "lowercase!", "kind": "percent", "value": 1000})
	if _, err := h.CreateForAgent(ctx, siteID, raw); err == nil {
		t.Errorf("expected code validation error")
	}
	// Bad kind
	raw, _ = json.Marshal(map[string]any{"code": "OK10", "kind": "bogo", "value": 1000})
	if _, err := h.CreateForAgent(ctx, siteID, raw); err == nil {
		t.Errorf("expected kind validation error")
	}
	// Percent out of range
	raw, _ = json.Marshal(map[string]any{"code": "OK11", "kind": "percent", "value": 0})
	if _, err := h.CreateForAgent(ctx, siteID, raw); err == nil {
		t.Errorf("expected percent value error")
	}
	raw, _ = json.Marshal(map[string]any{"code": "OK12", "kind": "percent", "value": 20000})
	if _, err := h.CreateForAgent(ctx, siteID, raw); err == nil {
		t.Errorf("expected percent value cap error")
	}
	// Happy path
	raw, _ = json.Marshal(map[string]any{"code": "summer15", "kind": "percent", "value": 1500, "is_active": true})
	d, err := h.CreateForAgent(ctx, siteID, raw)
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if d.Code != "SUMMER15" || d.Kind != "percent" || d.Value != 1500 || d.IsActive != 1 {
		t.Errorf("got %#v", d)
	}
}

func TestDiscountCodes_UniqueCodePerSite(t *testing.T) {
	_, q := setupDeployTestDB(t)
	siteID := "psite0000000000aaaa0007"
	seedSite(t, q, siteID)
	h := NewDiscountCodeHandler(nil, q)
	ctx := context.Background()
	raw, _ := json.Marshal(map[string]any{"code": "DUP10", "kind": "percent", "value": 1000})
	if _, err := h.CreateForAgent(ctx, siteID, raw); err != nil {
		t.Fatalf("first create: %v", err)
	}
	if _, err := h.CreateForAgent(ctx, siteID, raw); err == nil {
		t.Errorf("expected duplicate code error")
	}
}
