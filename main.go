package main

import (
	"github.com/dashsoftaps/tf-custom-resources/dashsoftaws"
	"github.com/hashicorp/terraform/plugin"
)

func main() {
	plugin.Serve(&plugin.ServeOpts{
		ProviderFunc: dashsoftaws.Provider,
	})
}
