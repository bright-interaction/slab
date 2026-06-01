package server

import (
	"database/sql"
	"testing"

	"github.com/alicebob/miniredis/v2"

	"github.com/brightinteraction/atomicsite/internal/config"
	"github.com/brightinteraction/atomicsite/internal/shield"
)

func TestBuildShieldStore_DefaultsToSQLStore(t *testing.T) {
	// Empty ShieldRedisURL -> SQLStore even with a non-nil cfg.
	store, kind, err := buildShieldStore(&config.Config{}, &sql.DB{})
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if kind != "sqlite" {
		t.Errorf("expected kind=sqlite, got %q", kind)
	}
	if _, ok := store.(*shield.SQLStore); !ok {
		t.Errorf("expected *shield.SQLStore, got %T", store)
	}
}

func TestBuildShieldStore_NilConfigDefaultsToSQLStore(t *testing.T) {
	// nil cfg should still hand back a usable SQLStore so a misconfigured
	// boot does not crash mcpServer.WithShield.
	store, kind, err := buildShieldStore(nil, &sql.DB{})
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if kind != "sqlite" {
		t.Errorf("expected kind=sqlite for nil cfg, got %q", kind)
	}
	if store == nil {
		t.Error("store must not be nil even with nil cfg")
	}
}

func TestBuildShieldStore_RedisURLPicksRedis(t *testing.T) {
	mr := miniredis.RunT(t)
	url := "redis://" + mr.Addr()
	store, kind, err := buildShieldStore(&config.Config{ShieldRedisURL: url}, nil)
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if kind != "redis" {
		t.Errorf("expected kind=redis, got %q", kind)
	}
	rs, ok := store.(*shield.RedisStore)
	if !ok {
		t.Fatalf("expected *shield.RedisStore, got %T", store)
	}
	// Prefix should default to "" so the store applies its own "shield:" default.
	if rs.Prefix != "" {
		t.Errorf("expected default empty prefix, got %q", rs.Prefix)
	}
}

func TestBuildShieldStore_RedisPrefixPropagates(t *testing.T) {
	mr := miniredis.RunT(t)
	store, _, err := buildShieldStore(&config.Config{
		ShieldRedisURL:    "redis://" + mr.Addr(),
		ShieldRedisPrefix: "tenant-acme:shield:",
	}, nil)
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	rs := store.(*shield.RedisStore)
	if rs.Prefix != "tenant-acme:shield:" {
		t.Errorf("prefix did not propagate; got %q", rs.Prefix)
	}
}

func TestBuildShieldStore_GarbageRedisURLReturnsError(t *testing.T) {
	_, _, err := buildShieldStore(&config.Config{ShieldRedisURL: "not-a-real-url"}, nil)
	if err == nil {
		t.Error("garbage URL should return an error; caller logs it and disables shield")
	}
}
