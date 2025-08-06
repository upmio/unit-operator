package common

import (
	"fmt"
	"go.uber.org/zap"
)

// Response interface for response types with Message field
type Response interface {
	GetMessage() string
}

// ResponseWithMessage represents any response type that has a Message field
type ResponseWithMessage struct {
	Message string `json:"message"`
}

// GetMessage implements Response interface
func (r *ResponseWithMessage) GetMessage() string {
	return r.Message
}

// LogAndReturnError logs error and returns standardized response
// This function creates a response object with the given message constructor function
func LogAndReturnError[T any](logger *zap.SugaredLogger, newResponse func(string) *T, errMsg string, err error) (*T, error) {
	if err != nil {
		errMsg = fmt.Sprintf("%s: %v", errMsg, err)
	}
	logger.Error(errMsg)
	response := newResponse(errMsg)
	return response, fmt.Errorf(errMsg)
}

// LogAndReturnSuccess logs success and returns response
// This function creates a response object with the given message constructor function
func LogAndReturnSuccess[T any](logger *zap.SugaredLogger, newResponse func(string) *T, msg string) (*T, error) {
	logger.Info(msg)
	response := newResponse(msg)
	return response, nil
}

// EventRecorder interface for sending events
type EventRecorder interface {
	SendWarningEventToUnit(unitName, namespace, reason, message string) error
	SendNormalEventToUnit(unitName, namespace, reason, message string) error
}

// LogAndReturnErrorWithEvent logs error, sends warning event, and returns standardized response
func LogAndReturnErrorWithEvent[T any](logger *zap.SugaredLogger, recorder EventRecorder, newResponse func(string) *T, unitName, namespace, reason, errMsg string, err error) (*T, error) {
	if err != nil {
		errMsg = fmt.Sprintf("%s: %v", errMsg, err)
	}
	recorder.SendWarningEventToUnit(unitName, namespace, reason, errMsg)
	logger.Error(errMsg)
	response := newResponse(errMsg)
	return response, fmt.Errorf(errMsg)
}

// LogAndReturnSuccessWithEvent logs success, sends normal event, and returns response
func LogAndReturnSuccessWithEvent[T any](logger *zap.SugaredLogger, recorder EventRecorder, newResponse func(string) *T, unitName, namespace, reason, msg string) (*T, error) {
	recorder.SendNormalEventToUnit(unitName, namespace, reason, msg)
	logger.Info(msg)
	response := newResponse(msg)
	return response, nil
}