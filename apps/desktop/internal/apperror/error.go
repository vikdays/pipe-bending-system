// Package apperror defines stable, public errors independent of the Wails transport.
package apperror

import (
	"encoding/json"
	"errors"
)

type Code string

const (
	ValidationError    Code = "VALIDATION_ERROR"
	InvalidCredentials Code = "INVALID_CREDENTIALS"
	Unauthenticated    Code = "UNAUTHENTICATED"
	Forbidden          Code = "FORBIDDEN"
	NotFound           Code = "NOT_FOUND"
	Conflict           Code = "CONFLICT"
	RateLimited        Code = "RATE_LIMITED"
	InternalError      Code = "INTERNAL_ERROR"
)

// Payload is placed in RuntimeError.cause by the Wails error marshaler.
type Payload struct {
	Code    Code   `json:"code"`
	Message string `json:"message"`
}

type Error struct{ payload Payload }

func New(code Code, message string) *Error {
	if !validCode(code) || message == "" {
		return internal
	}
	return &Error{payload: Payload{Code: code, Message: message}}
}

var internal = &Error{payload: Payload{
	Code:    InternalError,
	Message: "Внутренняя ошибка приложения. Повторите попытку позже.",
}}

func validCode(code Code) bool {
	switch code {
	case ValidationError, InvalidCredentials, Unauthenticated, Forbidden,
		NotFound, Conflict, RateLimited, InternalError:
		return true
	default:
		return false
	}
}

func (e *Error) Error() string {
	if e == nil {
		return internal.payload.Message
	}
	return e.payload.Message
}

// Public strips internal wrapping text as well as unknown errors before Wails
// builds its outer error.message. Never send raw database errors to the WebView.
func Public(err error) error {
	if err == nil {
		return nil
	}
	var known *Error
	if errors.As(err, &known) && known != nil && validCode(known.payload.Code) && known.payload.Message != "" {
		return known
	}
	return internal
}

func Describe(err error) Payload {
	if known, ok := Public(err).(*Error); ok {
		return known.payload
	}
	return internal.payload
}

// Marshal always returns valid JSON; it is suitable for ServiceOptions.MarshalError.
func Marshal(err error) []byte {
	if err == nil {
		return []byte("null")
	}
	data, _ := json.Marshal(Describe(err)) // Payload contains only JSON-safe primitive values.
	return data
}
