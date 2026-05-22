package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"

	"github.com/go-chi/chi/v5"

	"github.com/bright-interaction/slab/internal/store"
)

// Sprint 2 slice B (WP/Webflow roadmap): order pipeline tests.

func seedActiveProductWithVariant(t *testing.T, q *store.Queries, siteID, slug string, inventoryCount int64) (store.Product, store.ProductVariant) {
	t.Helper()
	ctx := context.Background()
	ph := NewProductHandler(nil, q)
	p, err := ph.CreateForAgent(ctx, siteID, ProductInput{
		Name: slug, Slug: slug, Status: "active", BasePriceCents: 1000, Currency: "EUR",
	})
	if err != nil {
		t.Fatalf("seed product: %v", err)
	}
	rawV, _ := json.Marshal(VariantInput{
		Name: "default", PriceCents: 1000, InventoryCount: inventoryCount,
	})
	v, err := ph.CreateVariantForAgent(ctx, siteID, p.ID, rawV)
	if err != nil {
		t.Fatalf("seed variant: %v", err)
	}
	return p, v
}

func checkoutRouter(h *OrderHandler) chi.Router {
	r := chi.NewRouter()
	r.Post("/api/sites/{siteID}/checkout", h.Checkout)
	r.Post("/api/sites/{siteID}/orders/{orderID}/status", h.UpdateStatus)
	r.Get("/api/sites/{siteID}/orders", h.List)
	return r
}

func TestOrders_CheckoutHappyPathNoMollie(t *testing.T) {
	_, q := setupDeployTestDB(t)
	siteID := "ositea0000000000aaaa01"
	seedSite(t, q, siteID)
	_, v := seedActiveProductWithVariant(t, q, siteID, "hat", 10)
	h := NewOrderHandler(nil, q)
	r := checkoutRouter(h)

	code, body := postJSON(t, r, "/api/sites/"+siteID+"/checkout", map[string]any{
		"items":    []map[string]any{{"variant_id": v.ID, "quantity": 2}},
		"customer": map[string]any{"email": "buyer@example.com", "name": "Buyer"},
	})
	if code != http.StatusCreated {
		t.Fatalf("status = %d", code)
	}
	order := body["order"].(map[string]any)
	if order["status"] != "pending" {
		t.Errorf("status = %v; want pending", order["status"])
	}
	if order["total_cents"].(float64) != 2000 {
		t.Errorf("total = %v; want 2000", order["total_cents"])
	}
	if body["checkout_url"] != "" {
		t.Errorf("checkout_url should be empty (no Mollie configured); got %v", body["checkout_url"])
	}
}

func TestOrders_CheckoutRejectsInactiveProduct(t *testing.T) {
	_, q := setupDeployTestDB(t)
	siteID := "ositea0000000000aaaa02"
	seedSite(t, q, siteID)
	ph := NewProductHandler(nil, q)
	p, _ := ph.CreateForAgent(context.Background(), siteID, ProductInput{
		Name: "Draft", Slug: "draft", Status: "draft", BasePriceCents: 1000,
	})
	rawV, _ := json.Marshal(VariantInput{Name: "x", PriceCents: 1000, InventoryCount: 5})
	v, _ := ph.CreateVariantForAgent(context.Background(), siteID, p.ID, rawV)

	h := NewOrderHandler(nil, q)
	r := checkoutRouter(h)
	code, _ := postJSON(t, r, "/api/sites/"+siteID+"/checkout", map[string]any{
		"items":    []map[string]any{{"variant_id": v.ID, "quantity": 1}},
		"customer": map[string]any{"email": "x@example.com"},
	})
	if code != http.StatusBadRequest {
		t.Errorf("status = %d; want 400 (inactive product)", code)
	}
}

func TestOrders_CheckoutRejectsInsufficientInventory(t *testing.T) {
	_, q := setupDeployTestDB(t)
	siteID := "ositea0000000000aaaa03"
	seedSite(t, q, siteID)
	_, v := seedActiveProductWithVariant(t, q, siteID, "scarce", 2)
	h := NewOrderHandler(nil, q)
	r := checkoutRouter(h)
	code, _ := postJSON(t, r, "/api/sites/"+siteID+"/checkout", map[string]any{
		"items":    []map[string]any{{"variant_id": v.ID, "quantity": 5}},
		"customer": map[string]any{"email": "x@example.com"},
	})
	if code != http.StatusConflict {
		t.Errorf("status = %d; want 409 (insufficient inventory)", code)
	}
}

func TestOrders_CheckoutAppliesDiscountCode(t *testing.T) {
	_, q := setupDeployTestDB(t)
	siteID := "ositea0000000000aaaa04"
	seedSite(t, q, siteID)
	_, v := seedActiveProductWithVariant(t, q, siteID, "tshirt", 100)
	dh := NewDiscountCodeHandler(nil, q)
	raw, _ := json.Marshal(map[string]any{
		"code": "PCT10", "kind": "percent", "value": 1000, "is_active": true,
	})
	if _, err := dh.CreateForAgent(context.Background(), siteID, raw); err != nil {
		t.Fatalf("seed code: %v", err)
	}
	h := NewOrderHandler(nil, q)
	r := checkoutRouter(h)
	code, body := postJSON(t, r, "/api/sites/"+siteID+"/checkout", map[string]any{
		"items":         []map[string]any{{"variant_id": v.ID, "quantity": 3}},
		"customer":      map[string]any{"email": "x@example.com"},
		"discount_code": "pct10",
	})
	if code != http.StatusCreated {
		t.Fatalf("status = %d", code)
	}
	order := body["order"].(map[string]any)
	if order["subtotal_cents"].(float64) != 3000 {
		t.Errorf("subtotal = %v; want 3000", order["subtotal_cents"])
	}
	if order["discount_cents"].(float64) != 300 {
		t.Errorf("discount = %v; want 300 (10%% of 3000)", order["discount_cents"])
	}
	if order["total_cents"].(float64) != 2700 {
		t.Errorf("total = %v; want 2700", order["total_cents"])
	}
}

func TestOrders_StateMachineRejectsIllegalTransition(t *testing.T) {
	_, q := setupDeployTestDB(t)
	siteID := "ositea0000000000aaaa05"
	seedSite(t, q, siteID)
	_, v := seedActiveProductWithVariant(t, q, siteID, "shirt", 50)
	h := NewOrderHandler(nil, q)
	r := checkoutRouter(h)
	code, body := postJSON(t, r, "/api/sites/"+siteID+"/checkout", map[string]any{
		"items":    []map[string]any{{"variant_id": v.ID, "quantity": 1}},
		"customer": map[string]any{"email": "x@example.com"},
	})
	if code != http.StatusCreated {
		t.Fatalf("seed order: status=%d", code)
	}
	orderID := body["order"].(map[string]any)["id"].(string)

	// pending -> fulfilled is illegal (must go through paid first).
	code, _ = postJSON(t, r, "/api/sites/"+siteID+"/orders/"+orderID+"/status", map[string]any{
		"status": "fulfilled",
	})
	if code != http.StatusBadRequest {
		t.Errorf("pending->fulfilled status = %d; want 400", code)
	}
	// pending -> cancelled is legal.
	code, _ = postJSON(t, r, "/api/sites/"+siteID+"/orders/"+orderID+"/status", map[string]any{
		"status": "cancelled",
	})
	if code != http.StatusOK {
		t.Errorf("pending->cancelled status = %d; want 200", code)
	}
	// cancelled is terminal.
	code, _ = postJSON(t, r, "/api/sites/"+siteID+"/orders/"+orderID+"/status", map[string]any{
		"status": "fulfilled",
	})
	if code != http.StatusBadRequest {
		t.Errorf("cancelled->fulfilled status = %d; want 400", code)
	}
}

func TestOrders_PaidSideEffectsDecrementInventoryAndIncrementDiscount(t *testing.T) {
	_, q := setupDeployTestDB(t)
	siteID := "ositea0000000000aaaa06"
	seedSite(t, q, siteID)
	_, v := seedActiveProductWithVariant(t, q, siteID, "thing", 10)
	dh := NewDiscountCodeHandler(nil, q)
	raw, _ := json.Marshal(map[string]any{
		"code": "Z5", "kind": "fixed", "value": 100, "is_active": true,
	})
	dc, err := dh.CreateForAgent(context.Background(), siteID, raw)
	if err != nil {
		t.Fatalf("seed code: %v", err)
	}
	h := NewOrderHandler(nil, q)
	r := checkoutRouter(h)
	code, body := postJSON(t, r, "/api/sites/"+siteID+"/checkout", map[string]any{
		"items":         []map[string]any{{"variant_id": v.ID, "quantity": 3}},
		"customer":      map[string]any{"email": "x@example.com"},
		"discount_code": "z5",
	})
	if code != http.StatusCreated {
		t.Fatalf("status=%d", code)
	}
	// Apply side-effects (the webhook handler does this on 'paid').
	orderID := body["order"].(map[string]any)["id"].(string)
	order, _ := q.GetOrderByID(context.Background(), store.GetOrderByIDParams{ID: orderID, SiteID: siteID})
	h.applyPaidSideEffects(context.Background(), order)

	// Inventory decremented.
	v2, _ := q.GetProductVariantByID(context.Background(), v.ID)
	if v2.InventoryCount != 7 {
		t.Errorf("inventory after paid = %d; want 7", v2.InventoryCount)
	}
	adjs, _ := q.ListInventoryAdjustmentsByVariant(context.Background(), store.ListInventoryAdjustmentsByVariantParams{
		VariantID: v.ID, Limit: 5,
	})
	if len(adjs) != 1 || adjs[0].Reason != "sale" || adjs[0].Delta != -3 {
		t.Errorf("expected one sale adjustment delta=-3; got %#v", adjs)
	}
	dc2, _ := q.GetDiscountCodeByID(context.Background(), store.GetDiscountCodeByIDParams{ID: dc.ID, SiteID: siteID})
	if dc2.UsedCount != 1 {
		t.Errorf("discount used_count = %d; want 1", dc2.UsedCount)
	}
}

func TestOrders_PaymentEventIdempotencyByUniqueIndex(t *testing.T) {
	_, q := setupDeployTestDB(t)
	siteID := "ositea0000000000aaaa07"
	seedSite(t, q, siteID)
	ctx := context.Background()

	// Two inserts with the same (provider, payment_id, event_type) should
	// result in exactly one row (INSERT OR IGNORE on the unique index).
	for i := 0; i < 2; i++ {
		_ = q.CreatePaymentEvent(ctx, store.CreatePaymentEventParams{
			ID: newID(), SiteID: siteID, OrderID: "dummy-order",
			Provider: "mollie", PaymentID: "tr_test_abc", EventType: "paid",
			RawJson: "{}",
		})
	}
	row, err := q.GetPaymentEventByLookup(ctx, store.GetPaymentEventByLookupParams{
		Provider: "mollie", PaymentID: "tr_test_abc", EventType: "paid",
	})
	if err != nil {
		t.Fatalf("get event: %v", err)
	}
	if row.Processed != 0 {
		t.Errorf("processed = %d; want 0 before marker", row.Processed)
	}
	_ = q.MarkPaymentEventProcessed(ctx, store.MarkPaymentEventProcessedParams{
		Provider: "mollie", PaymentID: "tr_test_abc", EventType: "paid",
	})
	row, _ = q.GetPaymentEventByLookup(ctx, store.GetPaymentEventByLookupParams{
		Provider: "mollie", PaymentID: "tr_test_abc", EventType: "paid",
	})
	if row.Processed != 1 {
		t.Errorf("processed = %d; want 1", row.Processed)
	}
}
