package sharedsecret

import "testing"

func TestVerifier_PrimaryOnly(t *testing.T) {
	v := NewVerifier("primary-key", "")
	s := NewSigner("primary-key")
	body := []byte(`{"event":"ping"}`)
	if !v.Verify(body, s.Sign(body)) {
		t.Fatal("expected primary signature to verify")
	}
}

func TestVerifier_SecondaryAcceptsOldSignature(t *testing.T) {
	v := NewVerifier("new-current", "old-previous")
	oldSig := NewSigner("old-previous").Sign([]byte(`{"event":"ping"}`))
	if !v.Verify([]byte(`{"event":"ping"}`), oldSig) {
		t.Fatal("expected secondary (old) signature to verify during grace window")
	}
}

func TestVerifier_RejectsForeign(t *testing.T) {
	v := NewVerifier("p", "s")
	if v.Verify([]byte("x"), NewSigner("not-our-key").Sign([]byte("x"))) {
		t.Fatal("expected unknown signer to be rejected")
	}
}

func TestVerifier_NotConfigured(t *testing.T) {
	v := NewVerifier("", "")
	if v.Configured() {
		t.Fatal("empty primary -> not configured")
	}
	if v.Verify([]byte("x"), "deadbeef") {
		t.Fatal("verify must be false when not configured")
	}
}

func TestVerifier_TamperedBodyRejected(t *testing.T) {
	v := NewVerifier("p", "")
	sig := NewSigner("p").Sign([]byte(`{"event":"ping"}`))
	if v.Verify([]byte(`{"event":"pwn"}`), sig) {
		t.Fatal("expected tampered body to be rejected")
	}
}

func TestSigner_NotConfigured(t *testing.T) {
	if got := NewSigner("").Sign([]byte("x")); got != "" {
		t.Fatalf("expected empty signature when not configured, got %q", got)
	}
}
