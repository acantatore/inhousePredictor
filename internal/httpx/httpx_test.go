package httpx

import (
	"encoding/json"
	"errors"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestWriteJSONUsesDataEnvelope(t *testing.T) {
	res := httptest.NewRecorder()
	WriteJSON(res, 201, map[string]string{"status": "ok"})

	var body map[string]map[string]string
	require.NoError(t, json.Unmarshal(res.Body.Bytes(), &body))
	require.Equal(t, 201, res.Code)
	require.Equal(t, "ok", body["data"]["status"])
}

func TestWriteErrorUsesErrorEnvelope(t *testing.T) {
	res := httptest.NewRecorder()
	WriteError(res, NewError(409, CodeConflict, "Conflict.", errors.New("boom")))

	var body map[string]map[string]string
	require.NoError(t, json.Unmarshal(res.Body.Bytes(), &body))
	require.Equal(t, 409, res.Code)
	require.Equal(t, string(CodeConflict), body["error"]["code"])
	require.Equal(t, "Conflict.", body["error"]["message"])
}
