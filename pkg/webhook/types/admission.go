package types

import (
	"fmt"
	"strings"

	"github.com/rancher/wrangler/pkg/webhook"
	"github.com/sirupsen/logrus"
	admissionv1 "k8s.io/api/admission/v1"
	"k8s.io/apimachinery/pkg/runtime"

	"github.com/harvester/harvester/pkg/webhook/config"
	werror "github.com/harvester/harvester/pkg/webhook/error"
)

// JSON Patch operations to mutate input data. See https://jsonpatch.com/ for more information.
type PatchOps []string

// A Admitter interface is used by AdmissionHandler to check if a operation is allowed.
type Admitter interface {
	// Create checks if a CREATE operation is allowed. If no error is returned, the operation is allowed.
	// PatchOps contains JSON patch operations to be applied on the API object received by the server.
	Create(request *Request, newObj runtime.Object) (PatchOps, error)

	// Update checks if a UPDATE operation is allowed. If no error is returned, the operation is allowed.
	// PatchOps contains JSON patch operations to be applied on the API object received by the server.
	Update(request *Request, oldObj runtime.Object, newObj runtime.Object) (PatchOps, error)

	// Delete checks if a DELETE operation is allowed. If no error is returned, the operation is allowed.
	// PatchOps contains JSON patch operations to be applied on the API object received by the server.
	Delete(request *Request, oldObj runtime.Object) (PatchOps, error)

	// Connect checks if a CONNECT operation is allowed. If no error is returned, the operation is allowed.
	// PatchOps contains JSON patch operations to be applied on the API object received by the server.
	Connect(request *Request, newObj runtime.Object) (PatchOps, error)

	// Resource returns the resource that the admitter works on.
	Resource() Resource
}

type AdmissionHandler struct {
	admitter Admitter
	options  *config.Options
}

func NewAdmissionHandler(admitter Admitter, options *config.Options) *AdmissionHandler {
	if err := admitter.Resource().Validate(); err != nil {
		panic(err.Error())
	}
	return &AdmissionHandler{
		admitter: admitter,
		options:  options,
	}
}

func (v *AdmissionHandler) Admit(response *webhook.Response, request *webhook.Request) error {
	return v.admit(response, NewRequest(request, v.options))
}

func (v *AdmissionHandler) admit(response *webhook.Response, req *Request) error {
	logrus.Debugf("%s entering admit", req)
	if req.IsGarbageCollection() {
		logrus.Debugf("%s allow gabage collection", req)
		response.Allowed = true
		return nil
	}

	oldObj, newObj, err := req.DecodeObjects()
	if err != nil {
		logrus.Errorf("%s fail to decode objects: %s", req, err)
		response.Result = werror.NewInternalError(err.Error()).AsResult()
		response.Allowed = false
		return nil
	}

	var patchOps PatchOps

	switch req.Operation {
	case admissionv1.Create:
		patchOps, err = v.admitter.Create(req, newObj)
	case admissionv1.Delete:
		patchOps, err = v.admitter.Delete(req, oldObj)
	case admissionv1.Update:
		patchOps, err = v.admitter.Update(req, oldObj, newObj)
	case admissionv1.Connect:
		patchOps, err = v.admitter.Connect(req, newObj)
	default:
		err = fmt.Errorf("unsupported operation %s", req.Operation)
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
		logrus.Debugf("%s operation is rejected: %s", req, admitErr)
		return nil
	}

	if len(patchOps) > 0 {
		patchType := admissionv1.PatchTypeJSONPatch
		patchData := fmt.Sprintf("[%s]", strings.Join(patchOps, ","))
		response.PatchType = &patchType
		response.Patch = []byte(patchData)
		logrus.Debugf("%s patchOps: %s", req, patchData)
	}

	logrus.Debugf("%s operation is allowed", req)
	response.Allowed = true
	return nil
}

func (v *AdmissionHandler) decodeObjects(request *Request) (oldObj runtime.Object, newObj runtime.Object, err error) {
	operation := request.Operation
	if operation == admissionv1.Delete || operation == admissionv1.Update {
		oldObj, err = request.DecodeOldObject()
		if err != nil {
			return
		}
		if operation == admissionv1.Delete {
			// no new object for DELETE operation
			return
		}
	}
	newObj, err = request.DecodeObject()
	return
}
