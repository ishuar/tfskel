package ai

import "net/http"

// clientOptions collects transport-level overrides shared by both provider
// constructors. Zero values mean "use the SDK's production defaults".
type clientOptions struct {
	baseURL    string
	httpClient *http.Client
}

// Option configures transport-level behavior of a provider client. The
// options exist as a seam: production code never passes any, tests point the
// client at an httptest.Server to exercise the full streaming path (Explain,
// truncation detection, mid-stream failures) without the network.
type Option func(*clientOptions)

// WithBaseURL overrides the provider API endpoint.
func WithBaseURL(url string) Option {
	return func(o *clientOptions) { o.baseURL = url }
}

// WithHTTPClient overrides the HTTP client used for API requests.
func WithHTTPClient(c *http.Client) Option {
	return func(o *clientOptions) { o.httpClient = c }
}

// applyOptions folds opts into a clientOptions value.
func applyOptions(opts []Option) clientOptions {
	var o clientOptions
	for _, opt := range opts {
		opt(&o)
	}
	return o
}
