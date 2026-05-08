package shield

import (
	"context"
	"database/sql"
	"strings"
	"testing"
	"time"

	_ "modernc.org/sqlite"
)

func testKey() []byte {
	return []byte("01234567890123456789012345678901")
}

// stores returns one Store per backing implementation so the table-
// driven tests below cover the in-memory + SQL paths uniformly.
func stores(t *testing.T) []struct {
	name  string
	build func(t *testing.T) Store
} {
	return []struct {
		name  string
		build func(t *testing.T) Store
	}{
		{
			name: "memory",
			build: func(t *testing.T) Store {
				return NewMemoryStore()
			},
		},
		{
			name: "sqlite",
			build: func(t *testing.T) Store {
				t.Helper()
				db, err := sql.Open("sqlite", ":memory:")
				if err != nil {
					t.Fatalf("open sqlite: %v", err)
				}
				t.Cleanup(func() { _ = db.Close() })
				if _, err := db.Exec(`PRAGMA foreign_keys = ON;`); err != nil {
					t.Fatalf("fk: %v", err)
				}
				if _, err := db.Exec(testSchemaSQL); err != nil {
					t.Fatalf("ddl: %v", err)
				}
				return NewSQLStore(db)
			},
		},
	}
}

const testSchemaSQL = `
CREATE TABLE IF NOT EXISTS shield_sessions (
    id          TEXT PRIMARY KEY,
    created_at  TEXT NOT NULL DEFAULT (datetime('now')),
    last_seen   TEXT NOT NULL DEFAULT (datetime('now')),
    expires_at  TEXT NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_shield_sessions_expires ON shield_sessions(expires_at);
CREATE TABLE IF NOT EXISTS shield_tokens (
    token       TEXT PRIMARY KEY,
    session_id  TEXT NOT NULL REFERENCES shield_sessions(id) ON DELETE CASCADE,
    kind        TEXT NOT NULL,
    hint        TEXT NOT NULL DEFAULT '',
    ciphertext  TEXT NOT NULL,
    created_at  TEXT NOT NULL DEFAULT (datetime('now'))
);
`

func runOnAllStores(t *testing.T, fn func(t *testing.T, store Store)) {
	t.Helper()
	for _, sc := range stores(t) {
		sc := sc
		t.Run(sc.name, func(t *testing.T) {
			store := sc.build(t)
			fn(t, store)
		})
	}
}

func TestNewSessionCreatesAndTouches(t *testing.T) {
	runOnAllStores(t, func(t *testing.T, store Store) {
		ctx := context.Background()
		s1, err := NewSession(ctx, store, "session-A", testKey(), 10*time.Minute)
		if err != nil {
			t.Fatalf("create: %v", err)
		}
		if s1.ID != "session-A" {
			t.Fatalf("wrong id: %s", s1.ID)
		}
		s2, err := NewSession(ctx, store, "session-A", testKey(), 10*time.Minute)
		if err != nil {
			t.Fatalf("touch: %v", err)
		}
		if s2.ID != s1.ID {
			t.Fatalf("session id changed across calls")
		}
	})
}

func TestNewSessionRejectsBadKey(t *testing.T) {
	runOnAllStores(t, func(t *testing.T, store Store) {
		ctx := context.Background()
		if _, err := NewSession(ctx, store, "id", []byte("short"), time.Minute); err == nil {
			t.Fatal("expected error for short key")
		}
	})
}

func TestTokenizeAndResolveRoundTrip(t *testing.T) {
	runOnAllStores(t, func(t *testing.T, store Store) {
		ctx := context.Background()
		s, err := NewSession(ctx, store, "session-1", testKey(), time.Minute)
		if err != nil {
			t.Fatalf("session: %v", err)
		}
		cases := []struct {
			name  string
			kind  Kind
			value string
			hint  string
		}{
			{"email_with_domain", KindEmail, "anna@andersson-law.se", "domain=andersson-law.se,len=22"},
			{"name_with_initials", KindName, "Anna Andersson", "initials=AA"},
			{"phone_se", KindPhone, "+46701234567", "country=SE"},
			{"freeform_no_hint", KindFreeform, "secret blob", ""},
		}
		for _, tc := range cases {
			t.Run(tc.name, func(t *testing.T) {
				marker, err := s.Tokenize(ctx, tc.kind, tc.value, tc.hint)
				if err != nil {
					t.Fatalf("Tokenize: %v", err)
				}
				if !strings.HasPrefix(marker, "[shield:"+string(tc.kind)+":tok_") {
					t.Fatalf("marker format unexpected: %q", marker)
				}
				if strings.Contains(marker, tc.value) {
					t.Fatalf("marker leaks raw value: %q contains %q", marker, tc.value)
				}
				groups := MarkerPattern.FindStringSubmatch(marker)
				if len(groups) < 3 {
					t.Fatalf("marker did not parse: %q", marker)
				}
				pt, err := s.Resolve(ctx, groups[2])
				if err != nil {
					t.Fatalf("Resolve: %v", err)
				}
				if pt != tc.value {
					t.Fatalf("resolve mismatch: got %q want %q", pt, tc.value)
				}
			})
		}
	})
}

func TestResolveRejectsCrossSessionToken(t *testing.T) {
	runOnAllStores(t, func(t *testing.T, store Store) {
		ctx := context.Background()
		a, _ := NewSession(ctx, store, "session-A", testKey(), time.Minute)
		b, _ := NewSession(ctx, store, "session-B", testKey(), time.Minute)
		marker, err := a.Tokenize(ctx, KindEmail, "x@y.z", "")
		if err != nil {
			t.Fatalf("tokenize: %v", err)
		}
		groups := MarkerPattern.FindStringSubmatch(marker)
		if _, err := b.Resolve(ctx, groups[2]); err != ErrTokenNotFound {
			t.Fatalf("cross-session resolve: got %v, want ErrTokenNotFound", err)
		}
	})
}

func TestTokenizeRejectsUnknownKind(t *testing.T) {
	runOnAllStores(t, func(t *testing.T, store Store) {
		ctx := context.Background()
		s, _ := NewSession(ctx, store, "id", testKey(), time.Minute)
		if _, err := s.Tokenize(ctx, Kind("bogus"), "x", ""); err != ErrUnknownKind {
			t.Fatalf("got %v, want ErrUnknownKind", err)
		}
	})
}

func TestTokenizeRejectsNonWhitelistedHint(t *testing.T) {
	runOnAllStores(t, func(t *testing.T, store Store) {
		ctx := context.Background()
		s, _ := NewSession(ctx, store, "id", testKey(), time.Minute)
		if _, err := s.Tokenize(ctx, KindEmail, "x@y.z", "raw=x@y.z"); err == nil {
			t.Fatal("expected hint validation error")
		}
	})
}

func TestSweepExpiredRemovesStaleSessions(t *testing.T) {
	runOnAllStores(t, func(t *testing.T, store Store) {
		ctx := context.Background()
		past := time.Now().Add(-1 * time.Hour)
		future := time.Now().Add(1 * time.Hour)
		if err := store.InsertSession(ctx, "old", past); err != nil {
			t.Fatal(err)
		}
		if err := store.InsertSession(ctx, "fresh", future); err != nil {
			t.Fatal(err)
		}
		if err := SweepExpired(ctx, store); err != nil {
			t.Fatal(err)
		}
		if _, found, _ := store.GetSession(ctx, "old"); found {
			t.Error("stale session survived sweep")
		}
		if _, found, _ := store.GetSession(ctx, "fresh"); !found {
			t.Error("fresh session was incorrectly swept")
		}
	})
}

type contact struct {
	ID      string `json:"id"      shield:"id"`
	Email   string `json:"email"   shield:"tokenize,kind=email,hint=domain len"`
	Name    string `json:"name"    shield:"tokenize,kind=name,hint=initials"`
	Stage   string `json:"stage"`
	Company string `json:"company" shield:"tokenize,kind=company,hint=len"`
}

type lead struct {
	ID       string    `json:"id" shield:"id"`
	Contacts []contact `json:"contacts"`
	Notes    string    `json:"notes"`
}

func TestShieldStructWalk(t *testing.T) {
	runOnAllStores(t, func(t *testing.T, store Store) {
		ctx := context.Background()
		s, _ := NewSession(ctx, store, "id", testKey(), time.Minute)
		c := contact{
			ID:      "c-1",
			Email:   "anna@andersson-law.se",
			Name:    "Anna Andersson",
			Stage:   "qualified",
			Company: "Andersson Advokat",
		}
		if err := s.Shield(ctx, &c); err != nil {
			t.Fatalf("Shield: %v", err)
		}
		if c.ID != "c-1" {
			t.Fatalf("ID was tokenized: %q", c.ID)
		}
		if c.Stage != "qualified" {
			t.Fatalf("Stage was tokenized: %q", c.Stage)
		}
		if !strings.HasPrefix(c.Email, "[shield:email:tok_") {
			t.Fatalf("Email not shielded: %q", c.Email)
		}
		if !strings.Contains(c.Email, "domain=andersson-law.se") {
			t.Fatalf("email domain hint missing: %q", c.Email)
		}
		if !strings.HasPrefix(c.Name, "[shield:name:tok_") {
			t.Fatalf("Name not shielded: %q", c.Name)
		}
		if !strings.Contains(c.Name, "initials=AA") {
			t.Fatalf("name initials missing: %q", c.Name)
		}
		if strings.Contains(c.Email, "anna@andersson-law.se") {
			t.Fatalf("email leaks raw value: %q", c.Email)
		}
		if strings.Contains(c.Name, "Anna Andersson") {
			t.Fatalf("name leaks raw value: %q", c.Name)
		}
	})
}

func TestShieldThenUnshieldRoundTrip(t *testing.T) {
	runOnAllStores(t, func(t *testing.T, store Store) {
		ctx := context.Background()
		s, _ := NewSession(ctx, store, "id", testKey(), time.Minute)
		original := contact{
			ID:    "c-1",
			Email: "x@y.z",
			Name:  "Test User",
			Stage: "open",
		}
		working := original
		if err := s.Shield(ctx, &working); err != nil {
			t.Fatalf("Shield: %v", err)
		}
		if err := s.Unshield(ctx, &working); err != nil {
			t.Fatalf("Unshield: %v", err)
		}
		if working != original {
			t.Fatalf("round-trip mismatch:\nwant %+v\n got %+v", original, working)
		}
	})
}

func TestShieldNestedSlice(t *testing.T) {
	runOnAllStores(t, func(t *testing.T, store Store) {
		ctx := context.Background()
		s, _ := NewSession(ctx, store, "id", testKey(), time.Minute)
		l := lead{
			ID: "l-1",
			Contacts: []contact{
				{ID: "c-1", Email: "a@one.se", Name: "A One", Stage: "x"},
				{ID: "c-2", Email: "b@two.se", Name: "B Two", Stage: "y"},
			},
		}
		if err := s.Shield(ctx, &l); err != nil {
			t.Fatalf("Shield: %v", err)
		}
		for i, c := range l.Contacts {
			if !strings.HasPrefix(c.Email, "[shield:email:") {
				t.Fatalf("contacts[%d].Email not shielded: %q", i, c.Email)
			}
		}
	})
}

func TestUnshieldStringHandlesMultipleMarkers(t *testing.T) {
	runOnAllStores(t, func(t *testing.T, store Store) {
		ctx := context.Background()
		s, _ := NewSession(ctx, store, "id", testKey(), time.Minute)
		a, _ := s.Tokenize(ctx, KindEmail, "x@y.z", "")
		b, _ := s.Tokenize(ctx, KindName, "Bob", "")
		in := "Send to " + a + " and CC " + b + " please"
		out, err := s.UnshieldString(ctx, in)
		if err != nil {
			t.Fatalf("UnshieldString: %v", err)
		}
		if !strings.Contains(out, "x@y.z") || !strings.Contains(out, "Bob") {
			t.Fatalf("not all markers resolved: %q", out)
		}
		if strings.Contains(out, "[shield:") {
			t.Fatalf("residual markers in output: %q", out)
		}
	})
}

func TestUnshieldJSONNestedStrings(t *testing.T) {
	runOnAllStores(t, func(t *testing.T, store Store) {
		ctx := context.Background()
		s, _ := NewSession(ctx, store, "id", testKey(), time.Minute)
		m1, _ := s.Tokenize(ctx, KindEmail, "anna@x.se", "")
		m2, _ := s.Tokenize(ctx, KindName, "Anna", "")
		body := `{"to": "` + m1 + `", "cc": ["` + m2 + `"], "subject": "Hi ` + m2 + `"}`
		out, err := s.UnshieldJSON(ctx, []byte(body))
		if err != nil {
			t.Fatalf("UnshieldJSON: %v", err)
		}
		str := string(out)
		if !strings.Contains(str, "anna@x.se") || !strings.Contains(str, "Anna") {
			t.Fatalf("missing resolved values: %s", str)
		}
		if strings.Contains(str, "[shield:") {
			t.Fatalf("residual markers: %s", str)
		}
	})
}

func TestSessionExpiredOnRevisit(t *testing.T) {
	runOnAllStores(t, func(t *testing.T, store Store) {
		ctx := context.Background()
		// Plant an already-expired session row.
		if err := store.InsertSession(ctx, "id-expired", time.Now().Add(-1*time.Minute)); err != nil {
			t.Fatal(err)
		}
		if _, err := NewSession(ctx, store, "id-expired", testKey(), time.Minute); err != ErrSessionExpired {
			t.Fatalf("got %v, want ErrSessionExpired", err)
		}
	})
}

func TestParseShieldTagErrors(t *testing.T) {
	cases := []string{
		"",                              // empty
		"keep",                          // missing tokenize
		"tokenize",                      // missing kind
		"tokenize,kind=bogus",           // unknown kind
		"tokenize,kind=email,foo=bar",   // unknown key
		"tokenize,kind=email,hint",      // malformed hint pair
	}
	for _, c := range cases {
		if _, _, err := parseShieldTag(c); err == nil {
			t.Errorf("expected error for tag %q", c)
		}
	}
}
