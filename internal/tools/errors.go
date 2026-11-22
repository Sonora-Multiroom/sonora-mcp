package tools

import (
	"context"
	"errors"
	"fmt"
	"net"
	"strings"

	"github.com/Sonora-Multiroom/sonora-cli/hub"
)

// Error categories. Every failed tool call returns text starting with one of
// these followed by ": " (research R5).
const (
	CategoryNotFound           = "NotFound"
	CategoryValidation         = "Validation"
	CategoryRouteFailed        = "RouteFailed"
	CategoryConflict           = "Conflict"
	CategorySourceUnreachable  = "SourceUnreachable"
	CategoryServiceUnavailable = "ServiceUnavailable"
	CategoryTtsUnavailable     = "TtsUnavailable"
	CategoryRadioUnavailable   = "RadioUnavailable"
	CategoryNetwork            = "Network"
	CategoryTimeout            = "Timeout"
	CategoryMalformedResponse  = "MalformedResponse"
	CategoryHubError           = "HubError"
	CategoryInternal           = "Internal"
)

// radioProblemPrefix starts the type of every problem the hub's radio
// extension reports.
const radioProblemPrefix = "urn:multiroom:radio:error:"

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
	// A diagnosed "not offered" answer comes first: its cause may be the
	// timeout or decode failure of the inventory lookup, which must not hide
	// that text-to-speech is unavailable.
	var unavailErr *hub.TTSUnavailableError
	if errors.As(err, &unavailErr) {
		if unavailErr.Diagnosis == hub.DiagnosisHubAddress {
			return CategoryNetwork, unavailErr.Error()
		}
		return CategoryTtsUnavailable, unavailErr.Error()
	}
	var radioErr *hub.RadioUnavailableError
	if errors.As(err, &radioErr) {
		if radioErr.Diagnosis == hub.DiagnosisHubAddress {
			return CategoryNetwork, radioErr.Error()
		}
		return CategoryRadioUnavailable, radioErr.Error()
	}
	var radioNotOffered *hub.RadioNotOfferedError
	if errors.As(err, &radioNotOffered) {
		return CategoryRadioUnavailable, "radio is not available on this hub"
	}
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
	// A radio problem's message is already built from its detail.
	if errors.As(err, &apiErr) && apiErr.Detail != "" && message != apiErr.Detail &&
		!strings.HasPrefix(apiErr.Type, radioProblemPrefix) {
		message += ": " + apiErr.Detail
	}
	// Name why the hub refused a route, so the agent knows which state to
	// change; an unknown reason is passed through as is.
	if apiErr != nil && apiErr.Reason != "" {
		message += " (" + apiErr.Reason + ")"
	}

	// Name the input the hub refused, so the agent knows which to correct.
	if apiErr != nil && apiErr.Parameter != "" {
		message += " (parameter " + apiErr.Parameter + ")"
	}

	switch class {
	case hub.ClassNotFound, hub.ClassInputNotFound, hub.ClassTargetNotFound:
		return CategoryNotFound, message
	case hub.ClassValidation:
		return CategoryValidation, message
	case hub.ClassRouteFailed:
		return CategoryRouteFailed, message
	case hub.ClassConflict:
		return CategoryConflict, message
	case hub.ClassSourceUnreachable:
		return CategorySourceUnreachable, message
	case hub.ClassServiceUnavailable:
		return CategoryServiceUnavailable, message
	case hub.ClassTTSUnavailable:
		return CategoryTtsUnavailable, message
	case hub.ClassRadioUnavailable:
		return CategoryRadioUnavailable, message
	case hub.ClassNetwork:
		return CategoryNetwork, fmt.Sprintf("%s: %v", message, err)
	default:
		return CategoryHubError, message
	}
}
