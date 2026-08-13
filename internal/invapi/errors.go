package invapi

import (
	"encoding/json"
	"fmt"
)

// APIError is returned when InvAPI responds with a business error envelope.
type APIError struct {
	Code    int
	Message string
	Result  string
	Body    string
}

func (e *APIError) Error() string {
	switch {
	case e.Message != "":
		return fmt.Sprintf("invapi error (code=%d): %s", e.Code, e.Message)
	case e.Result != "" && e.Result != "OK":
		return fmt.Sprintf("invapi error: result=%s", e.Result)
	default:
		return fmt.Sprintf("invapi error: %s", e.Body)
	}
}

func decodeAPIError(body []byte) error {
	var envelope struct {
		Code    json.RawMessage `json:"code"`
		Message string          `json:"message"`
		Error   string          `json:"error"`
		Result  json.RawMessage `json:"result"`
	}
	if err := json.Unmarshal(body, &envelope); err != nil {
		return fmt.Errorf("invapi: invalid JSON response: %w; body: %s", err, truncate(body, 512))
	}

	if resultErr := resultFieldError(envelope.Result); resultErr != nil {
		if apiErr, ok := resultErr.(*APIError); ok && apiErr.Message == "" && envelope.Error != "" {
			apiErr.Message = envelope.Error
		}
		return resultErr
	}

	msg := envelope.Message
	if msg == "" {
		msg = envelope.Error
	}
	if len(envelope.Code) > 0 && string(envelope.Code) != "0" && string(envelope.Code) != "null" {
		return &APIError{Message: msg, Body: string(body), Code: codeAsInt(envelope.Code)}
	}
	if msg != "" {
		return &APIError{Message: msg, Body: string(body)}
	}

	return nil
}

func resultFieldError(raw json.RawMessage) error {
	if len(raw) == 0 {
		return nil
	}
	var s string
	if err := json.Unmarshal(raw, &s); err == nil {
		if s == "OK" || s == "" {
			return nil
		}
		return &APIError{Result: s, Body: string(raw)}
	}
	var n float64
	if err := json.Unmarshal(raw, &n); err == nil {
		if int(n) == 0 || int(n) == 1 {
			return nil
		}
		return &APIError{Code: int(n), Body: string(raw)}
	}
	return nil
}

func codeAsInt(raw json.RawMessage) int {
	var n int
	if err := json.Unmarshal(raw, &n); err == nil {
		return n
	}
	return -1
}

func truncate(b []byte, n int) string {
	if len(b) <= n {
		return string(b)
	}
	return string(b[:n]) + "..."
}
