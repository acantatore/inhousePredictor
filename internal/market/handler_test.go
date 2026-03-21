package market

import (
	"encoding/json"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestCreateInvalidResolverUsesErrorEnvelope(t *testing.T) {
	h := NewHandler(nil)
	req := httptest.NewRequest("POST", "/markets", strings.NewReader(`{"resolver_id":"not-a-uuid"}`))
	res := httptest.NewRecorder()

	h.Create(res, req)

	var body map[string]map[string]string
	require.NoError(t, json.Unmarshal(res.Body.Bytes(), &body))
	require.Equal(t, 401, res.Code)
	require.Equal(t, "unauthorized", body["error"]["code"])
}
