package main

import (
	"context"
	"flag"
	"fmt"
	"net"
	"os"
	"time"

	"github.com/golang/glog"
	"github.com/spf13/cobra"
	k8s_errors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

var (
	provisionIPStartCmd = &cobra.Command{
		Use:   "provision-ip-discovery",
		Short: "Start Provision IP Discovery Agent",
		Long:  "",
		Run:   runProvisionIPStartCmd,
	}

	provisionIPStartOpts struct {
		kubeconfig          string
		provIntf            string
		provServerName      string
		provServerNamespace string
	}
)

func init() {
	rootCmd.AddCommand(provisionIPStartCmd)
	provisionIPStartCmd.PersistentFlags().StringVar(&provisionIPStartOpts.provIntf, "prov-intf", "", "Provisioning interface name on the associated host")
	provisionIPStartCmd.PersistentFlags().StringVar(&provisionIPStartOpts.provServerName, "prov-server-name", "", "Provisioning server resource name")
	provisionIPStartCmd.PersistentFlags().StringVar(&provisionIPStartOpts.provServerNamespace, "prov-server-namespace", "", "Provisioning server resource namespace")
}

func runProvisionIPStartCmd(_ *cobra.Command, _ []string) {
	// Setup logging
	if err := flag.Set("logtostderr", "true"); err != nil {
		panic(err.Error())
	}
	flag.Parse()
	glog.V(0).Info("Starting ProvisionIpDiscoveryAgent")

	// Set required environment variables or fail gracefully
	provisionIPStartOpts.provIntf = getEnvOrFail("PROV_INTF", provisionIPStartOpts.provIntf)
	provisionIPStartOpts.provServerName = getEnvOrFail("PROV_SERVER_NAME", provisionIPStartOpts.provServerName)
	provisionIPStartOpts.provServerNamespace = getEnvOrFail("PROV_SERVER_NAMESPACE", provisionIPStartOpts.provServerNamespace)

	// Kubernetes client setup
	config, err := getKubeConfig()
	if err != nil {
		panic(err.Error())
	}

	dClient := dynamic.NewForConfigOrDie(config)
	provServerClient := dClient.Resource(openstackProvisionServerGVR)

	var ip string
	for {
		curIP, intfFound := getInterfaceIP(provisionIPStartOpts.provIntf)
		if curIP == "" || curIP != ip {
			err := updateProvisioningStatus(provServerClient, curIP, intfFound)
			if err != nil {
				glog.V(0).Infof("Error updating OpenStackProvisionServer %s (namespace %s) status: %s",
					provisionIPStartOpts.provServerName, provisionIPStartOpts.provServerNamespace, err)
				// Set ip to empty string as we've to update provisonserver in the next try
				ip = ""
			} else {
				ip = curIP
			}
		}
		time.Sleep(time.Second * 5)
	}
}

// getEnvOrFail retrieves an environment variable or returns a default value, failing if the value is empty.
func getEnvOrFail(envVar, defaultValue string) string {
	if defaultValue == "" {
		val, ok := os.LookupEnv(envVar)
		if !ok || val == "" {
			glog.Fatalf("%s is required", envVar)
		}
		return val
	}
	return defaultValue
}

// getKubeConfig returns the Kubernetes configuration, either from the provided KUBECONFIG environment variable or from the cluster.
func getKubeConfig() (*rest.Config, error) {
	kubeconfig := os.Getenv("KUBECONFIG")
	if kubeconfig != "" {
		return clientcmd.BuildConfigFromFlags("", kubeconfig)
	}
	return rest.InClusterConfig()
}

// getInterfaceIP returns the IP address for the given interface, or an empty string if not found.
func getInterfaceIP(interfaceName string) (string, bool) {
	ifaces, err := net.Interfaces()
	if err != nil {
		panic(err.Error())
	}

	intfFound := false

	for _, iface := range ifaces {
		if iface.Name == interfaceName {
			intfFound = true
			addrs, err := iface.Addrs()
			if err != nil {
				panic(err.Error())
			}

			for _, addr := range addrs {
				ipObj, _, err := net.ParseCIDR(addr.String())
				if err != nil || ipObj == nil {
					glog.V(0).Infof("WARNING: Cannot parse IP address for interface %s: %v", interfaceName, err)
					continue
				}

				if ipObj = ipObj.To4(); ipObj != nil {
					return ipObj.String(), intfFound
				}
				glog.V(0).Infof("INFO: Ignoring IPv6 address (%s) on interface %s", addr, interfaceName)
			}
		}
	}
	return "", intfFound
}

// updateProvisioningStatus updates the provisioning status in Kubernetes with the given IP and error status.
func updateProvisioningStatus(provServerClient dynamic.NamespaceableResourceInterface, curIP string, intfFound bool) error {
	unstructured, err := provServerClient.Namespace(provisionIPStartOpts.provServerNamespace).Get(context.Background(), provisionIPStartOpts.provServerName, metav1.GetOptions{}, "/status")
	if k8s_errors.IsNotFound(err) {
		// Server deleted, stop the loop
		return nil
	}
	if err != nil {
		return err
	}

	if unstructured.Object["status"] == nil {
		unstructured.Object["status"] = map[string]any{}
	}

	status := unstructured.Object["status"].(map[string]any)
	if curIP == "" {
		var errMsg, errMsgFull string
		if intfFound {
			errMsg = fmt.Sprintf("Unable to find provisioning IP on interface %s", provisionIPStartOpts.provIntf)
		} else {
			errMsg = fmt.Sprintf("Unable to find provisioning interface %s", provisionIPStartOpts.provIntf)
		}
		errMsgFull = fmt.Sprintf("%s for OpenStackProvisionServer %s (namespace %s)", errMsg, provisionIPStartOpts.provServerName, provisionIPStartOpts.provServerNamespace)
		glog.V(0).Infof("ERROR: %s", errMsgFull)

		status["provisionIp"] = ""
		status["provisionIpError"] = errMsg
	} else {
		status["provisionIp"] = curIP
		status["provisionIpError"] = ""
	}

	unstructured.Object["status"] = status
	_, err = provServerClient.Namespace(provisionIPStartOpts.provServerNamespace).UpdateStatus(context.Background(), unstructured, metav1.UpdateOptions{})
	return err
}
