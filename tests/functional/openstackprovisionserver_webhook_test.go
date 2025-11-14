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
	"errors"

	. "github.com/onsi/ginkgo/v2" //revive:disable:dot-imports
	. "github.com/onsi/gomega"    //revive:disable:dot-imports
	k8s_errors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/types"

	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

var _ = Describe("OpenStackProvisionServer Webhook", func() {

	var provisionServerName types.NamespacedName

	BeforeEach(func() {
		provisionServerName = types.NamespacedName{
			Name:      "test-provisionserver",
			Namespace: namespace,
		}
	})

	When("Creating ProvisionServer with valid configuration", func() {
		It("should succeed with all required fields", func() {
			spec := map[string]interface{}{
				"osImage":             "edpm-hardened-uefi.qcow2",
				"osContainerImageUrl": "quay.io/podified-antelope-centos9/edpm-hardened-uefi:current-podified",
				"apacheImageUrl":      "registry.redhat.io/ubi9/httpd-24:latest",
				"agentImageUrl":       "quay.io/openstack-k8s-operators/openstack-baremetal-operator-agent:latest",
			}
			raw := map[string]interface{}{
				"apiVersion": "baremetal.openstack.org/v1beta1",
				"kind":       "OpenStackProvisionServer",
				"metadata": map[string]interface{}{
					"name":      provisionServerName.Name,
					"namespace": provisionServerName.Namespace,
				},
				"spec": spec,
			}
			unstructuredObj := &unstructured.Unstructured{Object: raw}
			_, err := controllerutil.CreateOrPatch(
				th.Ctx, th.K8sClient, unstructuredObj, func() error { return nil })
			Expect(err).ShouldNot(HaveOccurred())
		})
	})

	When("Creating ProvisionServer with invalid name", func() {
		It("should fail with name not matching RFC1123", func() {
			invalidName := types.NamespacedName{
				Name:      "Test_Invalid_Name",
				Namespace: namespace,
			}
			spec := map[string]interface{}{
				"osImage":             "edpm-hardened-uefi.qcow2",
				"osContainerImageUrl": "quay.io/podified-antelope-centos9/edpm-hardened-uefi:current-podified",
				"apacheImageUrl":      "registry.redhat.io/ubi9/httpd-24:latest",
				"agentImageUrl":       "quay.io/openstack-k8s-operators/openstack-baremetal-operator-agent:latest",
			}
			raw := map[string]interface{}{
				"apiVersion": "baremetal.openstack.org/v1beta1",
				"kind":       "OpenStackProvisionServer",
				"metadata": map[string]interface{}{
					"name":      invalidName.Name,
					"namespace": invalidName.Namespace,
				},
				"spec": spec,
			}
			unstructuredObj := &unstructured.Unstructured{Object: raw}
			_, err := controllerutil.CreateOrPatch(
				th.Ctx, th.K8sClient, unstructuredObj, func() error { return nil })
			Expect(err).Should(HaveOccurred())
			var statusError *k8s_errors.StatusError
			Expect(errors.As(err, &statusError)).To(BeTrue())
			Expect(statusError.ErrStatus.Message).To(ContainSubstring("RFC 1123"))
		})
	})

	When("Creating ProvisionServer with port out of range", func() {
		It("should fail with port below minimum", func() {
			spec := map[string]interface{}{
				"osImage":             "edpm-hardened-uefi.qcow2",
				"osContainerImageUrl": "quay.io/podified-antelope-centos9/edpm-hardened-uefi:current-podified",
				"apacheImageUrl":      "registry.redhat.io/ubi9/httpd-24:latest",
				"agentImageUrl":       "quay.io/openstack-k8s-operators/openstack-baremetal-operator-agent:latest",
				"port":                int64(6100), // Below minimum 6190
			}
			raw := map[string]interface{}{
				"apiVersion": "baremetal.openstack.org/v1beta1",
				"kind":       "OpenStackProvisionServer",
				"metadata": map[string]interface{}{
					"name":      provisionServerName.Name,
					"namespace": provisionServerName.Namespace,
				},
				"spec": spec,
			}
			unstructuredObj := &unstructured.Unstructured{Object: raw}
			err := k8sClient.Create(ctx, unstructuredObj)
			Expect(err).Should(HaveOccurred())
		})

		It("should fail with port above maximum", func() {
			spec := map[string]interface{}{
				"osImage":             "edpm-hardened-uefi.qcow2",
				"osContainerImageUrl": "quay.io/podified-antelope-centos9/edpm-hardened-uefi:current-podified",
				"apacheImageUrl":      "registry.redhat.io/ubi9/httpd-24:latest",
				"agentImageUrl":       "quay.io/openstack-k8s-operators/openstack-baremetal-operator-agent:latest",
				"port":                int64(6230), // Above maximum 6220
			}
			raw := map[string]interface{}{
				"apiVersion": "baremetal.openstack.org/v1beta1",
				"kind":       "OpenStackProvisionServer",
				"metadata": map[string]interface{}{
					"name":      provisionServerName.Name,
					"namespace": provisionServerName.Namespace,
				},
				"spec": spec,
			}
			unstructuredObj := &unstructured.Unstructured{Object: raw}
			err := k8sClient.Create(ctx, unstructuredObj)
			Expect(err).Should(HaveOccurred())
		})
	})

	When("Creating ProvisionServer with duplicate port", func() {
		var provisionServer2Name types.NamespacedName

		BeforeEach(func() {
			provisionServer2Name = types.NamespacedName{
				Name:      "test-provisionserver-2",
				Namespace: namespace,
			}
			// Create first provision server and let it auto-assign an available port
			spec := map[string]interface{}{
				"osImage":             "edpm-hardened-uefi.qcow2",
				"osContainerImageUrl": "quay.io/podified-antelope-centos9/edpm-hardened-uefi:current-podified",
				"apacheImageUrl":      "registry.redhat.io/ubi9/httpd-24:latest",
				"agentImageUrl":       "quay.io/openstack-k8s-operators/openstack-baremetal-operator-agent:latest",
			}
			DeferCleanup(th.DeleteInstance, CreateProvisionServer(provisionServerName, spec))
		})

		It("should fail when trying to use same port", func() {
			// Get the port that was auto-assigned to the first provision server
			instance := GetProvisionServerDirect(provisionServerName)
			assignedPort := instance.Spec.Port

			spec := map[string]interface{}{
				"osImage":             "edpm-hardened-uefi.qcow2",
				"osContainerImageUrl": "quay.io/podified-antelope-centos9/edpm-hardened-uefi:current-podified",
				"apacheImageUrl":      "registry.redhat.io/ubi9/httpd-24:latest",
				"agentImageUrl":       "quay.io/openstack-k8s-operators/openstack-baremetal-operator-agent:latest",
				"port":                int64(assignedPort), // Same as first server
			}
			raw := map[string]interface{}{
				"apiVersion": "baremetal.openstack.org/v1beta1",
				"kind":       "OpenStackProvisionServer",
				"metadata": map[string]interface{}{
					"name":      provisionServer2Name.Name,
					"namespace": provisionServer2Name.Namespace,
				},
				"spec": spec,
			}
			unstructuredObj := &unstructured.Unstructured{Object: raw}
			_, err := controllerutil.CreateOrPatch(
				th.Ctx, th.K8sClient, unstructuredObj, func() error { return nil })
			Expect(err).Should(HaveOccurred())
			var statusError *k8s_errors.StatusError
			Expect(errors.As(err, &statusError)).To(BeTrue())
			Expect(statusError.ErrStatus.Message).To(ContainSubstring("already in use"))
		})

		It("should succeed when using different port", func() {
			// Don't specify port - it will auto-assign a different available port
			spec := map[string]interface{}{
				"osImage":             "edpm-hardened-uefi.qcow2",
				"osContainerImageUrl": "quay.io/podified-antelope-centos9/edpm-hardened-uefi:current-podified",
				"apacheImageUrl":      "registry.redhat.io/ubi9/httpd-24:latest",
				"agentImageUrl":       "quay.io/openstack-k8s-operators/openstack-baremetal-operator-agent:latest",
			}
			raw := map[string]interface{}{
				"apiVersion": "baremetal.openstack.org/v1beta1",
				"kind":       "OpenStackProvisionServer",
				"metadata": map[string]interface{}{
					"name":      provisionServer2Name.Name,
					"namespace": provisionServer2Name.Namespace,
				},
				"spec": spec,
			}
			unstructuredObj := &unstructured.Unstructured{Object: raw}
			_, err := controllerutil.CreateOrPatch(
				th.Ctx, th.K8sClient, unstructuredObj, func() error { return nil })
			Expect(err).ShouldNot(HaveOccurred())
		})
	})

	When("Creating ProvisionServer with defaulting", func() {
		It("should auto-assign port and default image URLs when not specified", func() {
			spec := map[string]interface{}{
				// Not specifying osImage, port, or image URLs to test defaulting
			}
			DeferCleanup(th.DeleteInstance, CreateProvisionServer(provisionServerName, spec))

			instance := GetProvisionServerDirect(provisionServerName)
			// Port should be auto-assigned
			Expect(instance.Spec.Port).Should(BeNumerically(">", 0))
			// Image URLs and OSImage should be defaulted
			Expect(instance.Spec.OSImage).ShouldNot(BeEmpty())
			Expect(instance.Spec.OSContainerImageURL).ShouldNot(BeEmpty())
			Expect(instance.Spec.ApacheImageURL).ShouldNot(BeEmpty())
			Expect(instance.Spec.AgentImageURL).ShouldNot(BeEmpty())
		})
	})
})
