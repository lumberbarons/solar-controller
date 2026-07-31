package testutil

import (
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"testing"

	"github.com/stretchr/testify/require"
)

// apiContract mirrors docs/api-contract.json. The comment block is ignored.
type apiContract struct {
	Endpoints map[string][]string `json:"endpoints"`
}

// APIContractFields returns the JSON field names docs/api-contract.json records
// for an endpoint, keyed as it is in that file (for example
// "GET /api/epever/metrics").
//
// The contract is shared with the frontend: site/src/api/types.test.ts asserts
// the TypeScript payload types declare the same fields. Asserting handler output
// against it from here means renaming a field in a Go response fails CI instead
// of silently rendering as undefined in the browser.
func APIContractFields(t *testing.T, endpoint string) []string {
	t.Helper()

	contract := loadAPIContract(t)
	fields, ok := contract.Endpoints[endpoint]
	require.Truef(t, ok, "docs/api-contract.json has no entry for %q", endpoint)
	return fields
}

func loadAPIContract(t *testing.T) apiContract {
	t.Helper()

	path := filepath.Join(repoRoot(t), "docs", "api-contract.json")
	data, err := os.ReadFile(path) //nolint:gosec // path is derived from this file's own location, not input
	require.NoError(t, err, "failed to read %s", path)

	var contract apiContract
	require.NoError(t, json.Unmarshal(data, &contract), "failed to parse %s", path)
	require.NotEmpty(t, contract.Endpoints, "%s declares no endpoints", path)
	return contract
}

// repoRoot locates the module root from this source file's own path, so callers
// work regardless of which package directory the test runs in.
func repoRoot(t *testing.T) string {
	t.Helper()

	_, file, _, ok := runtime.Caller(0)
	require.True(t, ok, "failed to determine caller path")
	// This file lives at <root>/internal/testutil/apicontract.go
	return filepath.Dir(filepath.Dir(filepath.Dir(file)))
}

// JSONFieldNames returns the sorted top-level field names of a JSON object,
// for comparison against a contract field list.
func JSONFieldNames(t *testing.T, body []byte) []string {
	t.Helper()

	var fields map[string]json.RawMessage
	require.NoError(t, json.Unmarshal(body, &fields), "response is not a JSON object: %s", body)

	names := make([]string, 0, len(fields))
	for name := range fields {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}
