package types

import (
	"fmt"
	"strings"

	"github.com/rancher/wrangler/pkg/webhook"
	"github.com/sirupsen/logrus"
	admissionv1 "k8s.io/api/admission/v1"
	"k8s.io/apimachinery/pkg/runtime"

	werror "github.com/harvester/harvester/pkg/webhook/error"
)

// JSON Patch operations to mutate input data. See https://jsonpatch.com/
type PatchOps []string

type Mutator interface {
	Create(request *webhook.Request, newObj runtime.Object) (PatchOps, error)
	Update(request *webhook.Request, oldObj runtime.Object, newObj runtime.Object) (PatchOps, error)
	Delete(request *webhook.Request, oldObj runtime.Object) (PatchOps, error)
	Connect(request *webhook.Request, newObj runtime.Object) (PatchOps, error)

	Info() ValidatorInfo
}

type MutationHandler struct {
	AdmissionHandler
	mutator Mutator
}

func NewMutationHandler(mutator Mutator) *MutationHandler {
	return &MutationHandler{
		mutator: mutator,
	}
}

func (v *MutationHandler) Admit(response *webhook.Response, request *webhook.Request) error {
	oldObj, newObj, err := v.decodeObjects(request)
	if err != nil {
		response.Allowed = false
		response.Result = werror.NewInternalError(err.Error()).AsResult()
		return nil
	}

	var patchOps PatchOps

	switch request.Operation {
	case admissionv1.Create:
		patchOps, err = v.mutator.Create(request, newObj)
	case admissionv1.Delete:
		patchOps, err = v.mutator.Delete(request, oldObj)
	case admissionv1.Update:
		patchOps, err = v.mutator.Update(request, oldObj, newObj)
	case admissionv1.Connect:
		patchOps, err = v.mutator.Connect(request, newObj)
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

	if len(patchOps) > 0 {
		patchType := admissionv1.PatchTypeJSONPatch
		patchData := fmt.Sprintf("[%s]", strings.Join(patchOps, ","))
		response.PatchType = &patchType
		response.Patch = []byte(patchData)
	}

	response.Allowed = true
	return nil
}

type DefaultMutator struct {
}

func (v *DefaultMutator) Create(request *webhook.Request, newObj runtime.Object) (PatchOps, error) {
	logrus.Info("entering DefaultMutator.Create")
	return nil, nil
}

func (v *DefaultMutator) Update(request *webhook.Request, oldObj runtime.Object, newObj runtime.Object) (PatchOps, error) {
	logrus.Info("entering DefaultMutator.Update")
	return nil, nil
}

func (v *DefaultMutator) Delete(request *webhook.Request, oldObj runtime.Object) (PatchOps, error) {
	logrus.Info("entering DefaultMutator.Delete")
	return nil, nil
}

func (v *DefaultMutator) Connect(request *webhook.Request, newObj runtime.Object) (PatchOps, error) {
	logrus.Info("entering DefaultMutator.Connect")
	return nil, nil
}
