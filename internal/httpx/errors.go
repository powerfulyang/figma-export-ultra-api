package httpx

import (
    "errors"
    "net/http"

    "github.com/gofiber/fiber/v2"
)

// APIError is a structured application error with code and message.
type APIError struct {
	HTTPStatus int         `json:"-"`
	Code       string      `json:"code"`
	Message    string      `json:"message"`
	Details    interface{} `json:"details,omitempty"`
}

func (e *APIError) Error() string { return e.Message }

func NewAPIError(httpStatus int, code, msg string, details interface{}) *APIError {
	return &APIError{HTTPStatus: httpStatus, Code: code, Message: msg, Details: details}
}

// Common helpers
func BadRequest(msg string, details interface{}) error {
	return NewAPIError(http.StatusBadRequest, "E_INVALID_PARAM", msg, details)
}
func NotFound(msg string) error { return NewAPIError(http.StatusNotFound, "E_NOT_FOUND", msg, nil) }
func InternalError(msg string, details interface{}) error {
	return NewAPIError(http.StatusInternalServerError, "E_INTERNAL", msg, details)
}

// ErrorHandler returns a Fiber error handler that emits unified error responses.
func ErrorHandler() fiber.ErrorHandler {
	return func(c *fiber.Ctx, err error) error {
		// Fiber error
		var fe *fiber.Error
		if errors.As(err, &fe) {
            return c.Status(fe.Code).JSON(fiber.Map{
                "code":       httpStatusToCode(fe.Code),
                "message":    fe.Message,
                "request_id": requestID(c),
            })
		}

		// Application error
		var ae *APIError
		if errors.As(err, &ae) {
            return c.Status(ae.HTTPStatus).JSON(fiber.Map{
                "code":       ae.Code,
                "message":    ae.Message,
                "details":    ae.Details,
                "request_id": requestID(c),
            })
		}

		// Fallback
        return c.Status(http.StatusInternalServerError).JSON(fiber.Map{
            "code":       "E_INTERNAL",
            "message":    "Internal Server Error",
            "request_id": requestID(c),
        })
	}
}

func httpStatusToCode(status int) string {
	switch status {
	case http.StatusBadRequest:
		return "E_INVALID_PARAM"
	case http.StatusNotFound:
		return "E_NOT_FOUND"
	case http.StatusUnauthorized:
		return "E_UNAUTHORIZED"
	case http.StatusForbidden:
		return "E_FORBIDDEN"
	default:
		if status >= 500 {
			return "E_INTERNAL"
		}
		return "E_UNKNOWN"
	}
}
