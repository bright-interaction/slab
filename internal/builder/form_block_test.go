package builder

import (
	"strings"
	"testing"
)

// Sprint 5 quick-win (2026-05-24): form block renderer tests. Cover
// the classic single-step form (existing behaviour, byte-identical
// for sites that never set visible_if / step) plus the two new
// additive behaviours: conditional field visibility and multi-step
// grouping.

func TestRenderFormBlock_ClassicSingleStepUnchanged(t *testing.T) {
	out := renderFormBlock(map[string]any{
		"action": "/submit",
		"fields": []any{
			map[string]any{"name": "email", "type": "email", "label": "Email", "required": true},
			map[string]any{"name": "msg", "type": "textarea", "label": "Message"},
		},
	})
	for _, want := range []string{
		`action="/submit"`,
		`name="email"`,
		`required`,
		`name="msg"`,
		`type="submit"`,
	} {
		if !strings.Contains(out, want) {
			t.Errorf("classic form missing %q in output:\n%s", want, out)
		}
	}
	// Single-step forms must NOT emit a fieldset wrapper, must NOT emit
	// the next/prev buttons, and must NOT emit the runtime script.
	if strings.Contains(out, "data-form-step") {
		t.Errorf("single-step form should NOT emit data-form-step; got:\n%s", out)
	}
	if strings.Contains(out, "data-form-next") {
		t.Errorf("single-step form should NOT emit data-form-next; got:\n%s", out)
	}
	if strings.Contains(out, "<script>") {
		t.Errorf("single-step form without visible_if should NOT emit inline script; got:\n%s", out)
	}
}

func TestRenderFormBlock_VisibleIfEmitsDataAttrsAndScript(t *testing.T) {
	out := renderFormBlock(map[string]any{
		"action": "/submit",
		"fields": []any{
			map[string]any{"name": "contact_pref", "type": "select", "label": "Prefer", "options": []any{"email", "phone"}},
			map[string]any{
				"name": "phone", "type": "tel", "label": "Phone",
				"required":          true,
				"visible_if_field":  "contact_pref",
				"visible_if_equals": "phone",
			},
		},
	})
	if !strings.Contains(out, `data-visible-if-field="contact_pref"`) {
		t.Errorf("expected data-visible-if-field attr; got:\n%s", out)
	}
	if !strings.Contains(out, `data-visible-if-equals="phone"`) {
		t.Errorf("expected data-visible-if-equals attr; got:\n%s", out)
	}
	// Conditional required fields use data-required, NOT native required.
	if strings.Contains(out, `name="phone" placeholder="" required`) {
		t.Errorf("conditional required field should use data-required, not native required (the form would reject submit on hidden field); got:\n%s", out)
	}
	if !strings.Contains(out, "data-required") {
		t.Errorf("expected data-required on conditional field; got:\n%s", out)
	}
	if !strings.Contains(out, "<script>") {
		t.Errorf("expected inline form logic script when conditional fields present; got:\n%s", out)
	}
	if !strings.Contains(out, "evalConditional") {
		t.Errorf("expected conditional evaluator in script; got:\n%s", out)
	}
}

func TestRenderFormBlock_MultiStepGroupsFieldsAndAddsNav(t *testing.T) {
	out := renderFormBlock(map[string]any{
		"action": "/submit",
		"fields": []any{
			map[string]any{"name": "name", "type": "text", "label": "Name", "step": float64(1)},
			map[string]any{"name": "email", "type": "email", "label": "Email", "step": float64(1)},
			map[string]any{"name": "phone", "type": "tel", "label": "Phone", "step": float64(2)},
			map[string]any{"name": "notes", "type": "textarea", "label": "Notes", "step": float64(2)},
		},
		"next_label":     "Continue",
		"previous_label": "Go back",
	})
	if !strings.Contains(out, `data-form-step="1"`) || !strings.Contains(out, `data-form-step="2"`) {
		t.Errorf("expected two step fieldsets; got:\n%s", out)
	}
	if !strings.Contains(out, "data-form-next") || !strings.Contains(out, "data-form-prev") {
		t.Errorf("expected next + prev buttons; got:\n%s", out)
	}
	if !strings.Contains(out, "Continue") || !strings.Contains(out, "Go back") {
		t.Errorf("expected custom step nav labels honoured; got:\n%s", out)
	}
	// Step 1 visible by default (no hidden), step 2 hidden until next.
	if !strings.Contains(out, `data-form-step="2" hidden`) {
		t.Errorf("expected step 2 to start hidden; got:\n%s", out)
	}
	if !strings.Contains(out, "showStep") {
		t.Errorf("expected step-switching script; got:\n%s", out)
	}
	// Submit button starts hidden so single-step finishers don't see
	// it before the last step.
	if !strings.Contains(out, `data-form-submit hidden`) {
		t.Errorf("expected submit button to start hidden in multi-step; got:\n%s", out)
	}
}

func TestRenderFormBlock_ConditionalEmptyEqualsMeansAnyValue(t *testing.T) {
	// visible_if_equals omitted = "show when the gating field has ANY
	// non-empty value". The renderer emits an empty equals attr; the
	// script evaluates val !== ''.
	out := renderFormBlock(map[string]any{
		"action": "/submit",
		"fields": []any{
			map[string]any{"name": "interested", "type": "checkbox", "label": "Interested?"},
			map[string]any{
				"name": "details", "type": "textarea", "label": "Tell us more",
				"visible_if_field": "interested",
			},
		},
	})
	if !strings.Contains(out, `data-visible-if-field="interested"`) {
		t.Errorf("expected data-visible-if-field on details; got:\n%s", out)
	}
	if !strings.Contains(out, `data-visible-if-equals=""`) {
		t.Errorf("expected empty data-visible-if-equals attr; got:\n%s", out)
	}
}
