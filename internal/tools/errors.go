package tools

import (
	"context"
	"errors"
	"fmt"
	"net"

	"github.com/Sonora-Multiroom/sonora-cli/hub"
)

// Error categories. Every failed tool call returns text starting with one of
// these followed by ": " (research R5).
const (
	CategoryNotFound           = "NotFound"
	CategoryValidation         = "Validation"
	CategoryRouteFailed        = "RouteFailed"
	CategorySourceUnreachable  = "SourceUnreachable"
	CategoryServiceUnavailable = "ServiceUnavailable"
	CategoryNetwork            = "Network"
	CategoryTimeout            = "Timeout"
	CategoryMalformedResponse  = "MalformedResponse"
	CategoryHubError           = "HubError"
	CategoryInternal           = "Internal"
)

// categoryError is a tool failure: a category and a message for the agent,
// wrapping the underlying error.
type categoryError struct {
	category string
	message  string
	err      error
}

func (e *categoryError) Error() string { return e.category + ": " + e.message }

func (e *categoryError) Unwrap() error { return e.err }

// toolError turns an error from a hub call into a categorized tool error
// whose text is "<Category>: <message>".
func toolError(err error) error {
	var ce *categoryError
	if errors.As(err, &ce) {
		return err
	}
	category, message := classify(err)
	return &categoryError{category: category, message: message, err: err}
}

func classify(err error) (category, message string) {
	var decodeErr *hub.DecodeError
	if errors.As(err, &decodeErr) {
		return CategoryMalformedResponse, fmt.Sprintf("the hub's response could not be read: %v", decodeErr.Err)
	}
	var netErr net.Error
	if errors.Is(err, context.DeadlineExceeded) || (errors.As(err, &netErr) && netErr.Timeout()) {
		return CategoryTimeout, "the hub did not respond in time"
	}

	class, message := hub.ClassifyError(err)

	// Keep the hub's own explanation where ClassifyError replaces it with a
	// generic message (FR-010).
	var apiErr *hub.APIError
	if errors.As(err, &apiErr) && apiErr.Detail != "" && message != apiErr.Detail {
		message += ": " + apiErr.Detail
	}

	switch class {
	case hub.ClassNotFound, hub.ClassInputNotFound, hub.ClassTargetNotFound:
		return CategoryNotFound, message
	case hub.ClassValidation:
		return CategoryValidation, message
	case hub.ClassRouteFailed:
		return CategoryRouteFailed, message
	case hub.ClassSourceUnreachable:
		return CategorySourceUnreachable, message
	case hub.ClassServiceUnavailable:
		return CategoryServiceUnavailable, message
	case hub.ClassNetwork:
		return CategoryNetwork, fmt.Sprintf("%s: %v", message, err)
	default:
		return CategoryHubError, message
	}
}
