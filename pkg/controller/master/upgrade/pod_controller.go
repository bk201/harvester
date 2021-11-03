package upgrade

import (
	"reflect"

	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/labels"

	harvesterv1 "github.com/harvester/harvester/pkg/apis/harvesterhci.io/v1beta1"
	ctlharvesterv1 "github.com/harvester/harvester/pkg/generated/controllers/harvesterhci.io/v1beta1"
	upgradev1 "github.com/harvester/harvester/pkg/generated/controllers/upgrade.cattle.io/v1"
	"github.com/harvester/harvester/pkg/kf"
)

// podHandler syncs upgrade CRD status on upgrade pod status changes
type podHandler struct {
	namespace     string
	planCache     upgradev1.PlanCache
	upgradeClient ctlharvesterv1.UpgradeClient
	upgradeCache  ctlharvesterv1.UpgradeCache
}

func (h *podHandler) OnChanged(key string, pod *v1.Pod) (*v1.Pod, error) {
	if pod == nil || pod.DeletionTimestamp != nil || pod.Labels == nil || pod.Namespace != upgradeNamespace || pod.Labels[harvesterUpgradeLabel] == "" {
		return pod, nil
	}

	kf.Debugf("pod change: %+v", pod.Name)
	upgrade, err := h.upgradeCache.Get(upgradeNamespace, pod.Labels[harvesterUpgradeLabel])
	if err != nil {
		return nil, err
	}

	component := pod.Labels[harvesterUpgradeComponentLabel]
	switch upgrade.Labels[upgradeStateLabel] {
	case statePreparingRepo:
		if component == upgradeComponentRepo && len(pod.Status.ContainerStatuses) > 0 {
			kf.Debugf("pod of upgrade %q: ready: %+v", pod.Labels[harvesterUpgradeLabel], pod.Status.ContainerStatuses[0].Ready)
			if pod.Status.ContainerStatuses[0].Ready {
				toUpdate := upgrade.DeepCopy()
				toUpdate.Labels[upgradeStateLabel] = stateUpgrading
				setRepoProvisionedCondition(toUpdate, v1.ConditionTrue, "", "")
				_, err = h.upgradeClient.Update(toUpdate)
				return pod, err
			}
		}
	default:
		kf.Debugf("pod controller: nothing to do, no state")
	}

	return pod, nil

	// switch podType
	// case bootstrap
	// case upgrade Rancher
	// case upgrade RKE2 (maybe no need)
	// case upgrade Harvester

	chartName := pod.Labels[helmChartLabel]
	planName := pod.Labels[upgradePlanLabel]
	nodeName := pod.Labels[upgradeNodeLabel]
	if chartName == harvesterChartname {
		return h.syncHelmChartPod(pod)
	} else if planName != "" && nodeName != "" {
		return h.syncNodeUpgradePod(pod, planName, nodeName)
	}
	return pod, nil
}

func (h *podHandler) syncHelmChartPod(pod *v1.Pod) (*v1.Pod, error) {
	if pod.Status.Phase == v1.PodSucceeded {
		return pod, nil
	}

	sets := labels.Set{
		harvesterLatestUpgradeLabel: "true",
	}
	onGoingUpgrades, err := h.upgradeCache.List(h.namespace, sets.AsSelector())
	if err != nil {
		return pod, err
	}
	if len(onGoingUpgrades) == 0 {
		return pod, nil
	}
	upgrade := onGoingUpgrades[0]
	if !harvesterv1.SystemServicesUpgraded.IsUnknown(upgrade) {
		return pod, nil
	}
	toUpdate := upgrade.DeepCopy()

	reason, message := getPodWaitingStatus(pod)
	setHelmChartUpgradeStatus(toUpdate, v1.ConditionUnknown, reason, message)

	if !reflect.DeepEqual(upgrade, toUpdate) {
		if _, err := h.upgradeClient.Update(toUpdate); err != nil {
			return pod, err
		}
	}
	return pod, nil
}

func (h *podHandler) syncNodeUpgradePod(pod *v1.Pod, planName string, nodeName string) (*v1.Pod, error) {
	if pod.Status.Phase == v1.PodSucceeded {
		return pod, nil
	}
	plan, err := h.planCache.Get(upgradeNamespace, planName)
	if err != nil {
		return pod, err
	}
	upgradeName, ok := plan.Labels[harvesterUpgradeLabel]
	if !ok {
		return pod, nil
	}
	upgrade, err := h.upgradeCache.Get(h.namespace, upgradeName)
	if err != nil {
		return pod, err
	}
	toUpdate := upgrade.DeepCopy()

	reason, message := getPodWaitingStatus(pod)
	setNodeUpgradeStatus(toUpdate, nodeName, stateUpgrading, reason, message)
	if !reflect.DeepEqual(upgrade, toUpdate) {
		if _, err := h.upgradeClient.Update(toUpdate); err != nil {
			return pod, err
		}
	}

	return pod, nil
}

func getPodWaitingStatus(pod *v1.Pod) (reason string, message string) {
	var containerStatuses []v1.ContainerStatus
	containerStatuses = append(containerStatuses, pod.Status.InitContainerStatuses...)
	containerStatuses = append(containerStatuses, pod.Status.ContainerStatuses...)

	for _, status := range containerStatuses {
		if status.State.Waiting != nil && len(status.State.Waiting.Reason) > 0 && status.State.Waiting.Reason != "PodInitializing" {
			reason = status.State.Waiting.Reason
			message = status.State.Waiting.Message
			return
		}
	}
	return
}
