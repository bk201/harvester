package keypair

import (
	"github.com/rancher/wrangler/pkg/webhook"
	"github.com/sirupsen/logrus"
	"golang.org/x/crypto/ssh"

	"github.com/harvester/harvester/pkg/apis/harvesterhci.io/v1beta1"
	ctlharvesterv1 "github.com/harvester/harvester/pkg/generated/controllers/harvesterhci.io/v1beta1"
	"github.com/harvester/harvester/pkg/webhook/utils"
)

func NewValidator(keypairs ctlharvesterv1.KeyPairCache) webhook.Handler {
	return &keyPairValidator{
		keypairs: keypairs,
	}
}

type keyPairValidator struct {
	keypairs ctlharvesterv1.KeyPairCache
}

func (v *keyPairValidator) Admit(response *webhook.Response, request *webhook.Request) error {
	logrus.Debug("entering keyPairValidator.Admit")
	keypair, err := keyPairObject(request)
	if err != nil {
		return utils.RejectInternalError(response, err.Error())
	}

	publicKey := keypair.Spec.PublicKey
	if publicKey == "" {
		return utils.RejectInvalid(response, "public key is required", "spec.publicKey")
	}

	if _, _, _, _, err := ssh.ParseAuthorizedKey([]byte(publicKey)); err != nil {
		return utils.RejectInvalid(response, "key is not in valid OpenSSH public key format", "spec.publicKey")
	}

	return utils.Allow(response)
}

func keyPairObject(request *webhook.Request) (*v1beta1.KeyPair, error) {
	object, err := request.DecodeObject()
	if err != nil {
		return nil, err
	}
	return object.(*v1beta1.KeyPair), nil
}
