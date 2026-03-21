package httpx

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"

	"github.com/jackc/pgx/v5/pgconn"
)

type ErrorCode string

const (
	CodeBadRequest          ErrorCode = "bad_request"
	CodeUnauthorized        ErrorCode = "unauthorized"
	CodeForbidden           ErrorCode = "forbidden"
	CodeNotFound            ErrorCode = "not_found"
	CodeConflict            ErrorCode = "conflict"
	CodeValidation          ErrorCode = "validation_error"
	CodeInsufficientBalance ErrorCode = "insufficient_balance"
	CodeMarketClosed        ErrorCode = "market_closed"
	CodeMarketBusy          ErrorCode = "market_busy"
	CodeCreatorRestricted   ErrorCode = "creator_cannot_trade_own_market"
	CodeDisputeClosed       ErrorCode = "dispute_window_closed"
	CodeInvalidState        ErrorCode = "invalid_market_state"
	CodeInternal            ErrorCode = "internal_error"
)

type Error struct {
	Status  int
	Code    ErrorCode
	Message string
	Err     error
}

func (e *Error) Error() string {
	if e == nil {
		return ""
	}
	if e.Message != "" {
		return e.Message
	}
	if e.Err != nil {
		return e.Err.Error()
	}
	return string(e.Code)
}

func (e *Error) Unwrap() error { return e.Err }

func NewError(status int, code ErrorCode, message string, err error) *Error {
	return &Error{Status: status, Code: code, Message: message, Err: err}
}

func WriteJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]any{"data": data})
}

func WriteError(w http.ResponseWriter, err error) {
	var appErr *Error
	if !errors.As(err, &appErr) {
		appErr = NewError(http.StatusInternalServerError, CodeInternal, "Internal server error.", err)
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(appErr.Status)
	_ = json.NewEncoder(w).Encode(map[string]any{
		"error": map[string]any{
			"code":    appErr.Code,
			"message": appErr.Message,
		},
	})
}

func IsSerialization(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "40001"
}

func WrapInternal(op string, err error) error {
	if err == nil {
		return nil
	}
	return NewError(http.StatusInternalServerError, CodeInternal, "Internal server error.", fmt.Errorf("%s: %w", op, err))
}
