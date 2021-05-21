package templateversion

import (
	"fmt"

	"github.com/rancher/wrangler/pkg/webhook"
	"github.com/sirupsen/logrus"
	admissionv1 "k8s.io/api/admission/v1"
	"k8s.io/apimachinery/pkg/runtime"

	"github.com/harvester/harvester/pkg/apis/harvesterhci.io/v1beta1"
	ctlharvesterv1 "github.com/harvester/harvester/pkg/generated/controllers/harvesterhci.io/v1beta1"
	"github.com/harvester/harvester/pkg/ref"
	"github.com/harvester/harvester/pkg/webhook/utils"
)

const (
	fieldTemplateID = "spec.templateId"
	fieldKeyPairIds = "spec.keyPairIds"
)

func NewMutator(harvesterControllerUsername string, templateCache ctlharvesterv1.VirtualMachineTemplateCache, templateVersionCache ctlharvesterv1.VirtualMachineTemplateVersionCache, keypairs ctlharvesterv1.KeyPairCache) webhook.Handler {
	return &templateVersionMutator{
		ctlUsername:          harvesterControllerUsername,
		templateCache:        templateCache,
		templateVersionCache: templateVersionCache,
		keypairs:             keypairs,
	}
}

type templateVersionMutator struct {
	ctlUsername          string
	templateCache        ctlharvesterv1.VirtualMachineTemplateCache
	templateVersionCache ctlharvesterv1.VirtualMachineTemplateVersionCache
	keypairs             ctlharvesterv1.KeyPairCache
}

func (m *templateVersionMutator) Admit(response *webhook.Response, request *webhook.Request) error {
	logrus.Debugf("entering templateVersionMutator.Admit")
	if request.DryRun != nil && *request.DryRun {
		logrus.Debugf("dryRun templateVersionMutator.Admit")
		return utils.Allow(response)
	}

	vmTemplate, err := virtualMachineTemplateObject(request)
	if err != nil {
		return utils.RejectInternalError(response, err.Error())
	}

	if request.Operation == admissionv1.Delete {
		return m.admitDelete(response, request, vmTemplate)
	}

	if request.Operation == admissionv1.Update && request.UserInfo.Username != m.ctlUsername {
		// deny update requests except those sent from the harvester controller
		return utils.RejectMethodNotAllowed(response, "Update templateVersion is not supported")
	}

	templateID := vmTemplate.Spec.TemplateID
	if templateID == "" {
		return utils.RejectInvalid(response, "TemplateId is empty", fieldTemplateID)
	}

	templateNs, templateName := ref.Parse(templateID)
	if vmTemplate.Namespace != templateNs {
		return utils.RejectInvalid(response, "Template version and template should reside in the same namespace", "metadata.namespace")
	}

	if _, err := m.templateCache.Get(templateNs, templateName); err != nil {
		return utils.RejectInvalid(response, err.Error(), fieldTemplateID)
	}

	keyPairIDs := vmTemplate.Spec.KeyPairIDs
	if len(keyPairIDs) > 0 {
		for i, v := range keyPairIDs {
			keyPairNs, keyPairName := ref.Parse(v)
			_, err := m.keypairs.Get(keyPairNs, keyPairName)
			if err != nil {
				message := fmt.Sprintf("KeyPairID %s is invalid, %v", v, err)
				field := fmt.Sprintf("%s[%d]", fieldKeyPairIds, i)
				return utils.RejectInvalid(response, message, field)
			}
		}
	}

	// patch "metadata.generateName" with "{templateName}-"
	patchData := fmt.Sprintf(`[{"op": "replace", "path": "/metadata/generateName", "value": "%s"}]`, templateName+"-")
	patchType := admissionv1.PatchTypeJSONPatch
	response.PatchType = &patchType
	response.Patch = []byte(patchData)

	return utils.Allow(response)
}

func (m *templateVersionMutator) admitDelete(response *webhook.Response, request *webhook.Request, vmTemplateVersion *v1beta1.VirtualMachineTemplateVersion) error {
	version, err := m.templateVersionCache.Get(vmTemplateVersion.Namespace, vmTemplateVersion.Name)
	if err != nil {
		return utils.RejectInternalError(response, err.Error())
	}

	templNs, templName := ref.Parse(version.Spec.TemplateID)
	vt, err := m.templateCache.Get(templNs, templName)
	if err != nil {
		return utils.RejectInternalError(response, err.Error())
	}

	vresionID := ref.Construct(vmTemplateVersion.Namespace, vmTemplateVersion.Name)
	if vt.Spec.DefaultVersionID == vresionID {
		return utils.RejectBadRequest(response, "Cannot delete the default templateVersion")
	}

	return utils.Allow(response)
}

func virtualMachineTemplateObject(request *webhook.Request) (*v1beta1.VirtualMachineTemplateVersion, error) {
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
	return object.(*v1beta1.VirtualMachineTemplateVersion), nil
}
