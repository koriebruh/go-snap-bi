package snap

// Profile provides the integration-specific hooks the SNAP standard leaves
// open to each PJP: timestamp formatting and URI path construction. A
// bank-specific deviation is expressed as a small struct embedding
// DefaultProfile and overriding one method, not a fork of HeaderBuilder.
type Profile interface {
	// TimestampLayout returns the Go time layout used to format X-TIMESTAMP
	// and the timestamp segment of the signed string.
	TimestampLayout() string

	// BuildPath returns the request URI path for a given service group and
	// product type, per spec §2.1.11.
	BuildPath(serviceGroup, productType string) string
}

// DefaultProfile implements the standard SNAP behavior exactly. Domain and
// Version are integration-specific, so callers must construct one, e.g.
// DefaultProfile{Domain: "openapi.example.com", Version: "v1.0"}.
type DefaultProfile struct {
	Domain  string
	Version string
}

// TimestampLayout returns DefaultTimestampLayout.
func (p DefaultProfile) TimestampLayout() string {
	return DefaultTimestampLayout
}

// BuildPath returns "/{domain}/{version}/{service-group}/{product-type}".
// Version defaults to "v1.0" when unset.
func (p DefaultProfile) BuildPath(serviceGroup, productType string) string {
	version := p.Version
	if version == "" {
		version = "v1.0"
	}
	return "/" + p.Domain + "/" + version + "/" + serviceGroup + "/" + productType
}
