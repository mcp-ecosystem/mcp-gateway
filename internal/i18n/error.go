package i18n

import (
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/mcp-ecosystem/mcp-gateway/internal/common/cnst"
)

// ErrorCode represents an HTTP status code
type ErrorCode int

// Standard HTTP status codes
const (
	ErrorBadRequest         ErrorCode = http.StatusBadRequest
	ErrorUnauthorized       ErrorCode = http.StatusUnauthorized
	ErrorForbidden          ErrorCode = http.StatusForbidden
	ErrorNotFound           ErrorCode = http.StatusNotFound
	ErrorMethodNotAllowed   ErrorCode = http.StatusMethodNotAllowed
	ErrorConflict           ErrorCode = http.StatusConflict
	ErrorInternalServer     ErrorCode = http.StatusInternalServerError
	ErrorNotImplemented     ErrorCode = http.StatusNotImplemented
	ErrorServiceUnavailable ErrorCode = http.StatusServiceUnavailable
	ErrorTooManyRequests    ErrorCode = http.StatusTooManyRequests
	ErrorGatewayTimeout     ErrorCode = http.StatusGatewayTimeout
	ErrorUnsupportedMedia   ErrorCode = http.StatusUnsupportedMediaType
)

// I18nError represents an internationalized error
type I18nError struct {
	// MessageID is the key used for translation lookup
	MessageID string
	// DefaultMessage is used when translation is not available
	DefaultMessage string
	// Data holds template parameters for the message
	Data map[string]interface{}
}

// New creates a new I18nError with the given message ID
func New(messageID string) *I18nError {
	return &I18nError{
		MessageID:      messageID,
		DefaultMessage: messageID,
		Data:           make(map[string]interface{}),
	}
}

// NewWithMessage creates a new I18nError with a message ID and default message
func NewWithMessage(messageID, defaultMessage string) *I18nError {
	return &I18nError{
		MessageID:      messageID,
		DefaultMessage: defaultMessage,
		Data:           make(map[string]interface{}),
	}
}

// WithData adds template data to the error
func (e *I18nError) WithData(data map[string]interface{}) *I18nError {
	if data != nil {
		for k, v := range data {
			e.Data[k] = v
		}
	}
	return e
}

// WithParam adds a single template parameter to the error
func (e *I18nError) WithParam(key string, value interface{}) *I18nError {
	e.Data[key] = value
	return e
}

// Error implements the error interface
func (e *I18nError) Error() string {
	// 使用默认语言翻译消息
	t := GetTranslator()
	if t != nil {
		translated := t.Translate(e.MessageID, defaultLang, e.Data)
		if translated != e.MessageID {
			return translated
		}
	}

	// 如果没有翻译，返回默认消息
	if len(e.Data) == 0 {
		return e.DefaultMessage
	}

	// 尝试格式化消息
	msg := e.DefaultMessage
	for k, v := range e.Data {
		placeholder := fmt.Sprintf("{{.%s}}", k)
		msg = strings.Replace(msg, placeholder, fmt.Sprintf("%v", v), -1)
	}
	return msg
}

// GetMessageID returns the message ID for translation
func (e *I18nError) GetMessageID() string {
	return e.MessageID
}

// GetData returns the template data
func (e *I18nError) GetData() map[string]interface{} {
	return e.Data
}

// TranslateByContext translates the error based on the context's language preference
func (e *I18nError) TranslateByContext(c *gin.Context) string {
	lang, exists := c.Get(cnst.XLang)
	if !exists || lang == "" {
		lang = defaultLang
	}

	langStr, ok := lang.(string)
	if !ok {
		langStr = defaultLang
	}

	t := GetTranslator()
	if t != nil {
		translated := t.Translate(e.MessageID, langStr, e.Data)
		if translated != e.MessageID {
			return translated
		}
	}
	return e.Error()
}

// TranslateByRequest translates the error based on the HTTP request's language preference
func (e *I18nError) TranslateByRequest(r *http.Request) string {
	lang := getLanguageFromRequest(r)
	t := GetTranslator()
	if t != nil {
		translated := t.Translate(e.MessageID, lang, e.Data)
		if translated != e.MessageID {
			return translated
		}
	}
	return e.Error()
}

// ErrorWithCode is an error with a code that can be used in API responses
type ErrorWithCode struct {
	*I18nError
	Code ErrorCode
}

// NewErrorWithCode creates a new error with a code
func NewErrorWithCode(messageID string, code ErrorCode) *ErrorWithCode {
	return &ErrorWithCode{
		I18nError: New(messageID),
		Code:      code,
	}
}

// WithData adds template data to the error
func (e *ErrorWithCode) WithData(data map[string]interface{}) *ErrorWithCode {
	e.I18nError.WithData(data)
	return e
}

// WithParam adds a single template parameter to the error
func (e *ErrorWithCode) WithParam(key string, value interface{}) *ErrorWithCode {
	e.I18nError.WithParam(key, value)
	return e
}

// WithHttpCode allows changing the HTTP status code dynamically
func (e *ErrorWithCode) WithHttpCode(code ErrorCode) *ErrorWithCode {
	newErr := &ErrorWithCode{
		I18nError: e.I18nError,
		Code:      code,
	}
	return newErr
}

// GetCode returns the error code
func (e *ErrorWithCode) GetCode() ErrorCode {
	return e.Code
}

// IsI18nError checks if an error is an I18nError
func IsI18nError(err error) bool {
	if err == nil {
		return false
	}

	var i18nErr *I18nError
	return errors.As(err, &i18nErr)
}

// AsI18nError converts an error to an I18nError if possible, or returns nil
func AsI18nError(err error) *I18nError {
	var i18nErr *I18nError
	if errors.As(err, &i18nErr) {
		return i18nErr
	}
	return nil
}

// TranslateError translates an error using the context's language preference
func TranslateError(c *gin.Context, err error) string {
	if err == nil {
		return ""
	}

	var i18nErr *I18nError
	if errors.As(err, &i18nErr) {
		return i18nErr.TranslateByContext(c)
	}

	var errWithCode *ErrorWithCode
	if errors.As(err, &errWithCode) {
		return errWithCode.TranslateByContext(c)
	}

	// Try to see if it's a message ID
	errMsg := err.Error()
	if IsI18nError(err) {
		lang, exists := c.Get(cnst.XLang)
		if !exists || lang == "" {
			lang = defaultLang
		}

		langStr, ok := lang.(string)
		if !ok {
			langStr = defaultLang
		}

		t := GetTranslator()
		if t != nil {
			translated := t.Translate(errMsg, langStr, nil)
			if translated != errMsg {
				return translated
			}
		}
	}

	return errMsg
}
