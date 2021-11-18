package upgrade

import (
	"fmt"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"

	harvesterv1 "github.com/harvester/harvester/pkg/apis/harvesterhci.io/v1beta1"
	ctlharvesterv1 "github.com/harvester/harvester/pkg/generated/controllers/harvesterhci.io/v1beta1"
	"github.com/harvester/harvester/pkg/kf"
)

type vmImageHandler struct {
	namespace     string
	upgradeClient ctlharvesterv1.UpgradeClient
	upgradeCache  ctlharvesterv1.UpgradeCache
}

func (h *vmImageHandler) OnChanged(key string, image *harvesterv1.VirtualMachineImage) (*harvesterv1.VirtualMachineImage, error) {
	if image == nil || image.DeletionTimestamp != nil || image.Labels == nil || image.Namespace != upgradeNamespace || image.Labels[harvesterUpgradeLabel] == "" {
		return image, nil
	}

	kf.Debugf("Image change: %+v", image)
	upgrade, err := h.upgradeCache.Get(upgradeNamespace, image.Labels[harvesterUpgradeLabel])
	if err != nil {
		if apierrors.IsNotFound(err) {
			return image, nil
		}
		return nil, err
	}

	if harvesterv1.ImageImported.GetStatus(image) == string(corev1.ConditionTrue) {
		toUpdate := upgrade.DeepCopy()
		if toUpdate.Annotations == nil {
			toUpdate.Annotations = make(map[string]string)
		}
		toUpdate.Annotations[harvesterUpgradeImageLabel] = fmt.Sprintf("%s/%s", image.Namespace, image.Name)
		setImageReadyCondition(toUpdate, corev1.ConditionTrue, "", "")
		_, err := h.upgradeClient.Update(toUpdate)
		return image, err
	}

	return image, nil
}
