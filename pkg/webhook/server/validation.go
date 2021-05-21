package server

import (
	"net/http"

	cni "github.com/k8snetworkplumbingwg/network-attachment-definition-client/pkg/apis/k8s.cni.cncf.io"
	cniv1 "github.com/k8snetworkplumbingwg/network-attachment-definition-client/pkg/apis/k8s.cni.cncf.io/v1"
	"github.com/rancher/wrangler/pkg/webhook"
	cdicore "kubevirt.io/containerized-data-importer/pkg/apis/core"
	cdiv1 "kubevirt.io/containerized-data-importer/pkg/apis/core/v1beta1"

	"github.com/harvester/harvester/pkg/apis/harvesterhci.io"
	"github.com/harvester/harvester/pkg/apis/harvesterhci.io/v1beta1"
	"github.com/harvester/harvester/pkg/webhook/clients"
	"github.com/harvester/harvester/pkg/webhook/config"
	"github.com/harvester/harvester/pkg/webhook/resources/datavolume"
	"github.com/harvester/harvester/pkg/webhook/resources/keypair"
	"github.com/harvester/harvester/pkg/webhook/resources/network"
	"github.com/harvester/harvester/pkg/webhook/resources/restore"
	"github.com/harvester/harvester/pkg/webhook/resources/upgrade"
	"github.com/harvester/harvester/pkg/webhook/resources/virtualmachineimage"
)

func Validation(clients *clients.Clients, options config.Options) (http.Handler, error) {
	dataVolumeValidator := datavolume.NewValidator(clients.CDIFactory.Cdi().V1beta1().DataVolume().Cache())
	vmimagesValidator := virtualmachineimage.NewValidator(clients.HarvesterFactory.Harvesterhci().V1beta1().VirtualMachineImage().Cache())
	keypairValidator := keypair.NewValidator(clients.HarvesterFactory.Harvesterhci().V1beta1().KeyPair().Cache())
	networkValidator := network.NewValidator(clients.CNIFactory.K8s().V1().NetworkAttachmentDefinition().Cache(), clients.KubevirtFactory.Kubevirt().V1().VirtualMachine().Cache())
	upgradeValidator := upgrade.NewValidator(clients.HarvesterFactory.Harvesterhci().V1beta1().Upgrade().Cache())
	restoreValidator := restore.NewValidator(clients.KubevirtFactory.Kubevirt().V1().VirtualMachine().Cache())
	router := webhook.NewRouter()
	router.Kind("DataVolume").Group(cdicore.GroupName).Type(&cdiv1.DataVolume{}).Handle(dataVolumeValidator)
	router.Kind("VirtualMachineImage").Group(harvesterhci.GroupName).Type(&v1beta1.VirtualMachineImage{}).Handle(vmimagesValidator)
	router.Kind("KeyPair").Group(harvesterhci.GroupName).Type(&v1beta1.KeyPair{}).Handle(keypairValidator)
	router.Kind("NetworkAttachmentDefinition").Group(cni.GroupName).Type(&cniv1.NetworkAttachmentDefinition{}).Handle(networkValidator)
	router.Kind("Upgrade").Group(harvesterhci.GroupName).Type(&v1beta1.Upgrade{}).Handle(upgradeValidator)
	router.Kind("VirtualMachineRestore").Group(harvesterhci.GroupName).Type(&v1beta1.VirtualMachineRestore{}).Handle(restoreValidator)
	return router, nil
}
