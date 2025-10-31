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
	. "github.com/onsi/ginkgo/v2" //revive:disable:dot-imports
	. "github.com/onsi/gomega"    //revive:disable:dot-imports
	baremetalv1 "github.com/openstack-k8s-operators/openstack-baremetal-operator/api/v1beta1"

	//revive:disable-next-line:dot-imports
	"github.com/openstack-k8s-operators/lib-common/modules/common/condition"
	. "github.com/openstack-k8s-operators/lib-common/modules/common/test/helpers"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
)

var _ = Describe("ProvisionServer Test", func() {
	var provisionServerName types.NamespacedName

	BeforeEach(func() {
		provisionServerName = types.NamespacedName{
			Name:      "test-provisionserver",
			Namespace: namespace,
		}
	})

	When("A ProvisionServer resource is created", func() {
		BeforeEach(func() {
			spec := map[string]interface{}{
				"osImage":             "edpm-hardened-uefi.qcow2",
				"osContainerImageUrl": "quay.io/podified-antelope-centos9/edpm-hardened-uefi:current-podified",
				"apacheImageUrl":      "registry.redhat.io/ubi9/httpd-24:latest",
				"agentImageUrl":       "quay.io/openstack-k8s-operators/openstack-baremetal-operator-agent:latest",
			}
			DeferCleanup(th.DeleteInstance, CreateProvisionServer(provisionServerName, spec))
		})

		It("should have Conditions initialized", func() {
			th.ExpectCondition(
				provisionServerName,
				ConditionGetterFunc(ProvisionServerConditionGetter),
				condition.ReadyCondition,
				corev1.ConditionFalse,
			)
		})

		It("should auto-assign port within valid range", func() {
			instance := GetProvisionServerDirect(provisionServerName)
			Expect(instance.Spec.Port).Should(BeNumerically(">=", baremetalv1.ProvisionServerPortStart))
			Expect(instance.Spec.Port).Should(BeNumerically("<=", baremetalv1.ProvisionServerPortEnd))
		})

		It("should default OSImageDir", func() {
			instance := GetProvisionServerDirect(provisionServerName)
			Expect(instance.Spec.OSImageDir).ShouldNot(BeNil())
			Expect(*instance.Spec.OSImageDir).Should(Equal("/usr/local/apache2/htdocs"))
		})
	})

	When("Two ProvisionServer instances are created in same namespace", func() {
		var provisionServer2Name types.NamespacedName

		BeforeEach(func() {
			provisionServer2Name = types.NamespacedName{
				Name:      "test-provisionserver-2",
				Namespace: namespace,
			}

			spec1 := map[string]interface{}{
				"osImage":             "edpm-hardened-uefi.qcow2",
				"osContainerImageUrl": "quay.io/podified-antelope-centos9/edpm-hardened-uefi:current-podified",
				"apacheImageUrl":      "registry.redhat.io/ubi9/httpd-24:latest",
				"agentImageUrl":       "quay.io/openstack-k8s-operators/openstack-baremetal-operator-agent:latest",
			}
			DeferCleanup(th.DeleteInstance, CreateProvisionServer(provisionServerName, spec1))

			spec2 := map[string]interface{}{
				"osImage":             "edpm-hardened-uefi.qcow2",
				"osContainerImageUrl": "quay.io/podified-antelope-centos9/edpm-hardened-uefi:current-podified",
				"apacheImageUrl":      "registry.redhat.io/ubi9/httpd-24:latest",
				"agentImageUrl":       "quay.io/openstack-k8s-operators/openstack-baremetal-operator-agent:latest",
			}
			DeferCleanup(th.DeleteInstance, CreateProvisionServer(provisionServer2Name, spec2))
		})

		It("should assign different ports", func() {
			instance1 := GetProvisionServerDirect(provisionServerName)
			instance2 := GetProvisionServerDirect(provisionServer2Name)
			Expect(instance1.Spec.Port).ShouldNot(Equal(instance2.Spec.Port))
		})
	})
})
