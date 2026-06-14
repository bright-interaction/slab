package atrest

import "testing"

func key() []byte { return []byte("01234567890123456789012345678901") }

func TestRoundTrip(t *testing.T) {
	c := New(key())
	if !c.Enabled() {
		t.Fatal("want enabled cipher")
	}
	for _, pt := range []string{"hunter2", `{"api_key":"sk_live_123"}`, "JBSWY3DPEHPK3PXP"} {
		enc, err := c.Encrypt(pt)
		if err != nil {
			t.Fatalf("encrypt: %v", err)
		}
		if enc == pt || !IsEncrypted(enc) {
			t.Fatalf("ciphertext not prefixed/changed: %q", enc)
		}
		got, err := c.Decrypt(enc)
		if err != nil || got != pt {
			t.Fatalf("decrypt = %q err=%v; want %q", got, err, pt)
		}
	}
}

func TestEmptyAndLegacyPlaintextPassThrough(t *testing.T) {
	c := New(key())
	if enc, _ := c.Encrypt(""); enc != "" {
		t.Errorf("empty encrypt = %q; want empty", enc)
	}
	// A legacy plaintext value (no prefix) decrypts to itself.
	if got, err := c.Decrypt("legacy-plaintext-secret"); err != nil || got != "legacy-plaintext-secret" {
		t.Errorf("legacy decrypt = %q err=%v; want passthrough", got, err)
	}
}

func TestPassthroughCipherWithoutKey(t *testing.T) {
	c := New(nil) // no key -> passthrough
	if c.Enabled() {
		t.Fatal("nil-key cipher must not be enabled")
	}
	enc, _ := c.Encrypt("secret")
	if enc != "secret" {
		t.Errorf("passthrough encrypt = %q; want plaintext", enc)
	}
	got, _ := c.Decrypt("secret")
	if got != "secret" {
		t.Errorf("passthrough decrypt = %q; want plaintext", got)
	}
}

func TestEncryptedValueWithoutKeyErrors(t *testing.T) {
	enc, _ := New(key()).Encrypt("secret")
	// Operator removed the key after encrypting: decrypt must error, not
	// silently hand back ciphertext.
	if _, err := New(nil).Decrypt(enc); err == nil {
		t.Fatal("want error decrypting ciphertext without a key")
	}
}

func TestWrongKeyFails(t *testing.T) {
	enc, _ := New(key()).Encrypt("secret")
	other := New([]byte("ABCDEFABCDEFABCDEFABCDEFABCDEF01"))
	if _, err := other.Decrypt(enc); err == nil {
		t.Fatal("want error decrypting with wrong key")
	}
}
