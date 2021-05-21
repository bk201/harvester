package user

import (
	"github.com/rancher/steve/pkg/schema"
	"github.com/rancher/steve/pkg/server"
	"github.com/rancher/steve/pkg/stores/proxy"

	"github.com/harvester/harvester/pkg/config"
)

const (
	userSchemaID = "harvesterhci.io.user"
)

func RegisterSchema(scaled *config.Scaled, server *server.Server, options config.Options) error {
	t := schema.Template{
		ID:        userSchemaID,
		Store:     proxy.NewProxyStore(server.ClientFactory, nil, server.AccessSetLookup),
		Formatter: formatter,
	}

	server.SchemaFactory.AddTemplate(t)
	return nil
}
