package virtualmachineimage

import (
	"github.com/rancher/wrangler/pkg/webhook"
	"github.com/sirupsen/logrus"
	"k8s.io/apimachinery/pkg/labels"

	"github.com/harvester/harvester/pkg/apis/harvesterhci.io/v1beta1"
	ctlharvesterv1 "github.com/harvester/harvester/pkg/generated/controllers/harvesterhci.io/v1beta1"
	"github.com/harvester/harvester/pkg/webhook/utils"
)

func NewValidator(vmimages ctlharvesterv1.VirtualMachineImageCache) webhook.Handler {
	return &virtualMachineImageValidator{
		vmimages: vmimages,
	}
}

type virtualMachineImageValidator struct {
	vmimages ctlharvesterv1.VirtualMachineImageCache
}

func (v *virtualMachineImageValidator) Admit(response *webhook.Response, request *webhook.Request) error {
	logrus.Debug("entering virtualMachineImageValidator.Admit")
	newImage, err := vmimageObject(request)
	if err != nil {
		return utils.RejectInternalError(response, err.Error())
	}

	if newImage.Spec.DisplayName == "" {
		return utils.RejectInvalid(response, "displayName is required", "spec.displayName")
	}

	allImages, err := v.vmimages.List(newImage.Namespace, labels.Everything())
	if err != nil {
		return utils.RejectInternalError(response, err.Error())
	}
	for _, image := range allImages {
		if newImage.Name == image.Name {
			continue
		}
		if newImage.Spec.DisplayName == image.Spec.DisplayName {
			return utils.RejectConflict(response, "A resource with the same name exists.")
		}
	}

	return utils.Allow(response)
}

func vmimageObject(request *webhook.Request) (*v1beta1.VirtualMachineImage, error) {
	object, err := request.DecodeObject()
	if err != nil {
		return nil, err
	}
	return object.(*v1beta1.VirtualMachineImage), nil
}
