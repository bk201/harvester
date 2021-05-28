package keypair

import (
	"errors"

	"github.com/rancher/wrangler/pkg/webhook"
	"github.com/sirupsen/logrus"
	"golang.org/x/crypto/ssh"
	"k8s.io/apimachinery/pkg/runtime"

	"github.com/harvester/harvester/pkg/apis/harvesterhci.io"
	"github.com/harvester/harvester/pkg/apis/harvesterhci.io/v1beta1"
	ctlharvesterv1 "github.com/harvester/harvester/pkg/generated/controllers/harvesterhci.io/v1beta1"
	werror "github.com/harvester/harvester/pkg/webhook/error"
	"github.com/harvester/harvester/pkg/webhook/types"
)

const (
	fieldPublicKey = "spec.publicKey"
)

func NewValidator(keypairs ctlharvesterv1.KeyPairCache) types.Validator {
	return &keyPairValidator{
		keypairs: keypairs,
	}
}

type keyPairValidator struct {
	types.DefaultValidator
	keypairs ctlharvesterv1.KeyPairCache
}

func (v *keyPairValidator) Info() types.ValidatorInfo {
	return types.ValidatorInfo{
		GroupName:  harvesterhci.GroupName,
		ObjectType: &v1beta1.KeyPair{},
	}
}

func (v *keyPairValidator) Create(request *webhook.Request, newObj runtime.Object) error {
	logrus.Debug("entering keyPairValidator.create")
	keypair := newObj.(*v1beta1.KeyPair)

	if err := v.checkPublicKey(keypair.Spec.PublicKey); err != nil {
		return werror.NewInvalidError(err.Error(), fieldPublicKey)
	}
	return nil
}

func (v *keyPairValidator) Update(request *webhook.Request, oldObj runtime.Object, newObj runtime.Object) error {
	logrus.Debug("entering keyPairValidator.update")
	keypair := newObj.(*v1beta1.KeyPair)

	if err := v.checkPublicKey(keypair.Spec.PublicKey); err != nil {
		return werror.NewInvalidError(err.Error(), fieldPublicKey)
	}
	return nil
}

func (v *keyPairValidator) checkPublicKey(publicKey string) error {
	if publicKey == "" {
		return errors.New("public key is required")
	}

	if _, _, _, _, err := ssh.ParseAuthorizedKey([]byte(publicKey)); err != nil {
		return errors.New("key is not in valid OpenSSH public key format")
	}

	return nil
}
