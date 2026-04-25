package deploy

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const fakePEM = "-----BEGIN OPENSSH PRIVATE KEY-----\nfakekeycontents\n-----END OPENSSH PRIVATE KEY-----\n"

func validRsyncTarget() Target {
	return Target{
		ID:     "tgt_1",
		SiteID: "site_1",
		Name:   "prod",
		Kind:   "rsync",
		Config: map[string]any{
			"host":            "deploy.example.com",
			"user":            "deployer",
			"port":            float64(2222),
			"path":            "/var/www/example",
			"private_key_pem": fakePEM,
			"public_url":      "https://example.com",
		},
	}
}

func TestRsyncDeployer_Kind(t *testing.T) {
	d := NewRsyncDeployer()
	if got := d.Kind(); got != "rsync" {
		t.Fatalf("Kind() = %q, want %q", got, "rsync")
	}
}

func TestRsyncDeployer_Validate_Happy(t *testing.T) {
	d := NewRsyncDeployer()
	if err := d.Validate(validRsyncTarget()); err != nil {
		t.Fatalf("Validate(valid) returned error: %v", err)
	}
}

func TestRsyncDeployer_Validate_MissingFields(t *testing.T) {
	tests := []struct {
		name    string
		mutate  func(map[string]any)
		errFrag string
	}{
		{"missing host", func(c map[string]any) { delete(c, "host") }, "host is required"},
		{"missing user", func(c map[string]any) { delete(c, "user") }, "user is required"},
		{"missing path", func(c map[string]any) { delete(c, "path") }, "path is required"},
		{"missing key", func(c map[string]any) { delete(c, "private_key_pem") }, "private_key_pem is required"},
		{"empty host", func(c map[string]any) { c["host"] = "" }, "host is required"},
		{"empty user", func(c map[string]any) { c["user"] = "" }, "user is required"},
		{"empty path", func(c map[string]any) { c["path"] = "" }, "path is required"},
		{"empty key", func(c map[string]any) { c["private_key_pem"] = "" }, "private_key_pem is required"},
	}

	d := NewRsyncDeployer()
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			tgt := validRsyncTarget()
			tc.mutate(tgt.Config)
			err := d.Validate(tgt)
			if err == nil {
				t.Fatalf("Validate succeeded, want error containing %q", tc.errFrag)
			}
			if !strings.Contains(err.Error(), tc.errFrag) {
				t.Fatalf("Validate error = %q, want substring %q", err.Error(), tc.errFrag)
			}
		})
	}
}

func TestRsyncDeployer_Validate_RejectsMalformedPEM(t *testing.T) {
	d := NewRsyncDeployer()
	tgt := validRsyncTarget()
	tgt.Config["private_key_pem"] = "not a pem block at all"
	err := d.Validate(tgt)
	if err == nil {
		t.Fatal("expected error for malformed PEM, got nil")
	}
	if !strings.Contains(err.Error(), "PEM") {
		t.Fatalf("error = %q, want PEM mention", err.Error())
	}
}

func TestRsyncDeployer_Validate_RejectsBadHost(t *testing.T) {
	d := NewRsyncDeployer()
	tgt := validRsyncTarget()
	tgt.Config["host"] = "not a host name with spaces"
	err := d.Validate(tgt)
	if err == nil {
		t.Fatal("expected error for malformed host, got nil")
	}
	if !strings.Contains(err.Error(), "hostname") {
		t.Fatalf("error = %q, want hostname mention", err.Error())
	}
}

func TestRsyncDeployer_Validate_PortRange(t *testing.T) {
	d := NewRsyncDeployer()
	for _, port := range []float64{0, -1, 65536, 100000} {
		tgt := validRsyncTarget()
		tgt.Config["port"] = port
		err := d.Validate(tgt)
		if err == nil {
			t.Fatalf("port %v: expected error, got nil", port)
		}
		if !strings.Contains(err.Error(), "out of range") {
			t.Fatalf("port %v: error = %q, want out-of-range mention", port, err.Error())
		}
	}
}

func TestRsyncDeployer_Validate_PortDefaultIs22(t *testing.T) {
	tgt := validRsyncTarget()
	delete(tgt.Config, "port")
	cfg, err := parseRsyncConfig(tgt.Config)
	if err != nil {
		t.Fatalf("parseRsyncConfig: %v", err)
	}
	if cfg.Port != 22 {
		t.Fatalf("default port = %d, want 22", cfg.Port)
	}
}

func TestBuildRsyncArgs(t *testing.T) {
	cfg := rsyncConfig{
		Host: "deploy.example.com",
		User: "deployer",
		Port: 2222,
		Path: "/var/www/example",
	}
	args := buildRsyncArgs(cfg, "/tmp/dist", "/tmp/keyfile")

	if len(args) < 5 {
		t.Fatalf("buildRsyncArgs returned %d args, want >= 5: %v", len(args), args)
	}
	if args[0] != "-avz" {
		t.Errorf("args[0] = %q, want -avz", args[0])
	}
	if args[1] != "--delete" {
		t.Errorf("args[1] = %q, want --delete", args[1])
	}
	if args[2] != "-e" {
		t.Errorf("args[2] = %q, want -e", args[2])
	}

	sshCmd := args[3]
	for _, want := range []string{
		"ssh -i /tmp/keyfile",
		"-p 2222",
		"StrictHostKeyChecking=no",
		"UserKnownHostsFile=/dev/null",
	} {
		if !strings.Contains(sshCmd, want) {
			t.Errorf("ssh cmd %q missing %q", sshCmd, want)
		}
	}

	src := args[4]
	if src != "/tmp/dist/" {
		t.Errorf("src = %q, want trailing-slash form /tmp/dist/", src)
	}

	dst := args[5]
	if dst != "deployer@deploy.example.com:/var/www/example/" {
		t.Errorf("dst = %q, want deployer@deploy.example.com:/var/www/example/", dst)
	}
}

func TestBuildRsyncArgs_StripsTrailingSlashes(t *testing.T) {
	cfg := rsyncConfig{
		Host: "h", User: "u", Port: 22, Path: "/srv/site/",
	}
	args := buildRsyncArgs(cfg, "/tmp/dist////", "/k")
	if args[4] != "/tmp/dist/" {
		t.Errorf("src = %q, want /tmp/dist/", args[4])
	}
	if args[5] != "u@h:/srv/site/" {
		t.Errorf("dst = %q, want u@h:/srv/site/", args[5])
	}
}

func TestWalkDist(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "a.txt"), []byte("hello"), 0o644); err != nil {
		t.Fatal(err)
	}
	sub := filepath.Join(dir, "sub")
	if err := os.MkdirAll(sub, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(sub, "b.bin"), []byte("worldworld"), 0o644); err != nil {
		t.Fatal(err)
	}

	size, count, err := walkDist(dir)
	if err != nil {
		t.Fatalf("walkDist: %v", err)
	}
	if count != 2 {
		t.Errorf("count = %d, want 2", count)
	}
	if size != int64(len("hello")+len("worldworld")) {
		t.Errorf("size = %d, want %d", size, len("hello")+len("worldworld"))
	}
}

func TestWriteKeyfile_PermsAndCleanup(t *testing.T) {
	path, cleanup, err := writeKeyfile(fakePEM)
	if err != nil {
		t.Fatalf("writeKeyfile: %v", err)
	}

	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("stat keyfile: %v", err)
	}
	if perm := info.Mode().Perm(); perm != 0o600 {
		t.Errorf("keyfile perm = %o, want 0600", perm)
	}

	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read keyfile: %v", err)
	}
	if string(got) != fakePEM {
		t.Errorf("keyfile contents differ from input")
	}

	cleanup()
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Errorf("expected keyfile to be removed, stat err = %v", err)
	}
}

func TestRsyncDeployer_Deploy_NetworkSkipUnlessOptIn(t *testing.T) {
	if os.Getenv("RUN_NETWORK_TESTS") != "1" {
		t.Skip("set RUN_NETWORK_TESTS=1 to exercise the real rsync invocation")
	}

	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "index.html"), []byte("<html></html>"), 0o644); err != nil {
		t.Fatal(err)
	}

	d := NewRsyncDeployer()
	tgt := validRsyncTarget()
	if _, err := d.Deploy(context.Background(), dir, tgt); err == nil {
		t.Log("network deploy succeeded (or fake target was somehow reachable)")
	}
}

func TestRsyncDeployer_Deploy_ValidatesBeforeRunning(t *testing.T) {
	d := NewRsyncDeployer()
	tgt := validRsyncTarget()
	tgt.Config["host"] = ""
	_, err := d.Deploy(context.Background(), t.TempDir(), tgt)
	if err == nil {
		t.Fatal("expected validation error before exec, got nil")
	}
	if !strings.Contains(err.Error(), "host is required") {
		t.Fatalf("error = %q, want host-required message", err.Error())
	}
}

func TestRsyncDeployer_Deploy_RejectsMissingDistDir(t *testing.T) {
	d := NewRsyncDeployer()
	tgt := validRsyncTarget()
	_, err := d.Deploy(context.Background(), "/nonexistent/path/atomicsite-test", tgt)
	if err == nil {
		t.Fatal("expected error for missing dist dir, got nil")
	}
	if !strings.Contains(err.Error(), "dist dir") {
		t.Fatalf("error = %q, want dist-dir mention", err.Error())
	}
}

// Compile-time assertion that RsyncDeployer satisfies the Deployer interface.
// If P8-A's deploy.go is not yet on disk the package will not compile, which
// surfaces the missing dependency rather than letting it slip through.
var _ Deployer = (*RsyncDeployer)(nil)
