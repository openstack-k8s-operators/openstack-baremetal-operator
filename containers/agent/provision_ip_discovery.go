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
	var err error
	err = flag.Set("logtostderr", "true")
	if err != nil {
		panic(err.Error())
	}

	flag.Parse()

	glog.V(0).Info("Starting ProvisionIpDiscoveryAgent")

	if provisionIPStartOpts.provIntf == "" {
		name, ok := os.LookupEnv("PROV_INTF")
		if !ok || name == "" {
			glog.Fatalf("prov-intf is required")
		}
		provisionIPStartOpts.provIntf = name
	}

	if provisionIPStartOpts.provServerName == "" {
		name, ok := os.LookupEnv("PROV_SERVER_NAME")
		if !ok || name == "" {
			glog.Fatalf("prov-server-name is required")
		}
		provisionIPStartOpts.provServerName = name
	}

	if provisionIPStartOpts.provServerNamespace == "" {
		name, ok := os.LookupEnv("PROV_SERVER_NAMESPACE")
		if !ok || name == "" {
			glog.Fatalf("prov-server-namespace is required")
		}
		provisionIPStartOpts.provServerNamespace = name
	}

	var config *rest.Config
	kubeconfig := os.Getenv("KUBECONFIG")
	if kubeconfig != "" {
		config, err = clientcmd.BuildConfigFromFlags("", kubeconfig)
	} else {
		// creates the in-cluster config
		config, err = rest.InClusterConfig()
	}

	if err != nil {
		panic(err.Error())
	}

	dClient := dynamic.NewForConfigOrDie(config)

	provServerClient := dClient.Resource(openstackProvisionServerGVR)

	ip := ""

	// Get provision interface IP and update the status, and then sleep 5 seconds
	// and check again over and over (because the IP address could change)
	for {
		ifaces, err := net.Interfaces()

		if err != nil {
			panic(err.Error())
		}

		curIP := ""
		intfFound := false

		for _, iface := range ifaces {
			if iface.Name == provisionIPStartOpts.provIntf {
				intfFound = true

				addrs, err := iface.Addrs()

				if err != nil {
					panic(err.Error())
				}

				for _, addr := range addrs {
					ipObj, _, err := net.ParseCIDR(addr.String())

					if err != nil || ipObj == nil {
						glog.V(0).Infof("WARNING: Cannot parse IP address for OpenStackProvisionServer %s (namespace %s) on interface %s!\n", provisionIPStartOpts.provServerName, provisionIPStartOpts.provServerName, provisionIPStartOpts.provIntf)
						if err != nil {
							glog.V(0).Infof("ERROR: %s", err.Error())
						}
						continue
					}

					if ipObj = ipObj.To4(); ipObj != nil {
						curIP = ipObj.String()
						break
					}
					glog.V(0).Infof("INFO: Ignoring IPv6 address (%s) for OpenStackProvisionServer %s (namespace %s) on interface %s!\n", addr, provisionIPStartOpts.provServerName, provisionIPStartOpts.provServerName, provisionIPStartOpts.provIntf)
				}
				break
			}
		}

		if curIP == "" || ip != curIP {
			unstructured, err := provServerClient.Namespace(provisionIPStartOpts.provServerNamespace).Get(context.Background(), provisionIPStartOpts.provServerName, metav1.GetOptions{}, "/status")

			if k8s_errors.IsNotFound(err) {
				// Deleted somehow, so just break
				break
			}

			if err != nil {
				panic(err.Error())
			}

			if unstructured.Object["status"] == nil {
				unstructured.Object["status"] = map[string]interface{}{}
			}

			status := unstructured.Object["status"].(map[string]interface{})

			if curIP == "" {
				var errMsg string     // shorter message intended for surfacing in OpenStackProvisionServer CR
				var errMsgFull string // longer message for provisiong agent pod logs

				if intfFound {
					// Missing IP
					errMsg = fmt.Sprintf("Unable to find provisioning IP on interface %s", provisionIPStartOpts.provIntf)
					errMsgFull = fmt.Sprintf("%s for OpenStackProvisionServer %s (namespace %s)\n", errMsg, provisionIPStartOpts.provServerName, provisionIPStartOpts.provServerNamespace)
				} else {
					// Missing interface entirely
					errMsg = fmt.Sprintf("Unable to find provisioning interface %s", provisionIPStartOpts.provIntf)
					errMsgFull = fmt.Sprintf("%s for OpenStackProvisionServer %s (namespace %s)\n", errMsg, provisionIPStartOpts.provServerName, provisionIPStartOpts.provServerNamespace)
				}

				glog.V(0).Infof("ERROR: %s", errMsgFull)

				status["provisionIp"] = ""
				status["provisionIpError"] = errMsg

				unstructured.Object["status"] = status
				ip = curIP

				_, err = provServerClient.Namespace(provisionIPStartOpts.provServerNamespace).UpdateStatus(context.Background(), unstructured, metav1.UpdateOptions{})

				if err != nil {
					glog.V(0).Infof("Error updating OpenStackProvisionServer %s (namespace %s) status with provisioning IP acquisition error: %s\n",
						provisionIPStartOpts.provServerName, provisionIPStartOpts.provServerNamespace,
						err)
				} else {
					glog.V(0).Infof("Updated OpenStackProvisionServer %s (namespace %s) status with provisioning IP acquisition error\n",
						provisionIPStartOpts.provServerName,
						provisionIPStartOpts.provServerNamespace)

				}
			} else {
				// ip != curIP case
				status["provisionIp"] = curIP
				status["provisionIpError"] = ""

				unstructured.Object["status"] = status

				_, err = provServerClient.Namespace(provisionIPStartOpts.provServerNamespace).UpdateStatus(context.Background(), unstructured, metav1.UpdateOptions{})

				if err != nil {
					// As the provision server status could not be updated, we need to reset this
					ip = ""
					glog.V(0).Infof("Error updating OpenStackProvisionServer %s (namespace %s) \"provisionIp\" status: %s\n", provisionIPStartOpts.provServerName, provisionIPStartOpts.provServerNamespace, err)
				} else {
					ip = curIP
					glog.V(0).Infof("Updated OpenStackProvisionServer %s (namespace %s) with status \"provisionIp\": %s\n", provisionIPStartOpts.provServerName, provisionIPStartOpts.provServerNamespace, ip)

				}
			}
		}

		time.Sleep(time.Second * 5)
	}

	glog.V(0).Info("Shutting down ProvisionIpDiscoveryAgent")
}
