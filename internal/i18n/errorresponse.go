package i18n

import (
	"errors"

	"github.com/gin-gonic/gin"
)

// ErrorResponse represents an error response
type ErrorResponse struct {
	StatusCode ErrorCode
	Err        error
}

// WithHttpCode sets the HTTP status code for the error
func (r *ErrorResponse) WithHttpCode(code ErrorCode) *ErrorResponse {
	r.StatusCode = code
	return r
}

// WithParam adds a parameter to the error
func (r *ErrorResponse) WithParam(key string, value interface{}) *ErrorResponse {
	var i18nErr *ErrorWithCode
	if errors.As(r.Err, &i18nErr) {
		r.Err = i18nErr.WithParam(key, value)
	}
	return r
}

// WithHeader adds a header to the response
func (r *ErrorResponse) WithHeader(key, value string) *ErrorResponse {
	// Note: Headers will be implemented in the Send method
	return r
}

// Send sends the error response to the client
func (r *ErrorResponse) Send(c *gin.Context) {
	RespondWithError(c, r.Err)
}

// BadRequest creates a new error response with status code 400
func BadRequest(msgID string) *ErrorResponse {
	return &ErrorResponse{
		StatusCode: ErrorBadRequest,
		Err:        NewErrorWithCode(msgID, ErrorBadRequest),
	}
}

// Unauthorized creates a new error response with status code 401
func Unauthorized(msgID string) *ErrorResponse {
	return &ErrorResponse{
		StatusCode: ErrorUnauthorized,
		Err:        NewErrorWithCode(msgID, ErrorUnauthorized),
	}
}

// Forbidden creates a new error response with status code 403
func Forbidden(msgID string) *ErrorResponse {
	return &ErrorResponse{
		StatusCode: ErrorForbidden,
		Err:        NewErrorWithCode(msgID, ErrorForbidden),
	}
}

// NotFound creates a new error response with status code 404
func NotFound(msgID string) *ErrorResponse {
	return &ErrorResponse{
		StatusCode: ErrorNotFound,
		Err:        NewErrorWithCode(msgID, ErrorNotFound),
	}
}

// Conflict creates a new error response with status code 409
func Conflict(msgID string) *ErrorResponse {
	return &ErrorResponse{
		StatusCode: ErrorConflict,
		Err:        NewErrorWithCode(msgID, ErrorConflict),
	}
}

// InternalError creates a new error response with status code 500
func InternalError(msgID string) *ErrorResponse {
	return &ErrorResponse{
		StatusCode: ErrorInternalServer,
		Err:        NewErrorWithCode(msgID, ErrorInternalServer),
	}
}

// Error creates an error response from a predefined error constant
func Error(predefinedErr error) *ErrorResponse {
	statusCode := ErrorInternalServer
	var errWithCode *ErrorWithCode
	if errors.As(predefinedErr, &errWithCode) {
		statusCode = errWithCode.GetCode()
	}
	return &ErrorResponse{
		StatusCode: statusCode,
		Err:        predefinedErr,
	}
}

// From is an alias for Error for backward compatibility
func From(err error) *ErrorResponse {
	return Error(err)
}

// NotFoundFromErr creates a not found error from a predefined error
func NotFoundFromErr(predefinedErr error) *ErrorResponse {
	var errWithCode *ErrorWithCode
	if errors.As(predefinedErr, &errWithCode) {
		return &ErrorResponse{
			StatusCode: ErrorNotFound,
			Err:        errWithCode.WithHttpCode(ErrorNotFound),
		}
	}
	return &ErrorResponse{
		StatusCode: ErrorNotFound,
		Err:        NewErrorWithCode(predefinedErr.Error(), ErrorNotFound),
	}
}

// BadRequestFromErr creates a bad request error from a predefined error
func BadRequestFromErr(predefinedErr error) *ErrorResponse {
	var errWithCode *ErrorWithCode
	if errors.As(predefinedErr, &errWithCode) {
		return &ErrorResponse{
			StatusCode: ErrorBadRequest,
			Err:        errWithCode.WithHttpCode(ErrorBadRequest),
		}
	}
	return &ErrorResponse{
		StatusCode: ErrorBadRequest,
		Err:        NewErrorWithCode(predefinedErr.Error(), ErrorBadRequest),
	}
}

// UnauthorizedFromErr creates an unauthorized error from a predefined error
func UnauthorizedFromErr(predefinedErr error) *ErrorResponse {
	var errWithCode *ErrorWithCode
	if errors.As(predefinedErr, &errWithCode) {
		return &ErrorResponse{
			StatusCode: ErrorUnauthorized,
			Err:        errWithCode.WithHttpCode(ErrorUnauthorized),
		}
	}
	return &ErrorResponse{
		StatusCode: ErrorUnauthorized,
		Err:        NewErrorWithCode(predefinedErr.Error(), ErrorUnauthorized),
	}
}

// ForbiddenFromErr creates a forbidden error from a predefined error
func ForbiddenFromErr(predefinedErr error) *ErrorResponse {
	var errWithCode *ErrorWithCode
	if errors.As(predefinedErr, &errWithCode) {
		return &ErrorResponse{
			StatusCode: ErrorForbidden,
			Err:        errWithCode.WithHttpCode(ErrorForbidden),
		}
	}
	return &ErrorResponse{
		StatusCode: ErrorForbidden,
		Err:        NewErrorWithCode(predefinedErr.Error(), ErrorForbidden),
	}
}

// ConflictFromErr creates a conflict error from a predefined error
func ConflictFromErr(predefinedErr error) *ErrorResponse {
	var errWithCode *ErrorWithCode
	if errors.As(predefinedErr, &errWithCode) {
		return &ErrorResponse{
			StatusCode: ErrorConflict,
			Err:        errWithCode.WithHttpCode(ErrorConflict),
		}
	}
	return &ErrorResponse{
		StatusCode: ErrorConflict,
		Err:        NewErrorWithCode(predefinedErr.Error(), ErrorConflict),
	}
}

// InternalServerFromErr creates an internal server error from a predefined error
func InternalServerFromErr(predefinedErr error) *ErrorResponse {
	var errWithCode *ErrorWithCode
	if errors.As(predefinedErr, &errWithCode) {
		return &ErrorResponse{
			StatusCode: ErrorInternalServer,
			Err:        errWithCode.WithHttpCode(ErrorInternalServer),
		}
	}
	return &ErrorResponse{
		StatusCode: ErrorInternalServer,
		Err:        NewErrorWithCode(predefinedErr.Error(), ErrorInternalServer),
	}
}

// ErrorWithParam creates an error response with parameters
func ErrorWithParam(predefinedErr error, key string, value interface{}) *ErrorResponse {
	var errWithCode *ErrorWithCode
	if errors.As(predefinedErr, &errWithCode) {
		return &ErrorResponse{
			StatusCode: errWithCode.GetCode(),
			Err:        errWithCode.WithParam(key, value),
		}
	}

	// For other error types, create an internal server error
	return &ErrorResponse{
		StatusCode: ErrorInternalServer,
		Err:        NewErrorWithCode(predefinedErr.Error(), ErrorInternalServer).WithParam(key, value),
	}
}

// ErrorWithParams creates an error response with multiple parameters
func ErrorWithParams(predefinedErr error, params map[string]interface{}) *ErrorResponse {
	var errWithCode *ErrorWithCode
	if errors.As(predefinedErr, &errWithCode) {
		paramErr := errWithCode
		for k, v := range params {
			paramErr = paramErr.WithParam(k, v)
		}
		return &ErrorResponse{
			StatusCode: errWithCode.GetCode(),
			Err:        paramErr,
		}
	}

	// For other error types, create an internal server error
	internalErr := NewErrorWithCode(predefinedErr.Error(), ErrorInternalServer)
	for k, v := range params {
		internalErr = internalErr.WithParam(k, v)
	}
	return &ErrorResponse{
		StatusCode: ErrorInternalServer,
		Err:        internalErr,
	}
}
