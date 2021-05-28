package user

import (
	"crypto/sha256"
	"encoding/base32"
	"fmt"
	"strings"

	"github.com/sirupsen/logrus"
	"k8s.io/apimachinery/pkg/runtime"

	"github.com/harvester/harvester/pkg/apis/harvesterhci.io"
	"github.com/harvester/harvester/pkg/apis/harvesterhci.io/v1beta1"
	ctlharvesterv1 "github.com/harvester/harvester/pkg/generated/controllers/harvesterhci.io/v1beta1"
	"github.com/harvester/harvester/pkg/indexeres"
	pkguser "github.com/harvester/harvester/pkg/user"
	werror "github.com/harvester/harvester/pkg/webhook/error"
	"github.com/harvester/harvester/pkg/webhook/types"
)

func NewMutator(users ctlharvesterv1.UserCache) types.Mutator {
	users.AddIndexer(indexeres.UserNameIndex, indexeres.IndexUserByUsername)
	return &userMutator{users: users}
}

type userMutator struct {
	types.DefaultMutator

	users ctlharvesterv1.UserCache
}

func (m *userMutator) Resource() types.Resource {
	return types.Resource{
		GroupName:  harvesterhci.GroupName,
		ObjectType: &v1beta1.User{},
	}
}

func (m *userMutator) Create(request *types.Request, newObj runtime.Object) (types.PatchOps, error) {
	logrus.Debug("entering userMutator.Create")
	user := newObj.(*v1beta1.User)
	return m.createOrUpdateUser(user, true)
}

func (m *userMutator) Update(request *types.Request, oldObj runtime.Object, newObj runtime.Object) (types.PatchOps, error) {
	logrus.Debug("entering userMutator.Update")
	user := newObj.(*v1beta1.User)
	return m.createOrUpdateUser(user, false)
}

func (m *userMutator) Delete(request *types.Request, oldObj runtime.Object) (types.PatchOps, error) {
	logrus.Debug("entering userMutator.Delete")
	user := oldObj.(*v1beta1.User)
	if user.Name == request.AdmissionRequest.UserInfo.Username {
		return nil, werror.NewInvalidError("can't delete self", "metadata.name")
	}
	return nil, nil
}

func (m *userMutator) createOrUpdateUser(user *v1beta1.User, create bool) (types.PatchOps, error) {
	if user.Username == "" {
		return nil, werror.NewInvalidError("username is required", "username")
	}

	if user.Password == "" {
		return nil, werror.NewInvalidError("password is required", "password")
	}

	if create {
		users, err := m.users.GetByIndex(indexeres.UserNameIndex, user.Username)
		if err != nil {
			return nil, werror.NewInternalError(err.Error())
		}
		if len(users) > 0 {
			return nil, werror.NewConflict("username is already in use")
		}
	}

	var patchOps types.PatchOps
	name := generateUserObjectName(user.Username)
	patchOps = append(patchOps, fmt.Sprintf(`{"op": "replace", "path": "/metadata/name", "value": "%s"}`, name))

	// FIXME: mutation webhook needs to be idempotent.
	// If the hook is called again, we'll hash a hash rather than plain text
	// Mabye we can add a hashedPassword field to fix this.
	// The mutator might be deprecated soon after moving the auth to Rancher.
	hashed, err := pkguser.HashPasswordString(user.Password)
	if err != nil {
		return nil, werror.NewInvalidError("Failed to encrypt password", "password")
	}
	patchOps = append(patchOps, fmt.Sprintf(`{"op": "replace", "path": "/password", "value": "%s"}`, hashed))

	return patchOps, nil
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
