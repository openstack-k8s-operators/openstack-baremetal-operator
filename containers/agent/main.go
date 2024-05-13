package main

import (
	"flag"

	"github.com/golang/glog"
	"github.com/spf13/cobra"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

const (
	componentName = "openstack-baremetal-agent"
)

var (
	rootCmd = &cobra.Command{
		Use:   componentName,
		Short: "Run OpenStack Agent",
		Long:  "Runs the OpenStack Baremetal Operator Agent",
	}

	openstackProvisionServerGVR = schema.GroupVersionResource{
		Group:    "baremetal.openstack.org",
		Version:  "v1beta1",
		Resource: "openstackprovisionservers",
	}
)

func init() {
	rootCmd.PersistentFlags().AddGoFlagSet(flag.CommandLine)
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		glog.Exitf("Error executing cmd: %v", err)
	}
}
