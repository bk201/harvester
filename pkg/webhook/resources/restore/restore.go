package restore

import (
	"fmt"

	"github.com/sirupsen/logrus"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"

	"github.com/harvester/harvester/pkg/apis/harvesterhci.io"
	"github.com/harvester/harvester/pkg/apis/harvesterhci.io/v1beta1"
	ctlkubevirtv1 "github.com/harvester/harvester/pkg/generated/controllers/kubevirt.io/v1"
	werror "github.com/harvester/harvester/pkg/webhook/error"
	"github.com/harvester/harvester/pkg/webhook/types"
)

const (
	fieldTargetName               = "spec.target.name"
	fieldVirtualMachineBackupName = "spec.virtualMachineBackupName"
	fieldNewVM                    = "spec.newVM"
)

func NewValidator(vms ctlkubevirtv1.VirtualMachineCache) types.Validator {
	return &restoreValidator{
		vms: vms,
	}
}

type restoreValidator struct {
	types.DefaultValidator

	vms ctlkubevirtv1.VirtualMachineCache
}

func (v *restoreValidator) Resource() types.Resource {
	return types.Resource{
		GroupName:  harvesterhci.GroupName,
		ObjectType: &v1beta1.VirtualMachineRestore{},
	}
}

func (v *restoreValidator) Create(request *types.Request, newObj runtime.Object) error {
	logrus.Debug("entering restoreValidator.Create")
	newRestore := newObj.(*v1beta1.VirtualMachineRestore)

	targetVM := newRestore.Spec.Target.Name
	backupName := newRestore.Spec.VirtualMachineBackupName
	newVM := newRestore.Spec.NewVM

	if targetVM == "" {
		return werror.NewInvalidError("taget VM name is empty", fieldTargetName)
	}
	if backupName == "" {
		return werror.NewInvalidError("backup name is empty", fieldVirtualMachineBackupName)
	}

	vm, err := v.vms.Get(newRestore.Namespace, targetVM)
	if err != nil {
		if newVM && apierrors.IsNotFound(err) {
			return nil
		}
		return werror.NewInvalidError(err.Error(), fieldTargetName)
	}

	// restore a new vm but the vm is already exist
	if newVM && vm != nil {
		return werror.NewInvalidError(fmt.Sprintf("VM %s is already exists", vm.Name), fieldNewVM)
	}

	// restore an existing vm but the vm is still running
	if !newVM && vm.Status.Ready {
		return werror.NewInvalidError(fmt.Sprintf("please stop the VM %q before doing a restore", vm.Name), fieldTargetName)
	}

	return nil
}
