package datavolume

import (
	"fmt"
	"strings"

	"github.com/sirupsen/logrus"
	"k8s.io/apimachinery/pkg/runtime"
	kv1 "kubevirt.io/client-go/api/v1"
	cdicore "kubevirt.io/containerized-data-importer/pkg/apis/core"
	"kubevirt.io/containerized-data-importer/pkg/apis/core/v1beta1"

	cdiv1beta1 "github.com/harvester/harvester/pkg/generated/controllers/cdi.kubevirt.io/v1beta1"
	"github.com/harvester/harvester/pkg/ref"
	werror "github.com/harvester/harvester/pkg/webhook/error"
	"github.com/harvester/harvester/pkg/webhook/types"
)

func NewValidator(dataVolumes cdiv1beta1.DataVolumeCache) types.Validator {
	return &dataVolumeValidator{
		dataVolumes: dataVolumes,
	}
}

type dataVolumeValidator struct {
	types.DefaultValidator
	dataVolumes cdiv1beta1.DataVolumeCache
}

func (v *dataVolumeValidator) Resource() types.Resource {
	return types.Resource{
		GroupName:  cdicore.GroupName,
		ObjectType: &v1beta1.DataVolume{},
	}
}

func (v *dataVolumeValidator) Delete(request *types.Request, oldObj runtime.Object) error {
	logrus.Debug("entering dataVolumeValidator.Delete")
	dataVolume := oldObj.(*v1beta1.DataVolume)

	dv, err := v.dataVolumes.Get(dataVolume.Namespace, dataVolume.Name)
	if err != nil {
		return werror.NewInvalidError(err.Error(), "metadata.name")
	}

	annotationSchemaOwners, err := ref.GetSchemaOwnersFromAnnotation(dv)
	if err != nil {
		message := fmt.Sprintf("failed to get schema owners from annotation: %s", err)
		return werror.NewInternalError(message)
	}

	attachedList := annotationSchemaOwners.List(kv1.VirtualMachineGroupVersionKind.GroupKind())
	if len(attachedList) != 0 {
		message := fmt.Sprintf("can not delete the volume %s which is currently attached to VMs: %s", dataVolume.Name, strings.Join(attachedList, ", "))
		return werror.NewInvalidError(message, "")
	}

	if len(dv.OwnerReferences) == 0 {
		return nil
	}

	var ownerList []string
	for _, owner := range dv.OwnerReferences {
		if owner.Kind == kv1.VirtualMachineGroupVersionKind.Kind {
			ownerList = append(ownerList, owner.Name)
		}
	}
	if len(ownerList) > 0 {
		message := fmt.Sprintf("can not delete the volume %s which is currently owned by these VMs: %s", dataVolume.Name, strings.Join(ownerList, ","))
		return werror.NewInvalidError(message, "")
	}

	return nil
}
