package types

import (
	"github.com/rancher/wrangler/pkg/webhook"
	admissionv1 "k8s.io/api/admission/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

type ValidatorInfo struct {
	Kind       string
	GroupName  string
	ObjectType runtime.Object
}

type AdmissionHandler struct {
}

func (v *AdmissionHandler) decodeObjects(request *webhook.Request) (oldObj runtime.Object, newObj runtime.Object, err error) {
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
