package user

import (
	admissionregv1 "k8s.io/api/admissionregistration/v1"
	"k8s.io/apimachinery/pkg/runtime"

	"github.com/harvester/harvester/pkg/apis/harvesterhci.io"
	"github.com/harvester/harvester/pkg/apis/harvesterhci.io/v1beta1"
	werror "github.com/harvester/harvester/pkg/webhook/error"
	"github.com/harvester/harvester/pkg/webhook/types"
)

func NewValidator() types.Validator {
	return &userValidator{}
}

type userValidator struct {
	types.DefaultValidator
}

func (m *userValidator) Resource() types.Resource {
	return types.Resource{
		Name:       v1beta1.UserResourceName,
		Scope:      admissionregv1.ClusterScope,
		APIGroup:   harvesterhci.GroupName,
		APIVersion: v1beta1.SchemeGroupVersion.Version,
		ObjectType: &v1beta1.User{},
		OperationTypes: []admissionregv1.OperationType{
			admissionregv1.Delete,
		},
	}
}

func (m *userValidator) Delete(request *types.Request, oldObj runtime.Object) error {
	user := oldObj.(*v1beta1.User)
	if user.Name == request.AdmissionRequest.UserInfo.Username {
		return werror.NewInvalidError("can't delete self", "metadata.name")
	}
	return nil
}
