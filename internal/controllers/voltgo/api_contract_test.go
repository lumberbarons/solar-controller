package voltgo

import (
	"encoding/json"
	"testing"

	"github.com/lumberbarons/solar-controller/internal/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// MetricsGet and InfoGet serialise the collector structs directly, so their
// json tags are the wire contract. These tests pin those tags against
// docs/api-contract.json, which site/src/api/types.ts is checked against from
// the other side, so a rename cannot reach the browser as `undefined`.
func TestStatusPayloadMatchesAPIContract(t *testing.T) {
	payload, err := json.Marshal(BatteryStatus{})
	require.NoError(t, err)

	assert.Equal(t,
		testutil.APIContractFields(t, "GET /api/voltgo/metrics"),
		testutil.JSONFieldNames(t, payload),
		"BatteryStatus no longer marshals to the fields docs/api-contract.json "+
			"records; update the contract and site/src/api/types.ts together")
}

// JSONFieldNames only sees top-level keys, so the per-cell objects the battery
// panel reads need checking on their own.
func TestCellPayloadMatchesAPIContract(t *testing.T) {
	payload, err := json.Marshal(Cell{})
	require.NoError(t, err)

	assert.Equal(t,
		testutil.APIContractFields(t, "GET /api/voltgo/metrics#cells[]"),
		testutil.JSONFieldNames(t, payload),
		"Cell no longer marshals to the fields docs/api-contract.json records; "+
			"update the contract and site/src/api/types.ts together")
}

func TestInfoPayloadMatchesAPIContract(t *testing.T) {
	payload, err := json.Marshal(BatteryInfo{})
	require.NoError(t, err)

	assert.Equal(t,
		testutil.APIContractFields(t, "GET /api/voltgo/info"),
		testutil.JSONFieldNames(t, payload),
		"BatteryInfo no longer marshals to the fields docs/api-contract.json "+
			"records; update the contract and site/src/api/types.ts together")
}
