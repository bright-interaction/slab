package starterkits

import (
	"context"
	"testing"

	"github.com/brightinteraction/atomicsite/internal/store"
)

type fakeKit struct {
	id          string
	name        string
	description string
	targets     []string
	applyCalled bool
	applyErr    error
}

func (f *fakeKit) ID() string                { return f.id }
func (f *fakeKit) Name() string              { return f.name }
func (f *fakeKit) Description() string       { return f.description }
func (f *fakeKit) TargetSiteTypes() []string { return f.targets }
func (f *fakeKit) Apply(_ context.Context, _ *store.Queries, _ string) error {
	f.applyCalled = true
	return f.applyErr
}

func TestRegistry_RegisterGet(t *testing.T) {
	r := NewRegistry()
	k := &fakeKit{id: "fake", name: "Fake", description: "desc", targets: []string{"b2b"}}
	r.Register(k)

	got, ok := r.Get("fake")
	if !ok {
		t.Fatalf("expected fake kit to be registered")
	}
	if got.ID() != "fake" || got.Name() != "Fake" {
		t.Fatalf("unexpected kit returned: id=%q name=%q", got.ID(), got.Name())
	}
}

func TestRegistry_GetUnknown(t *testing.T) {
	r := NewRegistry()
	if _, ok := r.Get("does-not-exist"); ok {
		t.Fatalf("expected Get(unknown) to return false")
	}
}

func TestRegistry_List_SortedByID(t *testing.T) {
	r := NewRegistry()
	r.Register(&fakeKit{id: "zeta"})
	r.Register(&fakeKit{id: "alpha"})
	r.Register(&fakeKit{id: "mike"})

	got := r.List()
	if len(got) != 3 {
		t.Fatalf("expected 3 kits, got %d", len(got))
	}
	wantOrder := []string{"alpha", "mike", "zeta"}
	for i, w := range wantOrder {
		if got[i].ID() != w {
			t.Errorf("List()[%d] = %q, want %q", i, got[i].ID(), w)
		}
	}
}

func TestRegistry_RegisterNilSafe(t *testing.T) {
	r := NewRegistry()
	r.Register(nil)
	if got := r.List(); len(got) != 0 {
		t.Fatalf("expected nil register to be ignored, got %d kits", len(got))
	}
}

func TestRegistry_RegisterOverwrites(t *testing.T) {
	r := NewRegistry()
	r.Register(&fakeKit{id: "k", name: "first"})
	r.Register(&fakeKit{id: "k", name: "second"})

	got, ok := r.Get("k")
	if !ok {
		t.Fatalf("expected kit registered")
	}
	if got.Name() != "second" {
		t.Fatalf("expected later registration to win, got %q", got.Name())
	}
}
