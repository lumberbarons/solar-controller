package app

import (
	"encoding/json"
	"testing"

	"github.com/lumberbarons/solar-controller/internal/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// The /api/info handler serialises VersionInfo directly, so its json tags are
// the wire contract the frontend's VersionInfo type mirrors.
func TestVersionInfoMatchesAPIContract(t *testing.T) {
	payload, err := json.Marshal(VersionInfo{})
	require.NoError(t, err)

	assert.Equal(t,
		testutil.APIContractFields(t, "GET /api/info"),
		testutil.JSONFieldNames(t, payload),
		"VersionInfo no longer marshals to the fields docs/api-contract.json "+
			"records; update the contract and site/src/api/types.ts together")
}
