package server

import (
	"net/http"

	"github.com/rancher/wrangler/pkg/webhook"

	"github.com/harvester/harvester/pkg/apis/harvesterhci.io"
	"github.com/harvester/harvester/pkg/apis/harvesterhci.io/v1beta1"
	"github.com/harvester/harvester/pkg/webhook/clients"
	"github.com/harvester/harvester/pkg/webhook/config"
	"github.com/harvester/harvester/pkg/webhook/resources/templateversion"
	"github.com/harvester/harvester/pkg/webhook/resources/user"
)

func Mutation(clients *clients.Clients, options config.Options) (http.Handler, error) {
	userMutator := user.NewMutator(clients.HarvesterFactory.Harvesterhci().V1beta1().User().Cache())
	templateVersionMutator := templateversion.NewMutator(
		options.HarvesterControllerUsername,
		clients.HarvesterFactory.Harvesterhci().V1beta1().VirtualMachineTemplate().Cache(),
		clients.HarvesterFactory.Harvesterhci().V1beta1().VirtualMachineTemplateVersion().Cache(),
		clients.HarvesterFactory.Harvesterhci().V1beta1().KeyPair().Cache())
	router := webhook.NewRouter()
	router.Kind("User").Group(harvesterhci.GroupName).Type(&v1beta1.User{}).Handle(userMutator)
	router.Kind("VirtualMachineTemplateVersion").Group(harvesterhci.GroupName).Type(&v1beta1.VirtualMachineTemplateVersion{}).Handle(templateVersionMutator)
	return router, nil
}
