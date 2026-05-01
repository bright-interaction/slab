//go:build !ee

package ee

// IsAvailable reports whether the binary was built with the cloud control
// plane linked in. Always false in the OSS build.
func IsAvailable() bool { return false }

// NewProvider returns the no-op Provider used by the OSS build. Every
// cloud-shaped method returns ErrNotAvailable; core code that depends on
// real cloud behaviour should check IsAvailable() at startup or
// errors.Is(err, ErrNotAvailable) at the call site.
func NewProvider() Provider { return ossProvider{} }

type ossProvider struct{}

func (ossProvider) Mode() string { return "oss" }
