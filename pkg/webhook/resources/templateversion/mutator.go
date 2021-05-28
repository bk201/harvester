package templateversion

import (
	"fmt"

	"github.com/sirupsen/logrus"
	admissionregv1 "k8s.io/api/admissionregistration/v1"
	"k8s.io/apimachinery/pkg/runtime"

	"github.com/harvester/harvester/pkg/apis/harvesterhci.io/v1beta1"
	ctlharvesterv1 "github.com/harvester/harvester/pkg/generated/controllers/harvesterhci.io/v1beta1"
	"github.com/harvester/harvester/pkg/ref"
	werror "github.com/harvester/harvester/pkg/webhook/error"
	"github.com/harvester/harvester/pkg/webhook/types"
)

const (
	fieldTemplateID = "spec.templateId"
	fieldKeyPairIds = "spec.keyPairIds"
)

func NewMutator(templateCache ctlharvesterv1.VirtualMachineTemplateCache, templateVersionCache ctlharvesterv1.VirtualMachineTemplateVersionCache, keypairs ctlharvesterv1.KeyPairCache) types.Mutator {
	return &templateVersionMutator{
		templateCache:        templateCache,
		templateVersionCache: templateVersionCache,
		keypairs:             keypairs,
	}
}

type templateVersionMutator struct {
	types.DefaultMutator

	templateCache        ctlharvesterv1.VirtualMachineTemplateCache
	templateVersionCache ctlharvesterv1.VirtualMachineTemplateVersionCache
	keypairs             ctlharvesterv1.KeyPairCache
}

func (m *templateVersionMutator) Resource() types.Resource {
	return types.Resource{
		Name:       v1beta1.VirtualMachineTemplateVersionResourceName,
		Scope:      admissionregv1.NamespacedScope,
		APIGroup:   v1beta1.SchemeGroupVersion.Group,
		APIVersion: v1beta1.SchemeGroupVersion.Version,
		ObjectType: &v1beta1.VirtualMachineTemplateVersion{},
		OperationTypes: []admissionregv1.OperationType{
			admissionregv1.Create,
			admissionregv1.Update,
			admissionregv1.Delete,
		},
	}
}

func (m *templateVersionMutator) Create(request *types.Request, newObj runtime.Object) (types.PatchOps, error) {
	vmTemplVersion := newObj.(*v1beta1.VirtualMachineTemplateVersion)

	templateID := vmTemplVersion.Spec.TemplateID
	if templateID == "" {
		return nil, werror.NewInvalidError("TemplateId is empty", fieldTemplateID)
	}

	templateNs, templateName := ref.Parse(templateID)
	if vmTemplVersion.Namespace != templateNs {
		return nil, werror.NewInvalidError("Template version and template should reside in the same namespace", "metadata.namespace")
	}

	if _, err := m.templateCache.Get(templateNs, templateName); err != nil {
		return nil, werror.NewInvalidError(err.Error(), fieldTemplateID)
	}

	keyPairIDs := vmTemplVersion.Spec.KeyPairIDs
	if len(keyPairIDs) > 0 {
		for i, v := range keyPairIDs {
			keyPairNs, keyPairName := ref.Parse(v)
			_, err := m.keypairs.Get(keyPairNs, keyPairName)
			if err != nil {
				message := fmt.Sprintf("KeyPairID %s is invalid, %v", v, err)
				field := fmt.Sprintf("%s[%d]", fieldKeyPairIds, i)
				return nil, werror.NewInvalidError(message, field)
			}
		}
	}

	// Do not generate a name if there is a name.
	if vmTemplVersion.Name != "" {
		return nil, nil
	}

	// patch "metadata.generateName" with "{templateName}-"
	var patchOps types.PatchOps
	patchOps = append(patchOps, fmt.Sprintf(`{"op": "replace", "path": "/metadata/generateName", "value": "%s"}`, templateName+"-"))
	return patchOps, nil
}

func (m *templateVersionMutator) Update(request *types.Request, oldObj runtime.Object, newObj runtime.Object) (types.PatchOps, error) {
	if request.IsFromController() {
		return nil, nil
	}
	logrus.Infof("not allow for user %s", request.UserInfo.Username)
	return nil, werror.NewMethodNotAllowed("Update templateVersion is not supported")
}

func (m *templateVersionMutator) Delete(request *types.Request, oldObj runtime.Object) (types.PatchOps, error) {
	// If a template is deleted, its versions are garbage collected.
	// No need to check for template existence or if a version is the default version or not.
	if request.IsGarbageCollection() {
		return nil, nil
	}
	vmTemplVersion := oldObj.(*v1beta1.VirtualMachineTemplateVersion)
	version, err := m.templateVersionCache.Get(vmTemplVersion.Namespace, vmTemplVersion.Name)
	if err != nil {
		return nil, err
	}

	templNs, templName := ref.Parse(version.Spec.TemplateID)
	vt, err := m.templateCache.Get(templNs, templName)
	if err != nil {
		return nil, err
	}

	vresionID := ref.Construct(vmTemplVersion.Namespace, vmTemplVersion.Name)
	if vt.Spec.DefaultVersionID == vresionID {
		return nil, werror.NewBadRequest("Cannot delete the default templateVersion")
	}

	return nil, nil
}
