# Forms UX

Forms convert at the rate the operator's funnel can sustain. Bad forms tank conversion silently. The agent's job is to keep them simple, accessible, fast to fill, and honest about state.

## Single column

Multi-column forms read as bureaucratic and increase eye-jump. Stick to one column. The exception is small paired fields (city + zip, expiry month + year); pair them on a single row only when both fit comfortably.

## Field count

Every field is friction. Drop fields the operator does not need today. The default lead-capture form ships email + name; phone, company, and message fields are optional and the agent should ask before adding them. A 9-field form converts at half the rate of a 4-field form.

## Labels

Every input has a visible label above the field. Placeholder text is supplementary, not a label substitute (placeholders disappear on focus and screen readers cannot announce them as field names).

```astro
<label for="email">Email</label>
<input
  id="email"
  name="email"
  type="email"
  autocomplete="email"
  required
  placeholder="you@company.com"
>
```

## autocomplete

Email, name, tel, address, password, organization fields all need `autocomplete` attributes per the HTML autofill spec. Common values: `email`, `name` (or `given-name` + `family-name`), `tel`, `organization`, `street-address`, `current-password`, `new-password`, `one-time-code`.

The eval engine checks for these and the form-ux skill grades on completion-rate signals like autofill availability.

## Input types

Use the right input type. `type="email"` triggers email keyboard on mobile, basic format validation, autocomplete hint. Same for `type="tel"`, `type="url"`, `type="number"`, `type="date"`. Default `type="text"` only when nothing more specific applies.

## Required vs optional

Default to required. Mark optional fields with " (optional)" in the label. Required fields do not need an asterisk; users assume required.

## Validation timing

Validate on blur, not on every keystroke. On-keystroke validation flickers and reads as nagging. Pattern:

1. User leaves the field (blur)
2. If invalid, show the error inline below the field with `aria-describedby`
3. As they edit again, hide the error on input until next blur
4. On submit, focus the first invalid field

## Error messages

Specific and actionable. "Email is invalid" is bad. "Email needs an @ and a domain" is better. "We could not find a working server for that email" is best (when the validator can detect that). Errors are help, not punishment.

## Buttons

Submit button below the form, full width on mobile, fixed-width-with-label on desktop. Label states the action: "Send message", "Start trial", "Get the report". Avoid "Submit"; it is the technical name, not what the user is doing.

## Loading state

When the user submits, the button enters a loading state immediately:

- Disable the button (prevent double-submit)
- Replace label with "Sending..." or a spinner
- Re-enable on success or error
- Announce success / error via `aria-live="polite"` for screen readers

## Success state

Replace the form with a success message in place. Do not redirect to a new page; the user just gave you their email and a redirect feels like a brush-off. The success message confirms the action and previews next steps:

> Thanks. We will reach out within one business day. While you wait, here is the [GDPR guide] you might find useful.

## Honeypot, not CAPTCHA

CAPTCHAs add friction and accessibility issues for marginal spam reduction on low-volume forms. Use a honeypot field (hidden via CSS, not display:none which screen readers might announce). Atomic Site's form block emits a honeypot by default; do not remove it.

## What you must not do

- Do not split a 5-field form across 3 steps with a progress bar. That is for 20-field flows.
- Do not require phone for an inbound newsletter signup.
- Do not validate password strength before the user has typed 6 characters.
- Do not show validation errors before first blur.
- Do not auto-uppercase or auto-lowercase input. Let the user write what they wrote.

## What the eval grades

`internal/eval/accessibility.go` checks: every input has a label, autocomplete attributes are present where expected, error messages are linked via `aria-describedby`, focus moves to first error on submit. The form-ux discipline above passes all of these by default.
