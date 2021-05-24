package server

import (
	"net/http"
	"reflect"

	"github.com/rancher/wrangler/pkg/webhook"
	"github.com/sirupsen/logrus"

	"github.com/harvester/harvester/pkg/webhook/clients"
	"github.com/harvester/harvester/pkg/webhook/config"
	"github.com/harvester/harvester/pkg/webhook/resources/templateversion"
	"github.com/harvester/harvester/pkg/webhook/resources/user"
	"github.com/harvester/harvester/pkg/webhook/types"
)

func Mutation(clients *clients.Clients, options config.Options) (http.Handler, error) {
	router := webhook.NewRouter()

	mutators := []types.Mutator{
		user.NewMutator(clients.HarvesterFactory.Harvesterhci().V1beta1().User().Cache()),
		templateversion.NewMutator(
			options.HarvesterControllerUsername,
			clients.HarvesterFactory.Harvesterhci().V1beta1().VirtualMachineTemplate().Cache(),
			clients.HarvesterFactory.Harvesterhci().V1beta1().VirtualMachineTemplateVersion().Cache(),
			clients.HarvesterFactory.Harvesterhci().V1beta1().KeyPair().Cache()),
	}

	for _, m := range mutators {
		info := m.Info()
		kind := reflect.Indirect(reflect.ValueOf(info.ObjectType)).Type().Name()
		logrus.Infof("register mutator for kind %s, group %s", kind, info.GroupName)
		router.Kind(kind).Group(info.GroupName).Type(info.ObjectType).Handle(types.NewMutationHandler(m))
	}

	return router, nil
}
