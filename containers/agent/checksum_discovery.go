// Package main provides a checksum discovery agent for OpenStack baremetal provisioning
package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/golang/glog"
	metal3v1 "github.com/metal3-io/baremetal-operator/apis/metal3.io/v1alpha1"
	"github.com/spf13/cobra"
	k8s_errors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

var (
	checksumStartCmd = &cobra.Command{
		Use:   "checksum-discovery",
		Short: "Start Checksum Discovery Agent",
		Long:  "",
		Run:   runChecksumStartCmd,
	}

	checksumStartOpts struct {
		kubeconfig          string
		osImageDir          string
		provServerName      string
		provServerNamespace string
	}
)

func init() {
	rootCmd.AddCommand(checksumStartCmd)
	checksumStartCmd.PersistentFlags().StringVar(&checksumStartOpts.osImageDir, "os-image-dir", "", "OS image directory on the associated host")
	checksumStartCmd.PersistentFlags().StringVar(&checksumStartOpts.provServerName, "prov-server-name", "", "Provisioning server resource name")
	checksumStartCmd.PersistentFlags().StringVar(&checksumStartOpts.provServerNamespace, "prov-server-namespace", "", "Provisioning server resource namespace")
}

func runChecksumStartCmd(_ *cobra.Command, _ []string) {
	var err error
	err = flag.Set("logtostderr", "true")
	if err != nil {
		panic(err.Error())
	}

	flag.Parse()

	glog.V(0).Info("Starting ChecksumDiscoveryAgent")

	if checksumStartOpts.osImageDir == "" {
		dir, ok := os.LookupEnv("OS_IMAGE_DIR")
		if !ok || dir == "" {
			glog.Fatalf("os-image-dir is required")
		}
		checksumStartOpts.osImageDir = dir
	}

	if checksumStartOpts.provServerName == "" {
		name, ok := os.LookupEnv("PROV_SERVER_NAME")
		if !ok || name == "" {
			glog.Fatalf("prov-server-name is required")
		}
		checksumStartOpts.provServerName = name
	}

	if checksumStartOpts.provServerNamespace == "" {
		name, ok := os.LookupEnv("PROV_SERVER_NAMESPACE")
		if !ok || name == "" {
			glog.Fatalf("prov-server-namespace is required")
		}
		checksumStartOpts.provServerNamespace = name
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

	checksumFileName := ""
	var checksumType metal3v1.ChecksumType

	// First get the checksum data
	dir, err := os.Open(checksumStartOpts.osImageDir)

	if err != nil {
		panic(err.Error())
	}

	items, err := dir.Readdirnames(0)
	_ = dir.Close()

	if err != nil {
		panic(err.Error())
	}

	for _, item := range items {
		// Crude mechanism for detecting both the checksum file and its type
		if strings.Contains(item, "md5") {
			checksumFileName = item
			checksumType = metal3v1.MD5
			break
		} else if strings.Contains(item, "sha256") {
			checksumFileName = item
			checksumType = metal3v1.SHA256
			break
		} else if strings.Contains(item, "sha512") {
			checksumFileName = item
			checksumType = metal3v1.SHA512
			break
		}
	}

	if checksumFileName == "" {
		panic(fmt.Errorf("%w in %s", ErrOSImageNotFound, checksumStartOpts.osImageDir))
	}

	// Try to update status with checksum data until it succeeds, as it's possible to hit "object has been modified" k8s error here
	for {
		unstructured, err := provServerClient.Namespace(checksumStartOpts.provServerNamespace).Get(context.Background(), checksumStartOpts.provServerName, metav1.GetOptions{}, "/status")

		if k8s_errors.IsNotFound(err) {
			// Deleted somehow, so just break
			break
		}

		if err != nil {
			panic(err.Error())
		}

		if unstructured.Object["status"] == nil {
			unstructured.Object["status"] = map[string]any{}
		}

		status := unstructured.Object["status"].(map[string]any)

		status["osImageChecksumFilename"] = checksumFileName
		status["osImageChecksumType"] = checksumType

		unstructured.Object["status"] = status

		_, err = provServerClient.Namespace(checksumStartOpts.provServerNamespace).UpdateStatus(context.Background(), unstructured, metav1.UpdateOptions{})

		if err != nil {
			glog.V(0).Infof("Error updating OpenStackProvisionServer %s (namespace %s) \"osImageChecksumFilename\" and \"osImageChecksumType\" status: %s\n", checksumStartOpts.provServerName, checksumStartOpts.provServerNamespace, err)
		} else {
			glog.V(0).Infof("Updated OpenStackProvisionServer %s (namespace %s) with status \"osImageChecksumFilename\": %s and \"osImageChecksumType\": %s\n", checksumStartOpts.provServerName, checksumStartOpts.provServerNamespace, checksumFileName, checksumType)
			break
		}

		time.Sleep(time.Second * 1)
	}

	glog.V(0).Info("Shutting down ChecksumDiscoveryAgent")
}
