package virtualmachineimage

import (
	"github.com/sirupsen/logrus"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"

	"github.com/harvester/harvester/pkg/apis/harvesterhci.io"
	"github.com/harvester/harvester/pkg/apis/harvesterhci.io/v1beta1"
	ctlharvesterv1 "github.com/harvester/harvester/pkg/generated/controllers/harvesterhci.io/v1beta1"
	werror "github.com/harvester/harvester/pkg/webhook/error"
	"github.com/harvester/harvester/pkg/webhook/types"
)

const (
	fieldDisplayName = "spec.displayName"
)

func NewValidator(vmimages ctlharvesterv1.VirtualMachineImageCache) types.Validator {
	return &virtualMachineImageValidator{
		vmimages: vmimages,
	}
}

type virtualMachineImageValidator struct {
	types.DefaultValidator

	vmimages ctlharvesterv1.VirtualMachineImageCache
}

func (v *virtualMachineImageValidator) Resource() types.Resource {
	return types.Resource{
		GroupName:  harvesterhci.GroupName,
		ObjectType: &v1beta1.VirtualMachineImage{},
	}
}

func (v *virtualMachineImageValidator) Create(request *types.Request, newObj runtime.Object) error {
	logrus.Debug("entering virtualMachineImageValidator.Create")
	newImage := newObj.(*v1beta1.VirtualMachineImage)

	if newImage.Spec.DisplayName == "" {
		return werror.NewInvalidError("displayName is required", fieldDisplayName)
	}

	allImages, err := v.vmimages.List(newImage.Namespace, labels.Everything())
	if err != nil {
		return werror.NewInternalError(err.Error())
	}
	for _, image := range allImages {
		if newImage.Name == image.Name {
			continue
		}
		if newImage.Spec.DisplayName == image.Spec.DisplayName {
			return werror.NewConflict("A resource with the same name exists")
		}
	}

	return nil
}

func (v *virtualMachineImageValidator) Update(request *types.Request, oldObj runtime.Object, newObj runtime.Object) error {
	logrus.Debug("entering virtualMachineImageValidator.Update")
	return v.Create(request, newObj)
}
