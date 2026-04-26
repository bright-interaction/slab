package starterkits

import (
	"context"
	"fmt"

	"github.com/bright-interaction/slab/internal/store"
)

func init() {
	Default.Register(&ecommerceKit{})
}

// ecommerceKit ships the patterns that make Astro a competitive e-commerce
// foundation: SSR product pages with Product JSON-LD, lazy-loaded Svelte
// island for the cart drawer, Stripe Checkout (redirect-based, no client
// cart drama), and OG product images per SKU.
//
// We deliberately stick with Astro instead of forking to Qwik. Reasoning:
// (a) atomicsite's entire build pipeline is Astro-based (one build target,
// one eval engine, one set of guardrails), (b) Astro 5 server islands cover
// the resumability win for catalog / filter use cases, (c) Stripe Checkout
// (the recommended pattern here) is a redirect — the in-page cart is light
// enough that hydration cost is not the bottleneck, the network is.
type ecommerceKit struct{}

func (k *ecommerceKit) ID() string   { return "ecommerce" }
func (k *ecommerceKit) Name() string { return "E-commerce" }
func (k *ecommerceKit) Description() string {
	return "Astro storefront with Product / Offer schema, Svelte cart island, and Stripe Checkout redirect. Skips the SPA cart pitfalls — fast pages, real SEO, payments handled by Stripe."
}
func (k *ecommerceKit) TargetSiteTypes() []string { return []string{"ecommerce"} }

func (k *ecommerceKit) Apply(ctx context.Context, q *store.Queries, siteID string) error {
	if err := k.applyComponents(ctx, q, siteID); err != nil {
		return fmt.Errorf("ecommerce components: %w", err)
	}
	if err := k.applyCSS(ctx, q, siteID); err != nil {
		return fmt.Errorf("ecommerce css: %w", err)
	}
	if err := k.applyKnowledgebase(ctx, q, siteID); err != nil {
		return fmt.Errorf("ecommerce kb: %w", err)
	}
	if err := k.applyGuardrails(ctx, q, siteID); err != nil {
		return fmt.Errorf("ecommerce guardrails: %w", err)
	}
	if err := k.applyPages(ctx, q, siteID); err != nil {
		return fmt.Errorf("ecommerce pages: %w", err)
	}
	return nil
}

func (k *ecommerceKit) applyComponents(ctx context.Context, q *store.Queries, siteID string) error {
	defs := []componentDef{
		{
			Name:     "product-card",
			Category: "product",
			Template: `---
const { sku, title, price, currency = "EUR", image, alt, href } = Astro.props;
---
<article class="product-card" itemscope itemtype="https://schema.org/Product">
  <a href={href} class="product-link">
    <img src={image} alt={alt} width="600" height="600" loading="lazy" itemprop="image" />
    <h3 itemprop="name">{title}</h3>
    <p class="price" itemprop="offers" itemscope itemtype="https://schema.org/Offer">
      <meta itemprop="priceCurrency" content={currency} />
      <span itemprop="price" content={price}>{price}</span> {currency}
      <link itemprop="availability" href="https://schema.org/InStock" />
    </p>
  </a>
  <button type="button" class="btn btn-primary add-to-cart" data-sku={sku}>Add to cart</button>
</article>`,
			Props:      `{"type":"object","required":["sku","title","price","image","alt","href"],"properties":{"sku":{"type":"string"},"title":{"type":"string"},"price":{"type":"number"},"currency":{"type":"string"},"image":{"type":"string"},"alt":{"type":"string"},"href":{"type":"string"}}}`,
			CSSClasses: `["product-card","product-link","price","add-to-cart"]`,
			Usage:      "Product card with embedded Product / Offer microdata. Add-to-cart button is wired to the Svelte cart island via [data-sku]. Image is lazy-loaded with fixed dimensions (no CLS).",
		},
		{
			Name:     "product-grid",
			Category: "product",
			Template: `---
const { heading, products = [] } = Astro.props;
import ProductCard from "../components/product-card.astro";
---
<section class="section-shop">
  <div class="container">
    {heading && <h2>{heading}</h2>}
    <ul class="product-grid">
      {products.map((p) => (
        <li><ProductCard {...p} /></li>
      ))}
    </ul>
  </div>
</section>`,
			Props:      `{"type":"object","properties":{"heading":{"type":"string"},"products":{"type":"array","items":{"type":"object"}}}}`,
			CSSClasses: `["section-shop","product-grid"]`,
			Usage:      "Responsive product grid (2 col mobile, 3-4 col desktop). Pure HTML+CSS, no JS — filters and search go in a separate island.",
		},
		{
			Name:     "cart-drawer",
			Category: "cart",
			Template: `---
// Svelte island, hydrated only after first user interaction.
import Cart from "../islands/Cart.svelte";
---
<aside class="cart-drawer" id="cart-drawer" aria-label="Shopping cart">
  <Cart client:idle />
</aside>`,
			Props:      `{"type":"object","properties":{}}`,
			CSSClasses: `["cart-drawer"]`,
			Usage:      "Cart drawer wraps a Svelte island hydrated client:idle. Cart state lives in localStorage; checkout posts to /api/checkout which creates a Stripe Checkout session and redirects.",
		},
		{
			Name:     "stripe-checkout-cta",
			Category: "cta",
			Template: `---
const { label = "Checkout", successUrl, cancelUrl } = Astro.props;
---
<form method="POST" action="/api/checkout">
  <input type="hidden" name="success_url" value={successUrl} />
  <input type="hidden" name="cancel_url" value={cancelUrl} />
  <button type="submit" class="btn btn-primary">{label}</button>
</form>`,
			Props:      `{"type":"object","properties":{"label":{"type":"string"},"successUrl":{"type":"string"},"cancelUrl":{"type":"string"}}}`,
			CSSClasses: `[]`,
			Usage:      "Stripe Checkout redirect form. The /api/checkout endpoint creates a Checkout Session server-side and returns a 303 to the Stripe-hosted page. No client-side Stripe.js — minimal JS, payment data never touches the browser.",
		},
	}
	return seedComponents(ctx, q, siteID, defs)
}

func (k *ecommerceKit) applyCSS(ctx context.Context, q *store.Queries, siteID string) error {
	defs := []cssDef{
		{
			Name:     ".product-grid",
			Category: "product",
			CSS:      ".product-grid { display: grid; gap: 1rem; grid-template-columns: repeat(2, 1fr); list-style: none; padding: 0; } @media (min-width: 768px) { .product-grid { grid-template-columns: repeat(3, 1fr); gap: 1.5rem; } } @media (min-width: 1280px) { .product-grid { grid-template-columns: repeat(4, 1fr); } }",
			Usage:    "Responsive product grid. 2 col mobile, 3 col tablet, 4 col desktop.",
		},
		{
			Name:     ".product-card",
			Category: "product",
			CSS:      ".product-card { display: flex; flex-direction: column; gap: 0.5rem; } .product-card img { width: 100%; height: auto; aspect-ratio: 1; object-fit: cover; border-radius: 0.5rem; background: var(--bg-elevated, #f8fafc); } .product-card h3 { font-size: 1rem; margin: 0; } .product-card .price { margin: 0; font-weight: 600; } .product-card .btn { margin-top: auto; }",
			Usage:    "Square product card with title + price + add-to-cart pinned to bottom.",
		},
		{
			Name:     ".cart-drawer",
			Category: "cart",
			CSS:      ".cart-drawer { position: fixed; top: 0; right: 0; height: 100dvh; width: min(420px, 100vw); background: var(--bg, #fff); border-left: 1px solid var(--border, #e2e8f0); transform: translateX(100%); transition: transform 0.2s ease; z-index: 50; } .cart-drawer[data-open=\"true\"] { transform: translateX(0); }",
			Usage:    "Right-side cart drawer. Toggled via [data-open] attribute by the cart island.",
		},
	}
	return seedCSS(ctx, q, siteID, defs)
}

func (k *ecommerceKit) applyKnowledgebase(ctx context.Context, q *store.Queries, siteID string) error {
	entries := []kbDef{
		{Category: "ecom-architecture", Title: "Why Astro instead of Next / Qwik for e-com", Content: "Atomicsite's e-com kit uses Astro because: SSR product pages get real SEO (Product schema rendered on the server, crawlers see the price), the cart is a small Svelte island that hydrates lazily (no full-app hydration cost), and Stripe Checkout is a redirect — no in-page payment JS, so checkout latency is bounded by Stripe's servers, not bundle size. Qwik's resumability wins matter most for highly interactive SPAs; storefronts are mostly read-traffic with one interactive widget (the cart)."},
		{Category: "ecom-architecture", Title: "Cart state lives in localStorage", Content: "Cart island reads + writes to localStorage. On checkout, the form submits {sku, qty}[] to /api/checkout, which creates a Stripe Checkout Session and returns a 303 redirect to checkout.stripe.com. Never store cart server-side until the user actually checks out — saves DB writes and avoids cart-abandonment row spam."},
		{Category: "ecom-payments", Title: "Stripe Checkout vs Stripe Elements", Content: "Always start with Stripe Checkout (redirect to a Stripe-hosted page). Reasons: PCI scope is reduced (Stripe sees the card, not us), forex / 3DS / Apple Pay / Google Pay all handled by Stripe, conversion is similar to embedded Elements for most stores. Switch to Elements only when you need a deeply custom checkout flow that A/B tests show converts measurably better."},
		{Category: "ecom-schema", Title: "Product / Offer JSON-LD per page", Content: "Every product detail page emits Product schema with: name, description, sku, image (>=3 angles), offers (price, priceCurrency, availability InStock/OutOfStock, priceValidUntil), aggregateRating + review[] when reviews exist, brand. Atomicsite's product-card emits microdata; product detail pages add the full JSON-LD."},
		{Category: "ecom-images", Title: "Product image strategy", Content: "Three required variants per product: 600x600 thumbnail (grid), 1200x1200 detail (PDP zoom), 1200x630 OG image (social shares). Atomicsite's image pipeline auto-generates all three from one upload. Always set width + height on <img> to prevent CLS. Use loading=\"lazy\" except on the LCP image (first product on a category page or the main image on a PDP)."},
		{Category: "ecom-search", Title: "On-site search and filtering", Content: "For <500 SKUs: pre-build a JSON index at build time and ship a small Svelte island that filters in-memory. For >500 SKUs: use a hosted search service (Typesense, Meilisearch self-hosted, Algolia). Never use SQL LIKE on every keystroke — it doesn't scale and ranks poorly."},
		{Category: "ecom-conversion", Title: "Above-the-fold on a PDP", Content: "Product detail page above the fold: image (LCP), title, price, add-to-cart, ships-by-date, free-returns badge if applicable. Below: full description, specs, reviews, related products. Don't hide the price behind a tab. Don't gate add-to-cart behind an account creation step."},
		{Category: "ecom-legal", Title: "Required pages for e-com", Content: "Beyond the standard privacy / terms / cookies, e-com sites need: shipping policy, returns policy, T&Cs of sale (separate from site terms), VAT info if EU, accessibility statement (mandatory in EU from 2025-06-28). Atomicsite seeds shipping / returns / sale-terms pages on e-com sites in addition to the universal four."},
	}
	return seedKB(ctx, q, siteID, entries, 400)
}

func (k *ecommerceKit) applyGuardrails(ctx context.Context, q *store.Queries, siteID string) error {
	rules := []guardrailDef{
		{RuleType: "allow_block_type", Target: "*", Value: "hero,text,image,cta,feature_grid,product_card,product_grid,cart_drawer,faq,gallery,reviews", Severity: "error"},
		{RuleType: "forbid_pattern", Target: "block.data.text", Value: "lorem ipsum", Severity: "error"},
		{RuleType: "forbid_pattern", Target: "block.data.image", Value: "data:image", Severity: "error"},
	}
	return seedGuardrails(ctx, q, siteID, rules)
}

func (k *ecommerceKit) applyPages(ctx context.Context, q *store.Queries, siteID string) error {
	homeID := newKitID()
	if err := q.CreatePage(ctx, store.CreatePageParams{
		ID:        homeID,
		SiteID:    siteID,
		Title:     "Shop",
		Slug:      "/",
		Layout:    "landing",
		SortOrder: 0,
		ShowInNav: 1,
	}); err != nil {
		return err
	}
	homeBlocks := []blockDef{
		{BlockType: "hero", Data: `{"headline":"Built to last","subheadline":"Add a positioning statement that explains who this is for and why it's better than the obvious alternative.","primary_label":"Shop the collection","primary_href":"/products","secondary_label":"Read our story","secondary_href":"/about"}`},
		{BlockType: "feature_grid", Data: `{"component":"product-grid","heading":"Featured","products":[]}`},
	}
	if err := seedBlocks(ctx, q, homeID, homeBlocks); err != nil {
		return err
	}

	// Product index page (/products)
	productsID := newKitID()
	if err := q.CreatePage(ctx, store.CreatePageParams{
		ID:        productsID,
		SiteID:    siteID,
		Title:     "Products",
		Slug:      "/products",
		Layout:    "default",
		SortOrder: 10,
		ShowInNav: 1,
	}); err != nil {
		return err
	}
	if err := seedBlocks(ctx, q, productsID, []blockDef{
		{BlockType: "hero", Data: `{"headline":"All products","subheadline":"Browse the catalog. Use filters at top to narrow by category, price, or availability."}`},
		{BlockType: "feature_grid", Data: `{"component":"product-grid","products":[]}`},
	}); err != nil {
		return err
	}

	// E-com-specific legal pages.
	extras := []struct {
		title string
		slug  string
	}{
		{"Shipping", "/shipping"},
		{"Returns", "/returns"},
		{"Terms of Sale", "/terms-of-sale"},
	}
	for i, p := range extras {
		pageID := newKitID()
		if err := q.CreatePage(ctx, store.CreatePageParams{
			ID:        pageID,
			SiteID:    siteID,
			Title:     p.title,
			Slug:      p.slug,
			Layout:    "default",
			SortOrder: int64(200 + i),
			ShowInNav: 0,
		}); err != nil {
			return err
		}
	}

	return nil
}
