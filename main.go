//go:generate go run pkg/codegen/cleanup/main.go
//go:generate /bin/rm -rf pkg/generated
//go:generate go run pkg/codegen/main.go
//go:generate /bin/bash scripts/generate-manifest
//go:generate /bin/bash scripts/generate-openapi

package main

import (
	_ "net/http/pprof"

	"github.com/harvester/harvester/pkg/cmd"
)

func main() {
	cmd.Execute()
}
