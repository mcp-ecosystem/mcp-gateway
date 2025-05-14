package i18n

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
)

// RespondWithError sends an appropriate HTTP error response for the given error
func RespondWithError(c *gin.Context, err error) {
	if err == nil {
		return
	}

	// Default status for any kind of error
	statusCode := http.StatusInternalServerError
	errorMsg := TranslateError(c, err)

	// Try to extract status code from error
	var errWithCode *ErrorWithCode
	if errors.As(err, &errWithCode) {
		statusCode = int(errWithCode.GetCode())
	}

	c.JSON(statusCode, gin.H{"error": errorMsg})
}

// RespondWithSuccess sends a success HTTP response with an internationalized message
func RespondWithSuccess(c *gin.Context, statusCode int, msgID string, data map[string]any, payload interface{}) {
	message := TranslateMessage(c, msgID, data)

	response := gin.H{
		"message": message,
	}

	// Add data to the top level of the response
	if data != nil {
		for k, v := range data {
			response[k] = v
		}
	}

	// Add additional payload if provided
	if payload != nil {
		switch p := payload.(type) {
		case map[string]any:
			for k, v := range p {
				response[k] = v
			}
		case gin.H:
			for k, v := range p {
				response[k] = v
			}
		default:
			response["data"] = payload
		}
	}

	c.JSON(statusCode, response)
}

// SuccessResponse represents a response with success message
type SuccessResponse struct {
	StatusCode int
	MsgID      string
	Data       map[string]interface{}
	Payload    interface{}
}

// With adds a key-value pair to the response data
func (r *SuccessResponse) With(key string, value interface{}) *SuccessResponse {
	if r.Data == nil {
		r.Data = make(map[string]interface{})
	}
	r.Data[key] = value
	return r
}

// WithData adds multiple key-value pairs to the response data
func (r *SuccessResponse) WithData(data map[string]interface{}) *SuccessResponse {
	if r.Data == nil {
		r.Data = make(map[string]interface{})
	}
	for k, v := range data {
		r.Data[k] = v
	}
	return r
}

// WithPayload sets the payload for the response
func (r *SuccessResponse) WithPayload(payload interface{}) *SuccessResponse {
	r.Payload = payload
	return r
}

// Send sends the response to the client
func (r *SuccessResponse) Send(c *gin.Context) {
	RespondWithSuccess(c, r.StatusCode, r.MsgID, r.Data, r.Payload)
}

// Success creates a new success response with status code 200
func Success(msgID string) *SuccessResponse {
	return &SuccessResponse{
		StatusCode: http.StatusOK,
		MsgID:      msgID,
	}
}

// Created creates a new success response with status code 201
func Created(msgID string) *SuccessResponse {
	return &SuccessResponse{
		StatusCode: http.StatusCreated,
		MsgID:      msgID,
	}
}

// RespondOK sends a success HTTP response with status code 200
func RespondOK(c *gin.Context, msgID string, data map[string]interface{}, payload interface{}) {
	RespondWithSuccess(c, http.StatusOK, msgID, data, payload)
}

// RespondCreated sends a success HTTP response with status code 201
func RespondCreated(c *gin.Context, msgID string, data map[string]interface{}, payload interface{}) {
	RespondWithSuccess(c, http.StatusCreated, msgID, data, payload)
}
