package clients

import (
	"encoding/json"
	"fmt"
	"net/http"
)

// fetchJSON executes req using doer, ensures the response body is closed,
// asserts that the HTTP status is 200 OK, and decodes the JSON body into v.
// All errors are prefixed with prefix so callers can be identified in logs
// (e.g. prefix "countries" produces "countries: unexpected status 404").
// The context for cancellation/timeout must already be set on req via
// http.NewRequestWithContext before calling this function.
func fetchJSON(doer HTTPDoer, req *http.Request, v any, prefix string) error {
	resp, err := doer.Do(req)
	if err != nil {
		return fmt.Errorf("%s: request failed: %w", prefix, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("%s: unexpected status %d", prefix, resp.StatusCode)
	}

	if err := json.NewDecoder(resp.Body).Decode(v); err != nil {
		return fmt.Errorf("%s: decode response: %w", prefix, err)
	}

	return nil
}
