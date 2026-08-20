package invapi

import (
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"strings"
)

var (
	secretJSONRe = regexp.MustCompile(`(?i)("(?:token|key|password|root_pass|passphrase|secret)"\s*:\s*")[^"]*(")`)
	secretFormRe = regexp.MustCompile(`(?i)((?:^|&)(?:token|key|password|root_pass)=)[^&]*`)
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
		return fmt.Sprintf("invapi error: %s", redactSecrets(e.Body))
	}
}

// IsNotFound reports whether err means the InvAPI object is gone (safe to drop
// Terraform state). Matches ErrNotFound and common InvAPI business envelopes.
func IsNotFound(err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, ErrNotFound) {
		return true
	}
	var api *APIError
	if errors.As(err, &api) {
		blob := strings.ToLower(strings.TrimSpace(api.Message + " " + api.Result + " " + api.Body))
		return strings.Contains(blob, "not found") ||
			strings.Contains(blob, "no such") ||
			strings.Contains(blob, "unknown server") ||
			strings.Contains(blob, "server not exist") ||
			strings.Contains(blob, "does not exist")
	}
	return false
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
		return &APIError{Message: msg, Body: redactSecrets(string(body)), Code: codeAsInt(envelope.Code)}
	}
	if msg != "" {
		return &APIError{Message: msg, Body: redactSecrets(string(body))}
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
	s := redactSecrets(string(b))
	if len(s) <= n {
		return s
	}
	return s[:n] + "..."
}

func redactSecrets(s string) string {
	s = secretJSONRe.ReplaceAllString(s, `${1}***${2}`)
	s = secretFormRe.ReplaceAllString(s, `${1}***`)
	return s
}
