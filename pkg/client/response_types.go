package client

import (
	"encoding/json"
	"fmt"
)

// MultiResponse wraps responses that can have different schemas based on status code
type MultiResponse struct {
	Response
	parsedBody interface{}
}

// As attempts to unmarshal the response into the provided type
func (r *MultiResponse) As(v interface{}) error {
	if r.parsedBody != nil {
		return r.tryTypeAssertion(v)
	}
	return ParseJSON(&r.Response, v)
}

// Is checks if the response status code matches
func (r *MultiResponse) Is(statusCode int) bool {
	return r.StatusCode == statusCode
}

// IsSuccess returns true if the status code is in the 2xx range
func (r *MultiResponse) IsSuccess() bool {
	return r.StatusCode >= 200 && r.StatusCode < 300
}

// IsError returns true if the status code is 4xx or 5xx
func (r *MultiResponse) IsError() bool {
	return r.StatusCode >= 400
}

// tryTypeAssertion attempts to use cached parsed body
func (r *MultiResponse) tryTypeAssertion(v interface{}) error {
	switch target := v.(type) {
	case **json.RawMessage:
		if raw, ok := r.parsedBody.(*json.RawMessage); ok {
			*target = raw
			return nil
		}
	default:
		// For other types, re-parse from body
		return ParseJSON(&r.Response, v)
	}
	return fmt.Errorf("type assertion failed")
}
