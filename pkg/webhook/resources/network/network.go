package network

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/rancher/wrangler/pkg/webhook"
	"github.com/sirupsen/logrus"
	admissionv1 "k8s.io/api/admission/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"

	cniv1 "github.com/containernetworking/cni/pkg/types"
	v1 "github.com/k8snetworkplumbingwg/network-attachment-definition-client/pkg/apis/k8s.cni.cncf.io/v1"

	ctlcniv1 "github.com/harvester/harvester/pkg/generated/controllers/k8s.cni.cncf.io/v1"
	ctlkubevirtv1 "github.com/harvester/harvester/pkg/generated/controllers/kubevirt.io/v1"
	"github.com/harvester/harvester/pkg/indexeres"
	"github.com/harvester/harvester/pkg/webhook/utils"
)

const (
	fieldConfig     = "spec.config"
	fieldConfigVlan = "spec.config.vlan"
)

func NewValidator(netAttachDefs ctlcniv1.NetworkAttachmentDefinitionCache, vms ctlkubevirtv1.VirtualMachineCache) webhook.Handler {
	vms.AddIndexer(indexeres.VMByNetworkIndex, indexeres.VMByNetwork)
	return &networkAttachmentDefinitionValidator{
		netAttachDefs: netAttachDefs,
		vms:           vms,
	}
}

type networkAttachmentDefinitionValidator struct {
	netAttachDefs ctlcniv1.NetworkAttachmentDefinitionCache
	vms           ctlkubevirtv1.VirtualMachineCache
}

type NetConf struct {
	cniv1.NetConf
	BrName       string `json:"bridge"`
	IsGW         bool   `json:"isGateway"`
	IsDefaultGW  bool   `json:"isDefaultGateway"`
	ForceAddress bool   `json:"forceAddress"`
	IPMasq       bool   `json:"ipMasq"`
	MTU          int    `json:"mtu"`
	HairpinMode  bool   `json:"hairpinMode"`
	PromiscMode  bool   `json:"promiscMode"`
	Vlan         int    `json:"vlan"`
}

func (v *networkAttachmentDefinitionValidator) Admit(response *webhook.Response, request *webhook.Request) error {
	logrus.Debug("entering networkAttachmentDefinitionValidator.Admit")
	netAttachDef, err := netAttachDefObject(request)
	if err != nil {
		return utils.RejectInternalError(response, err.Error())
	}

	if request.Operation == admissionv1.Delete {
		return v.admitDelete(response, request, netAttachDef)
	}

	config := netAttachDef.Spec.Config
	if config == "" {
		return utils.RejectInvalid(response, "config is empty", fieldConfig)
	}

	var bridgeConf = &NetConf{}
	err = json.Unmarshal([]byte(config), &bridgeConf)
	if err != nil {
		message := fmt.Sprintf("failed to decode NAD config value, error: %s", err.Error())
		return utils.RejectInternalError(response, message)
	}

	if bridgeConf.Vlan < 1 || bridgeConf.Vlan > 4094 {
		return utils.RejectInvalid(response, "bridge VLAN ID must >=1 and <=4094", fieldConfigVlan)
	}

	allocated, err := v.getVLAN(netAttachDef.Namespace, bridgeConf.Vlan)
	if err != nil {
		return utils.RejectInternalError(response, err.Error())
	}
	if *allocated {
		message := fmt.Sprintf("VLAN ID %d is already allocated", bridgeConf.Vlan)
		return utils.RejectInvalid(response, message, fieldConfigVlan)
	}

	return utils.Allow(response)
}

// getVLAN checks if vid is already allocated to any network defs in a namespace
func (v *networkAttachmentDefinitionValidator) getVLAN(namespace string, vid int) (*bool, error) {
	allocated := false
	nads, err := v.netAttachDefs.List(namespace, labels.Everything())
	if err != nil {
		return nil, err
	}

	for _, nad := range nads {
		var bridgeConf = &NetConf{}
		err := json.Unmarshal([]byte(nad.Spec.Config), &bridgeConf)
		if err != nil {
			return nil, err
		}
		if bridgeConf.Vlan == vid {
			allocated = true
			break
		}
	}

	return &allocated, nil
}

func (v *networkAttachmentDefinitionValidator) admitDelete(response *webhook.Response, request *webhook.Request, netAttachDef *v1.NetworkAttachmentDefinition) error {
	networkName := netAttachDef.Name
	vms, err := v.vms.GetByIndex(indexeres.VMByNetworkIndex, networkName)
	if err != nil {
		return utils.RejectInternalError(response, err.Error())
	}

	if len(vms) > 0 {
		vmNameList := make([]string, 0, len(vms))
		for _, vm := range vms {
			vmNameList = append(vmNameList, vm.Name)
		}
		errorMessage := fmt.Sprintf("network %s is still used by vm(s): %s", networkName, strings.Join(vmNameList, ", "))
		return utils.RejectBadRequest(response, errorMessage)
	}

	return utils.Allow(response)
}

func netAttachDefObject(request *webhook.Request) (*v1.NetworkAttachmentDefinition, error) {
	var object runtime.Object
	var err error
	if request.Operation == admissionv1.Delete {
		object, err = request.DecodeOldObject()
	} else {
		object, err = request.DecodeObject()
	}
	if err != nil {
		return nil, err
	}
	return object.(*v1.NetworkAttachmentDefinition), nil
}
