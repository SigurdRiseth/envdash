package clients

import "net/http"

// HTTPDoer is the minimal interface required by all API clients.
// In production, *http.Client satisfies this. In tests, a stub can be injected.
type HTTPDoer interface {
	Do(req *http.Request) (*http.Response, error)
}
