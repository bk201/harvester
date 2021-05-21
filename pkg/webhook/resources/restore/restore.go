package restore

import (
	"fmt"

	"github.com/rancher/wrangler/pkg/webhook"
	"github.com/sirupsen/logrus"
	apierrors "k8s.io/apimachinery/pkg/api/errors"

	"github.com/harvester/harvester/pkg/apis/harvesterhci.io/v1beta1"
	ctlkubevirtv1 "github.com/harvester/harvester/pkg/generated/controllers/kubevirt.io/v1"
	"github.com/harvester/harvester/pkg/webhook/utils"
)

func NewValidator(vms ctlkubevirtv1.VirtualMachineCache) webhook.Handler {
	return &restoreValidator{
		vms: vms,
	}
}

type restoreValidator struct {
	vms ctlkubevirtv1.VirtualMachineCache
}

func (v *restoreValidator) Admit(response *webhook.Response, request *webhook.Request) error {
	logrus.Debug("entering restoreValidator.Admit")
	newRestore, err := restoreObject(request)
	if err != nil {
		return utils.RejectInternalError(response, err.Error())
	}

	targetVM := newRestore.Spec.Target.Name
	backupName := newRestore.Spec.VirtualMachineBackupName
	newVM := newRestore.Spec.NewVM

	if targetVM == "" {
		return utils.RejectInvalid(response, "taget VM name is empty", "spec.target.name")
	}
	if backupName == "" {
		return utils.RejectInvalid(response, "backup name is empty", "spec.virtualMachineBackupName")
	}

	vm, err := v.vms.Get(newRestore.Namespace, targetVM)
	if err != nil {
		if newVM && apierrors.IsNotFound(err) {
			return utils.Allow(response)
		}
		return utils.RejectInvalid(response, err.Error(), "spec.target.name")
	}

	// restore a new vm but the vm is already exist
	if newVM && vm != nil {
		return utils.RejectInvalid(response, fmt.Sprintf("VM %s is already exists", vm.Name), "spec.newVM")
	}

	// restore an existing vm but the vm is still running
	if !newVM && vm.Status.Ready {
		return utils.RejectInvalid(response, fmt.Sprintf("please stop the VM %q before doing a restore", vm.Name), "spec.target.name")
	}

	return utils.Allow(response)
}

func restoreObject(request *webhook.Request) (*v1beta1.VirtualMachineRestore, error) {
	object, err := request.DecodeObject()
	if err != nil {
		return nil, err
	}
	return object.(*v1beta1.VirtualMachineRestore), nil
}
