package user

import (
	"encoding/json"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestRegisterBadJSONUsesErrorEnvelope(t *testing.T) {
	h := NewHandler(nil, "secret")
	req := httptest.NewRequest("POST", "/auth/register", strings.NewReader("{"))
	res := httptest.NewRecorder()

	h.Register(res, req)

	var body map[string]map[string]string
	require.NoError(t, json.Unmarshal(res.Body.Bytes(), &body))
	require.Equal(t, 400, res.Code)
	require.Equal(t, "bad_request", body["error"]["code"])
}

func TestListRejectsInvalidLimit(t *testing.T) {
	h := NewHandler(nil, "secret")
	req := httptest.NewRequest("GET", "/users?limit=oops", nil)
	res := httptest.NewRecorder()

	h.List(res, req)

	var body map[string]map[string]string
	require.NoError(t, json.Unmarshal(res.Body.Bytes(), &body))
	require.Equal(t, 400, res.Code)
	require.Equal(t, "bad_request", body["error"]["code"])
}
