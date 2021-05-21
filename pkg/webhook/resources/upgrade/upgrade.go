package upgrade

import (
	"fmt"

	"github.com/rancher/wrangler/pkg/webhook"
	"github.com/sirupsen/logrus"
	"k8s.io/apimachinery/pkg/labels"

	"github.com/harvester/harvester/pkg/apis/harvesterhci.io/v1beta1"
	ctlharvesterv1 "github.com/harvester/harvester/pkg/generated/controllers/harvesterhci.io/v1beta1"
	"github.com/harvester/harvester/pkg/webhook/utils"
)

const (
	stateUpgrading    = "Upgrading"
	upgradeStateLabel = "harvesterhci.io/upgradeState"
)

func NewValidator(upgrades ctlharvesterv1.UpgradeCache) webhook.Handler {
	return &upgradeValidator{
		upgrades: upgrades,
	}
}

type upgradeValidator struct {
	upgrades ctlharvesterv1.UpgradeCache
}

func (v *upgradeValidator) Admit(response *webhook.Response, request *webhook.Request) error {
	logrus.Debug("entering upgradeValidator.Admit")
	newUpgrade, err := upgradeObject(request)
	if err != nil {
		return utils.RejectInternalError(response, err.Error())
	}

	sets := labels.Set{
		upgradeStateLabel: stateUpgrading,
	}
	upgrades, err := v.upgrades.List(newUpgrade.Namespace, sets.AsSelector())
	if err != nil {
		return utils.RejectInternalError(response, err.Error())
	}
	if len(upgrades) > 0 {
		msg := fmt.Sprintf("cannot proceed until previous upgrade %q completes", upgrades[0].Name)
		return utils.RejectConflict(response, msg)
	}

	return utils.Allow(response)
}

func upgradeObject(request *webhook.Request) (*v1beta1.Upgrade, error) {
	object, err := request.DecodeObject()
	if err != nil {
		return nil, err
	}
	return object.(*v1beta1.Upgrade), nil
}
