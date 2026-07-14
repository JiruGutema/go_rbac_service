package server

import (
	"fmt"

	"github.com/jirugutema/go_rbac_service/rbac_service/cmd/api"
)

func Server(config string) {
	api.APi()
	fmt.Println("Server finished booting.")
}
