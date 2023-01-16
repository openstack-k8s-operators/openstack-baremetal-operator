package main

import (
	"flag"
	"fmt"

	"github.com/spf13/cobra"
)

var (
	versionCmd = &cobra.Command{
		Use:   "version",
		Short: "Print the version number of OpenStack Baremetal Operator Agent",
		Long:  `All software has versions.  This is that of the OpenStack Baremetal Operator Agent.`,
		Run:   runVersionCmd,
	}
)

func init() {
	rootCmd.AddCommand(versionCmd)
}

func runVersionCmd(cmd *cobra.Command, args []string) {
	err := flag.Set("logtostderr", "true")
	if err != nil {
		panic(err.Error())
	}
	flag.Parse()

	program := "OpenStackBaremetalOperatorAgent"
	version := "v1.4.0"

	fmt.Println(program, version)
}
