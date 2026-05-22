package builder

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/bright-interaction/slab/internal/store"
)

// Sprint 2 slice C (2026-05-22): renderer + resolver tests for the
// four public storefront blocks. The pure render tests use
// hand-crafted _resolved_* data so they are DB-free. The resolver test
// uses the in-memory SQLite helper from collections_test.go.

func TestRenderProductGrid_RendersResolvedProducts(t *testing.T) {
	data := map[string]any{
		"heading": "All products",
		"columns": "3",
		"_resolved_products": []any{
			map[string]any{
				"id":             "p1",
				"name":           "Blue tee",
				"slug":           "blue-tee",
				"price_cents":    int64(2500),
				"currency":       "EUR",
				"first_image_id": "",
				"first_variant": map[string]any{
					"id":          "v1",
					"product_id":  "p1",
					"price_cents": int64(2500),
					"currency":    "EUR",
				},
			},
			map[string]any{
				"id":             "p2",
				"name":           "Red cap",
				"slug":           "red-cap",
				"price_cents":    int64(1500),
				"currency":       "EUR",
				"first_image_id": "",
				"first_variant": map[string]any{
					"id":          "v2",
					"product_id":  "p2",
					"price_cents": int64(1500),
					"currency":    "EUR",
				},
			},
		},
	}
	out := renderProductGridBlock(data, nil)
	if !strings.Contains(out, "Blue tee") || !strings.Contains(out, "Red cap") {
		t.Errorf("missing product names in output: %s", out)
	}
	if !strings.Contains(out, `25.00 EUR`) || !strings.Contains(out, `15.00 EUR`) {
		t.Errorf("missing formatted prices: %s", out)
	}
	if !strings.Contains(out, "data-cart-add") || !strings.Contains(out, "data-cart-item=") {
		t.Errorf("missing add-to-cart wiring: %s", out)
	}
	if !strings.Contains(out, "block--product_grid--cols-3") {
		t.Errorf("missing column class: %s", out)
	}
}

func TestRenderProductGrid_EmptyState(t *testing.T) {
	out := renderProductGridBlock(map[string]any{
		"heading":            "Catalog",
		"_resolved_products": []any{},
	}, nil)
	if !strings.Contains(out, "No products yet") {
		t.Errorf("expected empty state copy: %s", out)
	}
}

func TestRenderProductDetail_RendersResolvedProduct(t *testing.T) {
	data := map[string]any{
		"product_slug":  "blue-tee",
		"cta_label":     "Add to bag",
		"show_variants": true,
		"_resolved_product": map[string]any{
			"id":          "p1",
			"name":        "Blue tee",
			"slug":        "blue-tee",
			"description": "Soft cotton tee.",
			"currency":    "EUR",
			"base_price":  int64(2500),
			"variants": []any{
				map[string]any{
					"id":          "v_s",
					"product_id":  "p1",
					"name":        "Small",
					"price_cents": int64(2500),
					"currency":    "EUR",
				},
				map[string]any{
					"id":          "v_m",
					"product_id":  "p1",
					"name":        "Medium",
					"price_cents": int64(2700),
					"currency":    "EUR",
				},
			},
		},
	}
	out := renderProductDetailBlock(data, nil)
	if !strings.Contains(out, "Blue tee") {
		t.Errorf("missing product name: %s", out)
	}
	if !strings.Contains(out, "data-product-detail") {
		t.Errorf("missing data-product-detail anchor: %s", out)
	}
	if !strings.Contains(out, "data-variant-pick") {
		t.Errorf("missing variant picker: %s", out)
	}
	if !strings.Contains(out, "Add to bag") {
		t.Errorf("missing custom CTA label: %s", out)
	}
	if strings.Count(out, `name="variant"`) != 2 {
		t.Errorf("expected 2 variant radio inputs, output: %s", out)
	}
}

func TestRenderProductDetail_MissingProductEmitsNotice(t *testing.T) {
	out := renderProductDetailBlock(map[string]any{
		"product_slug":      "ghost-product",
		"_resolved_missing": true,
	}, nil)
	if !strings.Contains(out, "Product not found") {
		t.Errorf("expected missing-product notice, got: %s", out)
	}
	if !strings.Contains(out, "ghost-product") {
		t.Errorf("notice should echo the slug, got: %s", out)
	}
}

func TestRenderProductDetail_NoVariantsHidesPicker(t *testing.T) {
	data := map[string]any{
		"product_slug":  "single-sku",
		"show_variants": true,
		"_resolved_product": map[string]any{
			"id":         "p1",
			"name":       "Solo product",
			"slug":       "single-sku",
			"currency":   "EUR",
			"base_price": int64(999),
			"variants": []any{
				map[string]any{
					"id":          "v_only",
					"product_id":  "p1",
					"name":        "Default",
					"price_cents": int64(999),
					"currency":    "EUR",
				},
			},
		},
	}
	out := renderProductDetailBlock(data, nil)
	if strings.Contains(out, "data-variant-pick") {
		t.Errorf("variant picker should not render with a single variant: %s", out)
	}
	if !strings.Contains(out, "data-cart-add") {
		t.Errorf("Add-to-cart button must still render: %s", out)
	}
}

func TestRenderCartDrawer_EmitsShellAndCheckoutLink(t *testing.T) {
	out := renderCartDrawerBlock(map[string]any{
		"trigger_label":   "Bag",
		"checkout_url":    "/pay",
		"empty_message":   "Bag is empty.",
		"currency_locale": "sv-SE",
	})
	for _, want := range []string{
		"data-cart-open",
		"data-cart-drawer",
		"data-cart-empty",
		"data-cart-items",
		"data-cart-total",
		"data-cart-checkout",
		`href="/pay"`,
		`data-currency-locale="sv-SE"`,
		"Bag is empty.",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("cart drawer missing %q in output:\n%s", want, out)
		}
	}
}

func TestRenderCheckoutForm_EmbedsSiteIDAndReturnURL(t *testing.T) {
	out := renderCheckoutFormBlock(map[string]any{
		"heading":                  "Pay",
		"return_url":               "/thank-you",
		"submit_label":             "Place order",
		"require_shipping_address": true,
		"require_phone":            true,
	}, "site_abc")
	for _, want := range []string{
		`data-checkout-form`,
		`data-site-id="site_abc"`,
		`data-return-url="/thank-you"`,
		`data-require-shipping="1"`,
		`name="ship_country"`,
		`name="email"`,
		`name="phone"`,
		"Place order",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("checkout_form missing %q in output:\n%s", want, out)
		}
	}
}

func TestRenderCheckoutForm_OptionalPhoneAndNoShipping(t *testing.T) {
	out := renderCheckoutFormBlock(map[string]any{
		"return_url":               "/done",
		"require_phone":            false,
		"require_shipping_address": false,
	}, "site_abc")
	if !strings.Contains(out, `data-require-shipping="0"`) {
		t.Errorf("expected data-require-shipping=0 when shipping not required: %s", out)
	}
	if strings.Contains(out, `name="ship_street"`) {
		t.Errorf("shipping fields should be omitted when require_shipping_address=false: %s", out)
	}
	if !strings.Contains(out, "Phone (optional)") {
		t.Errorf("phone label should mark optional when require_phone=false: %s", out)
	}
}

// TestResolveProductGridBlock_PicksActiveProductsAndCaps drives the
// resolver against a real SQLite catalog. Seeds 3 products (2 active,
// 1 draft) plus variants and asserts the resolver: filters by
// status='active', looks up variants, picks the lowest variant price
// as display, and applies the requested limit.
func TestResolveProductGridBlock_PicksActiveProductsAndCaps(t *testing.T) {
	_, q, siteID, _ := newBuilderTestDB(t)
	ctx := context.Background()

	mustProduct(t, q, store.CreateProductParams{
		ID: "p_active1", SiteID: siteID, Name: "Active One", Slug: "active-one",
		Status: "active", Currency: "EUR", BasePriceCents: 1000, SortOrder: 1,
	})
	mustProduct(t, q, store.CreateProductParams{
		ID: "p_active2", SiteID: siteID, Name: "Active Two", Slug: "active-two",
		Status: "active", Currency: "EUR", BasePriceCents: 2000, SortOrder: 2,
	})
	mustProduct(t, q, store.CreateProductParams{
		ID: "p_draft", SiteID: siteID, Name: "Draft", Slug: "draft",
		Status: "draft", Currency: "EUR", BasePriceCents: 9999, SortOrder: 3,
	})
	mustVariant(t, q, store.CreateProductVariantParams{
		ID: "v_a1_lo", ProductID: "p_active1", Name: "Small", PriceCents: 1200, InventoryCount: 5,
	})
	mustVariant(t, q, store.CreateProductVariantParams{
		ID: "v_a1_hi", ProductID: "p_active1", Name: "Large", PriceCents: 1500, InventoryCount: 5, SortOrder: 1,
	})
	mustVariant(t, q, store.CreateProductVariantParams{
		ID: "v_a2", ProductID: "p_active2", Name: "Only", PriceCents: 2200, InventoryCount: 5,
	})

	site, err := q.GetSiteByID(ctx, siteID)
	if err != nil {
		t.Fatal(err)
	}
	bl := store.Block{
		ID: "bl_grid", BlockType: "product_grid",
		DataJson: `{"limit":1}`,
	}
	blocks := []store.Block{bl}
	resolveStorefrontBlocks(ctx, q, site, blocks)

	var data map[string]any
	if err := json.Unmarshal([]byte(blocks[0].DataJson), &data); err != nil {
		t.Fatalf("post-resolve json: %v", err)
	}
	resolved, ok := data["_resolved_products"].([]any)
	if !ok {
		t.Fatalf("missing _resolved_products: %v", data)
	}
	if len(resolved) != 1 {
		t.Fatalf("expected 1 product after limit=1, got %d", len(resolved))
	}
	first, _ := resolved[0].(map[string]any)
	if name, _ := first["name"].(string); name != "Active One" {
		t.Errorf("expected Active One (sort_order=1), got %q", name)
	}
	// Display price should be the lower variant (1200), not the base
	// price (1000) or the higher variant (1500).
	if price := int64OrZero(first["price_cents"]); price != 1200 {
		t.Errorf("expected display price 1200, got %d", price)
	}
	// Draft must not surface.
	for _, r := range resolved {
		if m, ok := r.(map[string]any); ok {
			if name, _ := m["name"].(string); name == "Draft" {
				t.Errorf("draft product leaked into product_grid")
			}
		}
	}
}

func TestResolveProductDetailBlock_LoadsVariants(t *testing.T) {
	_, q, siteID, _ := newBuilderTestDB(t)
	ctx := context.Background()

	mustProduct(t, q, store.CreateProductParams{
		ID: "p_detail", SiteID: siteID, Name: "Hat", Slug: "hat",
		Description: "A nice hat.", Status: "active", Currency: "EUR",
		BasePriceCents: 3000, SortOrder: 1,
	})
	mustVariant(t, q, store.CreateProductVariantParams{
		ID: "v_hat_s", ProductID: "p_detail", Name: "Small", PriceCents: 3000, InventoryCount: 2,
	})
	mustVariant(t, q, store.CreateProductVariantParams{
		ID: "v_hat_l", ProductID: "p_detail", Name: "Large", PriceCents: 3200, InventoryCount: 4, SortOrder: 1,
	})

	site, err := q.GetSiteByID(ctx, siteID)
	if err != nil {
		t.Fatal(err)
	}
	bl := store.Block{
		ID: "bl_detail", BlockType: "product_detail",
		DataJson: `{"product_slug":"hat"}`,
	}
	blocks := []store.Block{bl}
	resolveStorefrontBlocks(ctx, q, site, blocks)

	var data map[string]any
	if err := json.Unmarshal([]byte(blocks[0].DataJson), &data); err != nil {
		t.Fatalf("post-resolve json: %v", err)
	}
	prod, ok := data["_resolved_product"].(map[string]any)
	if !ok {
		t.Fatalf("missing _resolved_product: %v", data)
	}
	variants, _ := prod["variants"].([]any)
	if len(variants) != 2 {
		t.Fatalf("expected 2 variants, got %d", len(variants))
	}
	if name, _ := prod["name"].(string); name != "Hat" {
		t.Errorf("expected name=Hat, got %q", name)
	}
}

func TestResolveProductDetailBlock_MissingMarksFlag(t *testing.T) {
	_, q, siteID, _ := newBuilderTestDB(t)
	ctx := context.Background()
	site, err := q.GetSiteByID(ctx, siteID)
	if err != nil {
		t.Fatal(err)
	}
	bl := store.Block{
		ID: "bl_missing", BlockType: "product_detail",
		DataJson: `{"product_slug":"does-not-exist"}`,
	}
	blocks := []store.Block{bl}
	resolveStorefrontBlocks(ctx, q, site, blocks)

	var data map[string]any
	if err := json.Unmarshal([]byte(blocks[0].DataJson), &data); err != nil {
		t.Fatalf("post-resolve json: %v", err)
	}
	if missing, _ := data["_resolved_missing"].(bool); !missing {
		t.Errorf("expected _resolved_missing=true for unknown slug, got: %v", data)
	}
}

// TestStorefrontIslandScript_ContainsContract locks the data-attribute
// contract that the renderer and the cart island agree on. If the
// renderer changes a selector and the island isn't updated (or vice
// versa) this test catches the drift before runtime.
func TestStorefrontIslandScript_ContainsContract(t *testing.T) {
	js := StorefrontIslandScript("site_abc")
	if !strings.Contains(js, `"site_abc"`) {
		t.Errorf("site id not baked into island: %s", js[:200])
	}
	for _, sel := range []string{
		"data-cart-add",
		"data-cart-open",
		"data-cart-close",
		"data-cart-qty-inc",
		"data-cart-qty-dec",
		"data-cart-remove",
		"data-cart-drawer",
		"data-cart-items",
		"data-cart-total",
		"data-cart-empty",
		"data-cart-count",
		"data-cart-checkout",
		"data-checkout-form",
		"data-variant-pick",
		"data-product-price",
		"data-checkout-error",
		"/api/sites/",
		"/checkout",
		"atomic-cart-",
	} {
		if !strings.Contains(js, sel) {
			t.Errorf("storefront island missing contract token %q", sel)
		}
	}
}

func TestHasStorefrontBlock(t *testing.T) {
	if hasStorefrontBlock([]store.Block{{BlockType: "hero"}, {BlockType: "cta"}}) {
		t.Errorf("hasStorefrontBlock returned true for non-storefront page")
	}
	if !hasStorefrontBlock([]store.Block{{BlockType: "hero"}, {BlockType: "product_grid"}}) {
		t.Errorf("hasStorefrontBlock should match product_grid")
	}
	if !hasStorefrontBlock([]store.Block{{BlockType: "checkout_form"}}) {
		t.Errorf("hasStorefrontBlock should match checkout_form")
	}
}

func TestFirstImageIDFrom(t *testing.T) {
	if got := firstImageIDFrom(`["abc","def"]`); got != "abc" {
		t.Errorf("expected abc, got %q", got)
	}
	if got := firstImageIDFrom(`"single"`); got != "single" {
		t.Errorf("legacy single-string form should be supported, got %q", got)
	}
	if got := firstImageIDFrom(""); got != "" {
		t.Errorf("expected empty for empty input, got %q", got)
	}
	if got := firstImageIDFrom(`[]`); got != "" {
		t.Errorf("expected empty for empty array, got %q", got)
	}
}

// mustProduct / mustVariant are tiny test helpers that fatal on insert
// failure. The defaults shape an active product that the resolver +
// rendering paths exercise; tests override the fields they care about.
func mustProduct(t *testing.T, q *store.Queries, p store.CreateProductParams) {
	t.Helper()
	if p.Currency == "" {
		p.Currency = "EUR"
	}
	if p.Status == "" {
		p.Status = "active"
	}
	if p.ImagesJson == "" {
		p.ImagesJson = "[]"
	}
	if p.TaxClass == "" {
		p.TaxClass = "standard"
	}
	if err := q.CreateProduct(context.Background(), p); err != nil {
		t.Fatalf("create product %s: %v", p.ID, err)
	}
}

func mustVariant(t *testing.T, q *store.Queries, v store.CreateProductVariantParams) {
	t.Helper()
	if v.Name == "" {
		v.Name = "Default"
	}
	if v.AttributesJson == "" {
		v.AttributesJson = "{}"
	}
	if err := q.CreateProductVariant(context.Background(), v); err != nil {
		t.Fatalf("create variant %s: %v", v.ID, err)
	}
}
