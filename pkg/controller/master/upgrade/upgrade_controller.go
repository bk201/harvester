package upgrade

import (
	v1 "github.com/rancher/wrangler/pkg/generated/controllers/batch/v1"
	ctlcorev1 "github.com/rancher/wrangler/pkg/generated/controllers/core/v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/labels"

	harvesterv1 "github.com/harvester/harvester/pkg/apis/harvesterhci.io/v1beta1"
	ctlharvesterv1 "github.com/harvester/harvester/pkg/generated/controllers/harvesterhci.io/v1beta1"
	kubevirtctrl "github.com/harvester/harvester/pkg/generated/controllers/kubevirt.io/v1"
	upgradectlv1 "github.com/harvester/harvester/pkg/generated/controllers/upgrade.cattle.io/v1"
	"github.com/harvester/harvester/pkg/kf"
	"github.com/harvester/harvester/pkg/settings"
)

const (
	//system upgrade controller is deployed in cattle-system namespace
	upgradeNamespace               = "harvester-system"
	upgradeServiceAccount          = "system-upgrade-controller"
	harvesterSystemNamespace       = "harvester-system"
	harvesterVersionLabel          = "harvesterhci.io/version"
	harvesterUpgradeLabel          = "harvesterhci.io/upgrade"
	harvesterManagedLabel          = "harvesterhci.io/managed"
	harvesterLatestUpgradeLabel    = "harvesterhci.io/latestUpgrade"
	harvesterUpgradeComponentLabel = "harvesterhci.io/upgradeComponent"
	harvesterUpgradeImageLabel     = "harvesterhci.io/upgradeImage"
	upgradeImageRepository         = "rancher/harvester-upgrade"

	upgradeComponentRepo = "repo"
)

// upgradeHandler Creates Plan CRDs to trigger upgrades
type upgradeHandler struct {
	namespace     string
	nodeCache     ctlcorev1.NodeCache
	jobClient     v1.JobClient
	upgradeClient ctlharvesterv1.UpgradeClient
	upgradeCache  ctlharvesterv1.UpgradeCache
	planClient    upgradectlv1.PlanClient

	vmImageClient ctlharvesterv1.VirtualMachineImageClient
	vmImageCache  ctlharvesterv1.VirtualMachineImageCache
	vmClient      kubevirtctrl.VirtualMachineClient
	serviceClient ctlcorev1.ServiceClient
}

func (h *upgradeHandler) OnChanged(key string, upgrade *harvesterv1.Upgrade) (*harvesterv1.Upgrade, error) {
	if upgrade == nil || upgrade.DeletionTimestamp != nil {
		return upgrade, nil
	}

	repo := NewUpgradeRepo(upgrade, h)

	if harvesterv1.UpgradeCompleted.GetStatus(upgrade) == "" {
		if err := h.resetLatestUpgradeLabel(upgrade.Name); err != nil {
			return upgrade, err
		}

		toUpdate := upgrade.DeepCopy()
		initStatus(toUpdate)

		if upgrade.Spec.ISOURL != "" {
			if _, err := repo.CreateImageFromISO(); err != nil {
				return upgrade, err
			}
		} else {
			_, err := repo.GetImage(upgrade.Spec.Image)
			if err != nil {
				return upgrade, err
			}
			if toUpdate.Annotations == nil {
				toUpdate.Annotations = make(map[string]string)
			}
			toUpdate.Annotations[harvesterUpgradeImageLabel] = upgrade.Spec.Image
			setImageReadyCondition(toUpdate, corev1.ConditionTrue, "", "")
		}

		return h.upgradeClient.Update(toUpdate)
	}

	if harvesterv1.ImageReady.IsTrue(upgrade) && harvesterv1.RepoProvisioned.GetStatus(upgrade) == "" {
		repo := NewUpgradeRepo(upgrade, h)
		if err := repo.Bootstrap(); err != nil {
			return upgrade, err
		}
		toUpdate := upgrade.DeepCopy()
		toUpdate.Labels[upgradeStateLabel] = statePreparingRepo
		harvesterv1.RepoProvisioned.CreateUnknownIfNotExists(toUpdate)
		return h.upgradeClient.Update(toUpdate)
	}

	if harvesterv1.RepoProvisioned.IsTrue(upgrade) && harvesterv1.NodesUpgraded.GetStatus(upgrade) == "" {
		kf.Debugf("Start node upgrade")
		disableEviction, err := h.isSingleNodeCluster()
		if err != nil {
			return upgrade, err
		}
		// create plans if not initialized
		toUpdate := upgrade.DeepCopy()
		if _, err := h.planClient.Create(serverPlan(upgrade, disableEviction)); err != nil && !apierrors.IsAlreadyExists(err) {
			setNodesUpgradedCondition(toUpdate, corev1.ConditionFalse, "", err.Error())
			return h.upgradeClient.Update(toUpdate)
		}
		toUpdate.Labels[upgradeStateLabel] = stateUpgradingNodes
		harvesterv1.NodesUpgraded.CreateUnknownIfNotExists(toUpdate)
		return h.upgradeClient.Update(toUpdate)
	}

	if harvesterv1.NodesUpgraded.IsTrue(upgrade) && harvesterv1.SystemServicesUpgraded.GetStatus(upgrade) == "" {
		//nodes are upgraded, now upgrade the chart. Create a job to apply the manifests
		toUpdate := upgrade.DeepCopy()
		if _, err := h.jobClient.Create(applyManifestsJob(upgrade)); err != nil && !apierrors.IsAlreadyExists(err) {
			setHelmChartUpgradeStatus(toUpdate, corev1.ConditionFalse, "", err.Error())
			return h.upgradeClient.Update(toUpdate)
		}
		setHelmChartUpgradeStatus(toUpdate, corev1.ConditionUnknown, "", "")
		return h.upgradeClient.Update(toUpdate)
	}

	return upgrade, nil
}

func (h *upgradeHandler) isSingleNodeCluster() (bool, error) {
	nodes, err := h.nodeCache.List(labels.Everything())
	if err != nil {
		return false, err
	}
	return len(nodes) == 1, nil
}

func initStatus(upgrade *harvesterv1.Upgrade) {
	harvesterv1.UpgradeCompleted.CreateUnknownIfNotExists(upgrade)
	// harvesterv1.NodesUpgraded.CreateUnknownIfNotExists(upgrade)
	if upgrade.Labels == nil {
		upgrade.Labels = make(map[string]string)
	}
	upgrade.Labels[upgradeStateLabel] = stateCreatingUpgradeImage
	upgrade.Labels[harvesterLatestUpgradeLabel] = "true"
	upgrade.Status.PreviousVersion = settings.ServerVersion.Get()
}

func (h *upgradeHandler) resetLatestUpgradeLabel(latestUpgradeName string) error {
	sets := labels.Set{
		harvesterLatestUpgradeLabel: "true",
	}
	upgrades, err := h.upgradeCache.List(h.namespace, sets.AsSelector())
	if err != nil {
		return err
	}
	for _, upgrade := range upgrades {
		if upgrade.Name == latestUpgradeName {
			continue
		}
		toUpdate := upgrade.DeepCopy()
		delete(toUpdate.Labels, harvesterLatestUpgradeLabel)
		if _, err := h.upgradeClient.Update(toUpdate); err != nil {
			return err
		}
	}
	return nil
}
