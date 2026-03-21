package trade

import (
	"encoding/json"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestTradeBadJSONUsesErrorEnvelope(t *testing.T) {
	h := NewHandler(nil)
	req := httptest.NewRequest("POST", "/markets/123/trade", strings.NewReader("{"))
	res := httptest.NewRecorder()

	h.Trade(res, req)

	var body map[string]map[string]string
	require.NoError(t, json.Unmarshal(res.Body.Bytes(), &body))
	require.Equal(t, 401, res.Code)
	require.Equal(t, "unauthorized", body["error"]["code"])
}
