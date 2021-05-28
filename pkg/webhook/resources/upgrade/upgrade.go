package upgrade

import (
	"fmt"

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
	stateUpgrading    = "Upgrading"
	upgradeStateLabel = "harvesterhci.io/upgradeState"
)

func NewValidator(upgrades ctlharvesterv1.UpgradeCache) types.Validator {
	return &upgradeValidator{
		upgrades: upgrades,
	}
}

type upgradeValidator struct {
	types.DefaultValidator

	upgrades ctlharvesterv1.UpgradeCache
}

func (v *upgradeValidator) Resource() types.Resource {
	return types.Resource{
		GroupName:  harvesterhci.GroupName,
		ObjectType: &v1beta1.Upgrade{},
	}
}

func (v *upgradeValidator) Create(request *types.Request, newObj runtime.Object) error {
	logrus.Debug("entering upgradeValidator.Create")
	newUpgrade := newObj.(*v1beta1.Upgrade)

	sets := labels.Set{
		upgradeStateLabel: stateUpgrading,
	}
	upgrades, err := v.upgrades.List(newUpgrade.Namespace, sets.AsSelector())
	if err != nil {
		return werror.NewInternalError(err.Error())
	}
	if len(upgrades) > 0 {
		msg := fmt.Sprintf("cannot proceed until previous upgrade %q completes", upgrades[0].Name)
		return werror.NewConflict(msg)
	}

	return nil
}
