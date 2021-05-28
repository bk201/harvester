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

func Mutation(clients *clients.Clients, options *config.Options) (http.Handler, []types.Resource, error) {
	resources := []types.Resource{}
	mutators := []types.Mutator{
		user.NewMutator(clients.HarvesterFactory.Harvesterhci().V1beta1().User().Cache()),
		templateversion.NewMutator(
			clients.HarvesterFactory.Harvesterhci().V1beta1().VirtualMachineTemplate().Cache(),
			clients.HarvesterFactory.Harvesterhci().V1beta1().VirtualMachineTemplateVersion().Cache(),
			clients.HarvesterFactory.Harvesterhci().V1beta1().KeyPair().Cache()),
	}

	router := webhook.NewRouter()
	for _, m := range mutators {
		addHandler(router, m, options)
		resources = append(resources, m.Resource())
	}

	return router, resources, nil
}

func addHandler(router *webhook.Router, admitter types.Admitter, options *config.Options) {
	rsc := admitter.Resource()
	kind := reflect.Indirect(reflect.ValueOf(rsc.ObjectType)).Type().Name()
	router.Kind(kind).Group(rsc.APIGroup).Type(rsc.ObjectType).Handle(types.NewAdmissionHandler(admitter, options))
	logrus.Debugf("add handler for %s.%s (%s)", rsc.Name, rsc.APIGroup, kind)
}
