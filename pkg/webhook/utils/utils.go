package utils

import (
	"net/http"

	"github.com/rancher/wrangler/pkg/webhook"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// 400
func RejectBadRequest(response *webhook.Response, message string) error {
	response.Allowed = false
	response.Result = &metav1.Status{
		Status:  "Failure",
		Message: message,
		Code:    http.StatusBadRequest,
		Reason:  metav1.StatusReasonBadRequest,
	}
	return nil
}

// 400
func RejectMethodNotAllowed(response *webhook.Response, message string) error {
	response.Allowed = false
	response.Result = &metav1.Status{
		Status:  "Failure",
		Message: message,
		Code:    http.StatusMethodNotAllowed,
		Reason:  metav1.StatusReasonMethodNotAllowed,
	}
	return nil
}

// 422
func RejectInvalid(response *webhook.Response, message string, field string) error {
	response.Allowed = false
	response.Result = &metav1.Status{
		Status:  "Failure",
		Message: message,
		Code:    http.StatusUnprocessableEntity,
		Reason:  metav1.StatusReasonInvalid,
		Details: &metav1.StatusDetails{
			Causes: []metav1.StatusCause{
				{
					Type:    metav1.CauseTypeFieldValueInvalid,
					Message: message,
					Field:   field,
				},
			},
		},
	}
	return nil
}

// 409
func RejectConflict(response *webhook.Response, message string) error {
	response.Allowed = false
	response.Result = &metav1.Status{
		Status:  "Failure",
		Message: message,
		Code:    http.StatusConflict,
		Reason:  metav1.StatusReasonConflict,
	}
	return nil
}

// 500
func RejectInternalError(response *webhook.Response, message string) error {
	response.Allowed = false
	response.Result = &metav1.Status{
		Status:  "Failure",
		Message: message,
		Code:    http.StatusInternalServerError,
		Reason:  metav1.StatusReasonInternalError,
	}
	return nil
}

func Allow(response *webhook.Response) error {
	response.Allowed = true
	return nil
}
