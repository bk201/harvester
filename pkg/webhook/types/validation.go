package types

import (
	"fmt"

	"github.com/rancher/wrangler/pkg/webhook"
	"github.com/sirupsen/logrus"
	admissionv1 "k8s.io/api/admission/v1"
	"k8s.io/apimachinery/pkg/runtime"

	werror "github.com/harvester/harvester/pkg/webhook/error"
)

type Validator interface {
	Create(request *webhook.Request, newObj runtime.Object) error
	Update(request *webhook.Request, oldObj runtime.Object, newObj runtime.Object) error
	Delete(request *webhook.Request, oldObj runtime.Object) error
	Connect(request *webhook.Request, newObj runtime.Object) error

	Info() ValidatorInfo
}

type ValidationHandler struct {
	AdmissionHandler
	validator Validator
}

func NewValidationHandler(validator Validator) *ValidationHandler {
	return &ValidationHandler{
		validator: validator,
	}
}

func (v *ValidationHandler) Admit(response *webhook.Response, request *webhook.Request) error {
	oldObj, newObj, err := v.decodeObjects(request)
	if err != nil {
		response.Allowed = false
		response.Result = werror.NewInternalError(err.Error()).AsResult()
		return nil
	}

	switch request.Operation {
	case admissionv1.Create:
		err = v.validator.Create(request, newObj)
	case admissionv1.Delete:
		err = v.validator.Delete(request, oldObj)
	case admissionv1.Update:
		err = v.validator.Update(request, oldObj, newObj)
	case admissionv1.Connect:
		err = v.validator.Connect(request, newObj)
	default:
		err = fmt.Errorf("Unsupport operation %s", request.Operation)
	}

	if err != nil {
		var admitErr werror.AdmitError
		if e, ok := err.(werror.AdmitError); ok {
			admitErr = e
		} else {
			admitErr = werror.NewInternalError(err.Error())
		}
		response.Allowed = false
		response.Result = admitErr.AsResult()
		return nil
	}

	response.Allowed = true
	return nil
}

type DefaultValidator struct {
}

func (v *DefaultValidator) Create(request *webhook.Request, newObj runtime.Object) error {
	logrus.Info("entering DefaultValidator.Create")
	return nil
}

func (v *DefaultValidator) Update(request *webhook.Request, oldObj runtime.Object, newObj runtime.Object) error {
	logrus.Info("entering DefaultValidator.Update")
	return nil
}

func (v *DefaultValidator) Delete(request *webhook.Request, oldObj runtime.Object) error {
	logrus.Info("entering DefaultValidator.Delete")
	return nil
}

func (v *DefaultValidator) Connect(request *webhook.Request, newObj runtime.Object) error {
	logrus.Info("entering DefaultValidator.Connect")
	return nil
}
