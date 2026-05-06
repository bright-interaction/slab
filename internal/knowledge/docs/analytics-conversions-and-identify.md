# Analytics: conversions, goals, and visitor identify

Atomicsite ships a first-party analytics layer that does three
things no third-party pixel can do for a CMS-built site: it counts
conversions against operator-defined goals, it identifies visitors
by email when they fill a form (no separate tag-management round
trip), and it stitches both to the cookie-consent record on the
same fingerprint so every aggregate is consent-aware by
construction.

This doc is what an agent needs to know to set up, debug, and
reason about that layer. The reference is the code in
`internal/conversions/`, `internal/identify/`, and
`internal/handlers/conversion_goals.go`.

## The data layer

Three tables back the system. None of them store raw IPs, raw
user-agent strings, or anything you cannot derive from a visitor
explicitly handing over an identifier (email, atomic.track event
name).

`visit_events` is the row-per-pageview table the parser writes.
Every row carries a 16-hex-char fingerprint computed as
SHA-256(IP, UA, Accept-Language, per-site salt) and truncated to 8
bytes. The same hash goes into `consent_records.fingerprint`, so
the JOIN that powers the cookie-funnel analytics works without a
mapping table.

`conversion_goals` is the operator-authored definition table. One
row per goal per site. Columns: `slug` (stable identifier the JS
SDK calls and the agent uses to refer to the goal), `name`
(human label), `match_type`, `match_value`, `value_cents` (optional
intent value in minor currency units), `value_currency` (3-letter
code, defaults to EUR), `active` (0 or 1).

`conversion_events` is the row-per-conversion table written
synchronously by the matcher. Each row is keyed by `(site_id,
goal_id, fingerprint, ts)` and carries the path that fired, the
session id when known, the value at fire time, and a JSON blob of
event properties.

## The three match strategies

Goal evaluation is one pure-Go function per strategy. Picking the
right one for the question you are trying to answer matters more
than the technical detail of how it fires.

`url_pattern` matches a glob against the visit_event path. The
glob is anchored at both ends and supports `*` as a wildcard that
does NOT cross `/`. A pattern of `/thank-you/*` matches
`/thank-you/lead` but not `/thank-you` (no segment) and not
`/thank-you/a/b` (wildcard would have to cross `/`). A trailing
`*` (or a bare `*`) is a "match everything to the end" wildcard.
Use this strategy for static thank-you pages, post-checkout
redirect URLs, and any goal where the conversion is "the visitor
landed on a specific URL".

`event_name` matches against the explicit event name posted to
`/t/event` (or fired via the `track_event` MCP tool). On the
rendered site, `window.atomic.track('signup', {valueCents: 4900})`
is the call. The beacon is consent-gated: when the visitor has
not granted analytics consent, `track()` is a quiet no-op so
opt-out users do not trigger goals. Use this for client-side
events that have no dedicated URL: a button click, a video
completed, a wizard step finished, a custom workflow in a SPA.

`form_submit` matches against the form's stable `form_id`. Fires
inside `FormHandler.Submit` after the form_submissions row lands
and after the per-IP rate-limit + honeypot have cleared. Use this
for goals tied to a specific form rather than a URL pattern that
might be shared across multiple forms.

A single visitor can hit multiple goals on the same pageview
(every active goal whose match strategy + value match is appended
as its own conversion_events row). Operators express OR via
multiple goals; AND is not first-class because the per-goal rate
becomes ambiguous when the denominator is "visitors who triggered
both goals".

## Conversion rate semantics

`get_goals_analytics` returns `conversion_rate_pct` as
`unique_converters / unique_visitors * 100` over the same time
window. The denominator is unique visitors (distinct fingerprints
in `visit_events.ts >= cutoff`), not pageviews and not sessions.
This makes the per-goal rate the answer to "what fraction of the
people who showed up triggered this goal", which is the framing a
marketing team actually wants when comparing sources.

`unique_converters` is distinct fingerprints in
`conversion_events.ts >= cutoff` for the goal. A visitor who
triggers a goal twice in the window counts once toward
unique_converters and twice toward `conversions`. Use the
former for "how many people", the latter for "how often".

`total_value_cents` sums `value_cents` across every event in the
window. It is intent-value in P1, not realised revenue. Phase
31.4 closes the loop with a CRM revenue round-trip; until then,
treat this number as a planning input.

## The identify path

Identifying a visitor by email is the moat hinge: it turns an
anonymous fingerprint into a known contact while preserving the
full UTM + page history that fingerprint already accumulated.

Two paths fire identify, and both run through the same Recorder
in `internal/identify/`:

1. `POST /t/identify` is the explicit SDK call. Site code (a
post-checkout script, a newsletter-confirmed redirect, an
auth-elsewhere callback) posts `{siteId, email, page}`. The
fingerprint comes from the cookie. The Recorder validates the
email shape via `net/mail.ParseAddress`, calls
`SetVisitSessionEmail` (which uses `COALESCE(NULLIF(...))` so
re-identifying preserves the original `identified_at`
timestamp), then fires a best-effort crmsync identified event so
BrightCRM picks up the freshly-named contact.

2. `FormHandler.Submit` runs the same Recorder automatically when
a submitted form carries an email-shaped value. The matcher
checks field names first (regex on `email`, `your-email`,
`e_mail`, `e-mail`, `emailaddress`, `email_address`,
`user_email`, `contact_email`) then falls back to a value-shape
scan if no labelled field matched. This means an atomicsite form
identifies its visitor without any extra wiring on the rendered
site.

Identify is rate-limited (10 burst, 5 per minute sustained). The
endpoint returns 204 on success, 400 on a malformed email, 404
when no session exists for the (site, fingerprint) pair (the
visitor never granted analytics consent so no `visit_sessions`
row was upserted).

## Privacy boundaries

Identified-tier data (the email, the visitor_id, the
identified_at timestamp) is never exposed via MCP. The
`get_goals_analytics` tool returns aggregates only: counts and
rates, no email lists. The same constraint runs through the
admin-only `/api/sites/{id}/analytics/sessions?identified=true`
which is gated by site-access and not surfaced as an MCP
resource. An agent that needs per-email drill-down works through
the operator: "click into the Identified Sessions panel to see
who's behind that conversion".

Cookie consent gates analytics writes through the engagement
beacon. `window.atomic.track()` checks the same consent state
the engagement beacon uses, so a visitor on reject-all never
fires a goal. Server-side `/t/event` does the same fingerprint
lookup as `/t/pageview`, so a visitor without a session (no
prior consent) cannot create a conversion_events row.

## The end-to-end attribution chain

When everything is wired, the chain looks like this:

1. BrightCRM sends a sequence email. `injectTracking` stamps
   `utm_source=brightcrm&utm_medium=email&utm_campaign={seq}&utm_content={msg_id}`
   onto every href before the click-proxy wrap.
2. The visitor clicks. The click-proxy at `/t/{tid}/click?url=...`
   redirects to the atomicsite landing page with the UTM intact.
3. Atomicsite parser writes `visit_events.utm_source=brightcrm`,
   `utm_medium=email`, `utm_campaign={seq}`, `utm_content={msg_id}`.
4. CookieProof banner shows. Visitor accepts analytics. A
   visit_session row gets upserted, identified_at gets stamped
   with empty email.
5. Visitor browses, hits a goal-matching URL or fires
   `atomic.track('signup')`. A conversion_events row lands.
6. Visitor fills a form. The form-submit handler picks the email,
   the Recorder writes it onto the visit_session, fires
   crmsync identified event with the full UTM + path metadata.
7. BrightCRM matches the identified event to the contact and
   stamps the source attribution onto the lead.
8. (Phase 31.4, future) BrightCRM moves the deal to WON, posts a
   /t/revenue event back, atomicsite stores the realised value
   keyed to the original `utm_campaign`. The Analytics tab now
   shows revenue per source, end to end, single source of truth.

This is the chain no other CMS can offer: same-roof CMS plus
cookie consent plus analytics plus CRM, all under one fingerprint
that nobody else needs to share. Treat it as the moat thesis when
explaining the platform.
