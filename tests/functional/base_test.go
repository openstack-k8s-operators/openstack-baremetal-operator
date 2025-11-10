/*
Copyright 2023.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

	http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/
package functional

import (
	"strings"

	metal3v1 "github.com/metal3-io/baremetal-operator/apis/metal3.io/v1alpha1"
	. "github.com/onsi/gomega" //revive:disable:dot-imports
	"github.com/openstack-k8s-operators/lib-common/modules/common/condition"
	"github.com/openstack-k8s-operators/lib-common/modules/common/util"
	baremetalv1 "github.com/openstack-k8s-operators/openstack-baremetal-operator/api/v1beta1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/types"
)

// Create OpenstackBaremetalSet in k8s and test that no errors occur
func CreateBaremetalSet(name types.NamespacedName, spec map[string]any) *unstructured.Unstructured {
	instance := DefaultBaremetalSetTemplate(name, spec)
	return th.CreateUnstructured(instance)
}

// Build OpenStackBaremetalSet struct and fill it with preset values
func DefaultBaremetalSetTemplate(name types.NamespacedName, spec map[string]any) map[string]any {
	return map[string]any{

		"apiVersion": "baremetal.openstack.org/v1beta1",
		"kind":       "OpenStackBaremetalSet",
		"metadata": map[string]any{
			"name":      name.Name,
			"namespace": name.Namespace,
		},
		"spec": spec,
	}
}

// Build BaremetalSetSpec struct and fill it with preset values
func DefaultBaremetalSetSpec(name types.NamespacedName, withProvInterface bool) map[string]any {
	spec := map[string]any{
		"baremetalHosts": map[string]any{
			"compute-0": map[string]any{
				"ctlPlaneIP": "10.0.0.1/24",
			},
		},
		"bmhLabelSelector":      map[string]string{"app": "openstack"},
		"deploymentSSHSecret":   "mysecret",
		"ctlplaneInterface":     "eth0",
		"bmhNamespace":          name.Namespace,
		"osImageDeploymentType": "SelfExtracting",
	}
	if withProvInterface {
		spec["provisioningInterface"] = "eth1"
		spec["osContainerImageUrl"] = "quay.io/podified-antelope-centos9/edpm-hardened-uefi@latest"
		spec["agentImageUrl"] = "quay.io/openstack-k8s-operators/openstack-baremetal-operator-agent@latest"
		spec["apacheImageUrl"] = "registry.redhat.io/rhel8/httpd-24@latest"
		spec["osImage"] = "edpm-hardened-uefi.qcow2"
	}

	return spec
}

// Build BaremetalSetSpec struct for PassThrough mode and fill it with preset values
func PassThroughBaremetalSetSpec(name types.NamespacedName) map[string]any {
	spec := map[string]any{
		"baremetalHosts": map[string]any{
			"compute-0": map[string]any{
				"ctlPlaneIP": "10.0.0.1/24",
			},
		},
		"bmhLabelSelector":      map[string]string{"app": "openstack"},
		"deploymentSSHSecret":   "mysecret",
		"ctlplaneInterface":     "eth0",
		"bmhNamespace":          name.Namespace,
		"osImageDeploymentType": "PassThrough",
		"osContainerImageUrl":   "quay.io/podified-antelope-centos9/edpm-hardened-uefi@latest",
	}
	return spec
}

// Build BaremetalSetSpec struct for two nodes
func TwoNodeBaremetalSetSpec(namespace string) map[string]any {
	spec := map[string]any{
		"baremetalHosts": map[string]any{
			"compute-0": map[string]any{
				"ctlPlaneIP": "10.0.0.1/24",
			},
			"compute-1": map[string]any{
				"ctlPlaneIP": "10.0.0.2/24",
			},
		},
		"bmhLabelSelector":      map[string]string{"app": "openstack"},
		"deploymentSSHSecret":   "mysecret",
		"ctlplaneInterface":     "eth0",
		"bmhNamespace":          namespace,
		"osImageDeploymentType": "SelfExtracting",
	}
	return spec
}

func TwoNodeBaremetalSetSpecWithNodeLabel(namespace string) map[string]any {
	spec := map[string]any{
		"baremetalHosts": map[string]any{
			"compute-0": map[string]any{
				"ctlPlaneIP":       "10.0.0.1/24",
				"bmhLabelSelector": map[string]string{"nodeName": "compute-0"},
			},
			"compute-1": map[string]any{
				"ctlPlaneIP":       "10.0.0.2/24",
				"bmhLabelSelector": map[string]string{"nodeName": "compute-1"},
			},
		},
		"bmhLabelSelector":      map[string]string{"app": "openstack"},
		"deploymentSSHSecret":   "mysecret",
		"ctlplaneInterface":     "eth0",
		"bmhNamespace":          namespace,
		"osImageDeploymentType": "SelfExtracting",
	}
	return spec
}

func TwoNodeBaremetalSetSpecWithWrongNodeLabel(namespace string) map[string]any {
	spec := map[string]any{
		"baremetalHosts": map[string]any{
			"compute-0": map[string]any{
				"ctlPlaneIP":       "10.0.0.1/24",
				"bmhLabelSelector": map[string]string{"nodeName": "compute-0"},
			},
			"compute-1": map[string]any{
				"ctlPlaneIP":       "10.0.0.2/24",
				"bmhLabelSelector": map[string]string{"nodeName": "compute-2"},
			},
		},
		"bmhLabelSelector":      map[string]string{"app": "openstack"},
		"deploymentSSHSecret":   "mysecret",
		"ctlplaneInterface":     "eth0",
		"bmhNamespace":          namespace,
		"osImageDeploymentType": "SelfExtracting",
	}
	return spec
}

func MultiNodeBaremetalSetSpecWithSameNodeLabel(namespace string) map[string]any {
	spec := map[string]any{
		"baremetalHosts": map[string]any{
			"compute-0": map[string]any{
				"ctlPlaneIP":       "10.0.0.1/24",
				"bmhLabelSelector": map[string]string{"nodeType": "compute"},
			},
			"compute-1": map[string]any{
				"ctlPlaneIP":       "10.0.0.2/24",
				"bmhLabelSelector": map[string]string{"nodeType": "compute"},
			},
			"compute-2": map[string]any{
				"ctlPlaneIP":       "10.0.0.3/24",
				"bmhLabelSelector": map[string]string{"nodeType": "compute"},
			},
		},
		"bmhLabelSelector":      map[string]string{"app": "openstack"},
		"deploymentSSHSecret":   "mysecret",
		"ctlplaneInterface":     "eth0",
		"bmhNamespace":          namespace,
		"osImageDeploymentType": "SelfExtracting",
	}
	return spec
}

func MultiNodeBaremetalSetSpecWithOverlappingNodeLabels(namespace string) map[string]any {
	spec := map[string]any{
		"baremetalHosts": map[string]any{
			"compute-0": map[string]any{
				"ctlPlaneIP":       "10.0.0.1/24",
				"bmhLabelSelector": map[string]string{"nodeType": "compute", "dummyLabel": "dummy"},
			},
			"compute-1": map[string]any{
				"ctlPlaneIP":       "10.0.0.2/24",
				"bmhLabelSelector": map[string]string{"nodeType": "compute", "nodeName": "compute-1"},
			},
			"compute-2": map[string]any{
				"ctlPlaneIP":       "10.0.0.3/24",
				"bmhLabelSelector": map[string]string{"nodeType": "compute", "dummyLabel": "dummy"},
			},
		},
		"bmhLabelSelector":      map[string]string{"app": "openstack"},
		"deploymentSSHSecret":   "mysecret",
		"ctlplaneInterface":     "eth0",
		"bmhNamespace":          namespace,
		"osImageDeploymentType": "SelfExtracting",
	}
	return spec
}

// Default BMH Template with preset values
func DefaultBMHTemplate(name types.NamespacedName) map[string]any {
	return map[string]any{
		"apiVersion": "metal3.io/v1alpha1",
		"kind":       "BareMetalHost",
		"metadata": map[string]any{
			"name":      name.Name,
			"namespace": name.Namespace,
			"labels": map[string]string{
				"app": "openstack",
			},
			"annotations": map[string]any{
				"inspect.metal3.io": "disabled",
			},
		},
		"spec": map[string]any{
			"bmc": map[string]any{
				"address":         "fake_address",
				"credentialsName": "fake_credential",
			},
			"bootMACAddress": "52:54:00:39:a7:44",
			"bootMode":       "UEFI",
			"online":         false,
		},
	}
}

// Default BMH Template with preset values
func BMHTemplateWithNodeLabels(name types.NamespacedName, nodeLabels map[string]string) map[string]any {
	labels := util.MergeMaps(map[string]string{"app": "openstack"}, nodeLabels)
	return map[string]any{
		"apiVersion": "metal3.io/v1alpha1",
		"kind":       "BareMetalHost",
		"metadata": map[string]any{
			"name":      name.Name,
			"namespace": name.Namespace,
			"labels":    labels,
			"annotations": map[string]any{
				"inspect.metal3.io": "disabled",
			},
		},
		"spec": map[string]any{
			"bmc": map[string]any{
				"address":         "fake_address",
				"credentialsName": "fake_credential",
			},
			"bootMACAddress": "52:54:00:39:a7:44",
			"bootMode":       "UEFI",
			"online":         false,
		},
	}
}

// Get BaremetalSet
func GetBaremetalSet(name types.NamespacedName) *baremetalv1.OpenStackBaremetalSet {
	instance := &baremetalv1.OpenStackBaremetalSet{}
	Eventually(func(g Gomega) error {
		g.Expect(k8sClient.Get(ctx, name, instance)).Should(Succeed())
		return nil
	}, timeout, interval).Should(Succeed())
	return instance
}

// Create BaremetalHost
func CreateBaremetalHost(name types.NamespacedName) *unstructured.Unstructured {
	instance := DefaultBMHTemplate(name)
	return th.CreateUnstructured(instance)
}

// Create BaremetalHost with NodeLabel
func CreateBaremetalHostWithNodeLabel(name types.NamespacedName,
	nodeLabels map[string]string) *unstructured.Unstructured {
	instance := BMHTemplateWithNodeLabels(name, nodeLabels)
	return th.CreateUnstructured(instance)
}

// Get BaremetalHost
func GetBaremetalHost(name types.NamespacedName) *metal3v1.BareMetalHost {
	instance := &metal3v1.BareMetalHost{}
	Eventually(func(g Gomega) error {
		g.Expect(k8sClient.Get(ctx, name, instance)).Should(Succeed())
		return nil
	}, timeout, interval).Should(Succeed())
	return instance
}

// Get BaremetalSet conditions
func BaremetalSetConditionGetter(name types.NamespacedName) condition.Conditions {
	instance := GetBaremetalSet(name)
	return instance.Status.Conditions
}

// Create DeploymentSSHSecret
func CreateSSHSecret(name types.NamespacedName) *corev1.Secret {
	return th.CreateSecret(
		types.NamespacedName{Namespace: name.Namespace, Name: name.Name},
		map[string][]byte{
			"ssh-privatekey":  []byte("blah"),
			"authorized_keys": []byte("blih"),
		},
	)
}

// Create ProvisionServer
func CreateProvisionServer(name types.NamespacedName, spec map[string]any) *unstructured.Unstructured {
	raw := map[string]interface{}{
		"apiVersion": "baremetal.openstack.org/v1beta1",
		"kind":       "OpenStackProvisionServer",
		"metadata": map[string]interface{}{
			"name":      name.Name,
			"namespace": name.Namespace,
		},
		"spec": spec,
	}
	return th.CreateUnstructured(raw)
}

// Get ProvisionServer (for baremetalset-created provision servers with appended name)
func GetProvisionServer(name types.NamespacedName) *baremetalv1.OpenStackProvisionServer {
	instance := &baremetalv1.OpenStackProvisionServer{}
	name.Name = strings.Join([]string{name.Name, "provisionserver"}, "-")
	Eventually(func(g Gomega) error {
		g.Expect(k8sClient.Get(ctx, name, instance)).Should(Succeed())
		return nil
	}, timeout, interval).Should(Succeed())
	return instance
}

// Get ProvisionServer directly (without name transformation)
func GetProvisionServerDirect(name types.NamespacedName) *baremetalv1.OpenStackProvisionServer {
	instance := &baremetalv1.OpenStackProvisionServer{}
	Eventually(func(g Gomega) error {
		g.Expect(k8sClient.Get(ctx, name, instance)).Should(Succeed())
		return nil
	}, timeout, interval).Should(Succeed())
	return instance
}

// ProvisionServerConditionGetter
func ProvisionServerConditionGetter(name types.NamespacedName) condition.Conditions {
	instance := GetProvisionServerDirect(name)
	return instance.Status.Conditions
}
