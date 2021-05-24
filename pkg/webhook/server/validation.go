package server

import (
	"net/http"
	"reflect"

	"github.com/rancher/wrangler/pkg/webhook"
	"github.com/sirupsen/logrus"

	"github.com/harvester/harvester/pkg/webhook/clients"
	"github.com/harvester/harvester/pkg/webhook/config"
	"github.com/harvester/harvester/pkg/webhook/resources/datavolume"
	"github.com/harvester/harvester/pkg/webhook/resources/keypair"
	"github.com/harvester/harvester/pkg/webhook/resources/network"
	"github.com/harvester/harvester/pkg/webhook/resources/restore"
	"github.com/harvester/harvester/pkg/webhook/resources/upgrade"
	"github.com/harvester/harvester/pkg/webhook/resources/virtualmachineimage"
	"github.com/harvester/harvester/pkg/webhook/types"
)

func Validation(clients *clients.Clients, options config.Options) (http.Handler, error) {
	router := webhook.NewRouter()

	validators := []types.Validator{
		network.NewValidator(clients.CNIFactory.K8s().V1().NetworkAttachmentDefinition().Cache(), clients.KubevirtFactory.Kubevirt().V1().VirtualMachine().Cache()),
		datavolume.NewValidator(clients.CDIFactory.Cdi().V1beta1().DataVolume().Cache()),
		keypair.NewValidator(clients.HarvesterFactory.Harvesterhci().V1beta1().KeyPair().Cache()),
		virtualmachineimage.NewValidator(clients.HarvesterFactory.Harvesterhci().V1beta1().VirtualMachineImage().Cache()),
		upgrade.NewValidator(clients.HarvesterFactory.Harvesterhci().V1beta1().Upgrade().Cache()),
		restore.NewValidator(clients.KubevirtFactory.Kubevirt().V1().VirtualMachine().Cache()),
	}
	for _, v := range validators {
		info := v.Info()
		kind := reflect.Indirect(reflect.ValueOf(info.ObjectType)).Type().Name()
		logrus.Infof("register validator for kind %s, group %s", kind, info.GroupName)
		router.Kind(kind).Group(info.GroupName).Type(info.ObjectType).Handle(types.NewValidationHandler(v))
	}

	return router, nil
}
