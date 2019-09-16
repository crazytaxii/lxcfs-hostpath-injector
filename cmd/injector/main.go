package main

import (
	"os"

	"github.com/crazytaxii/lxcfs-sidecar-injector/cmd/injector/app"
	"k8s.io/klog"
)

func main() {
	cmd := app.NewMutateWebhookServer()
	if err := cmd.Execute(); err != nil {
		klog.Errorf("error: %v\n", err)
		os.Exit(1)
	}
}
