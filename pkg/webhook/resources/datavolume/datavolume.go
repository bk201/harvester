package datavolume

import (
	"fmt"
	"strings"

	"github.com/rancher/wrangler/pkg/webhook"
	"github.com/sirupsen/logrus"
	admissionv1 "k8s.io/api/admission/v1"
	"k8s.io/apimachinery/pkg/runtime"
	kv1 "kubevirt.io/client-go/api/v1"
	"kubevirt.io/containerized-data-importer/pkg/apis/core/v1beta1"

	cdiv1beta1 "github.com/harvester/harvester/pkg/generated/controllers/cdi.kubevirt.io/v1beta1"
	"github.com/harvester/harvester/pkg/ref"
	"github.com/harvester/harvester/pkg/webhook/utils"
)

func NewValidator(dataVolumes cdiv1beta1.DataVolumeCache) webhook.Handler {
	return &dataVolumeValidator{
		dataVolumes: dataVolumes,
	}
}

type dataVolumeValidator struct {
	dataVolumes cdiv1beta1.DataVolumeCache
}

func (v *dataVolumeValidator) Admit(response *webhook.Response, request *webhook.Request) error {
	logrus.Debug("entering dataVolumeValidator.Admit")
	dataVolume, err := dataVolumeObject(request)
	if err != nil {
		return utils.RejectInternalError(response, err.Error())
	}

	if request.Operation == admissionv1.Delete {
		dv, err := v.dataVolumes.Get(dataVolume.Namespace, dataVolume.Name)
		if err != nil {
			return utils.RejectInvalid(response, err.Error(), "metadata.name")
		}

		annotationSchemaOwners, err := ref.GetSchemaOwnersFromAnnotation(dv)
		if err != nil {
			message := fmt.Sprintf("failed to get schema owners from annotation: %s", err)
			return utils.RejectInternalError(response, message)
		}

		attachedList := annotationSchemaOwners.List(kv1.VirtualMachineGroupVersionKind.GroupKind())
		if len(attachedList) != 0 {
			message := fmt.Sprintf("can not delete the volume %s which is currently attached to VMs: %s", dataVolume.Name, strings.Join(attachedList, ", "))
			return utils.RejectInvalid(response, message, "")
		}

		if len(dv.OwnerReferences) == 0 {
			return utils.Allow(response)
		}

		var ownerList []string
		for _, owner := range dv.OwnerReferences {
			if owner.Kind == kv1.VirtualMachineGroupVersionKind.Kind {
				ownerList = append(ownerList, owner.Name)
			}
		}

		if len(ownerList) > 0 {
			message := fmt.Sprintf("can not delete the volume %s which is currently owned by these VMs: %s", dataVolume.Name, strings.Join(ownerList, ","))
			return utils.RejectInvalid(response, message, "")
		}
	}

	return utils.Allow(response)
}

func dataVolumeObject(request *webhook.Request) (*v1beta1.DataVolume, error) {
	var object runtime.Object
	var err error
	if request.Operation == admissionv1.Delete {
		object, err = request.DecodeOldObject()
	} else {
		object, err = request.DecodeObject()
	}
	if err != nil {
		return nil, err
	}
	return object.(*v1beta1.DataVolume), nil
}
