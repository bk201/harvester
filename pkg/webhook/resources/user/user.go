package user

import (
	"crypto/sha256"
	"encoding/base32"
	"fmt"
	"strings"

	"github.com/rancher/wrangler/pkg/webhook"
	"github.com/sirupsen/logrus"
	admissionv1 "k8s.io/api/admission/v1"
	"k8s.io/apimachinery/pkg/runtime"

	"github.com/harvester/harvester/pkg/apis/harvesterhci.io/v1beta1"
	ctlharvesterv1 "github.com/harvester/harvester/pkg/generated/controllers/harvesterhci.io/v1beta1"
	"github.com/harvester/harvester/pkg/indexeres"
	pkguser "github.com/harvester/harvester/pkg/user"
	"github.com/harvester/harvester/pkg/webhook/utils"
)

func NewMutator(users ctlharvesterv1.UserCache) webhook.Handler {
	users.AddIndexer(indexeres.UserNameIndex, indexeres.IndexUserByUsername)
	return &userMutator{users: users}
}

type userMutator struct {
	users ctlharvesterv1.UserCache
}

func (m *userMutator) Admit(response *webhook.Response, request *webhook.Request) error {
	logrus.Debug("entering userMutator.Admit")
	if request.DryRun != nil && *request.DryRun {
		logrus.Debugf("dryRun userMutator.Admit")
		return utils.Allow(response)
	}

	user, err := userObject(request)
	if err != nil {
		return utils.RejectInternalError(response, err.Error())
	}

	if request.Operation == admissionv1.Delete {
		logrus.Infof("request.AdmissionRequest: %+v", request.AdmissionRequest)
		if user.Name == request.AdmissionRequest.UserInfo.Username {
			return utils.RejectInvalid(response, "can't delete self", "metadata.name")
		}
		return utils.Allow(response)
	}

	var jsonPatchOps []string
	if request.Operation == admissionv1.Create {
		if user.Username == "" {
			return utils.RejectInvalid(response, "username is required", "username")
		}

		users, err := m.users.GetByIndex(indexeres.UserNameIndex, user.Username)
		if err != nil {
			return utils.RejectInternalError(response, err.Error())
		}
		if len(users) > 0 {
			return utils.RejectConflict(response, "username is already in use")
		}

		name := generateUserObjectName(user.Username)
		op := fmt.Sprintf(`{"op": "replace", "path": "/metadata/name", "value": "%s"}`, name)
		jsonPatchOps = append(jsonPatchOps, op)
	}

	// FIXME: mutation webhook needs to be idempotent.
	// If the hook is called again, we'll hash a hash rather than plain text
	// Can we add a hashed password field to fix this?
	// TODO: check password empty
	if user.Password == "" {
		return utils.RejectInvalid(response, "password is required", "password")
	}

	hashed, err := pkguser.HashPasswordString(user.Password)
	if err != nil {
		return utils.RejectInvalid(response, "Failed to encrypt password", "password")
	}
	op := fmt.Sprintf(`{"op": "replace", "path": "/password", "value": "%s"}`, hashed)
	jsonPatchOps = append(jsonPatchOps, op)

	if len(jsonPatchOps) > 0 {
		patchType := admissionv1.PatchTypeJSONPatch
		patchData := fmt.Sprintf("[%s]", strings.Join(jsonPatchOps, ","))
		response.PatchType = &patchType
		response.Patch = []byte(patchData)
	}

	return utils.Allow(response)
}

func userObject(request *webhook.Request) (*v1beta1.User, error) {
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
	return object.(*v1beta1.User), nil
}

func generateUserObjectName(username string) string {
	// Create a hash of the userName to use as the name for the user,
	// this lets k8s tell us if there are duplicate users with the same name
	// thus avoiding a race.
	h := sha256.New()
	_, _ = h.Write([]byte(username))
	sha := base32.StdEncoding.WithPadding(-1).EncodeToString(h.Sum(nil))[:10]
	return fmt.Sprintf("u-" + strings.ToLower(sha))
}
