package main

import (
	"context"

	"github.com/moby/buildkit/frontend/gateway/grpcclient"
)

func main() {
	err := grpcclient.RunFromEnvironment(context.Background(), nil)
	if err != nil {
		panic(err)
	}
}
