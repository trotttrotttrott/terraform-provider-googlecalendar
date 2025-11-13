package main

import (
	"context"
	"log"

	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/trotttrotttrott/terraform-provider-googlecalendar/googlecalendar"
)

func main() {
	err := providerserver.Serve(context.Background(), googlecalendar.New, providerserver.ServeOpts{
		Address: "registry.terraform.io/trotttrotttrott/googlecalendar",
	})
	if err != nil {
		log.Fatal(err)
	}
}
