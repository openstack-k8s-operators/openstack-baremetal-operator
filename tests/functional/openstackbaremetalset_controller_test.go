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
	metal3v1 "github.com/metal3-io/baremetal-operator/apis/metal3.io/v1alpha1"
	. "github.com/onsi/ginkgo/v2" //revive:disable:dot-imports
	. "github.com/onsi/gomega"    //revive:disable:dot-imports
	baremetalv1 "github.com/openstack-k8s-operators/openstack-baremetal-operator/api/v1beta1"

	//revive:disable-next-line:dot-imports
	"github.com/openstack-k8s-operators/lib-common/modules/common/condition"
	. "github.com/openstack-k8s-operators/lib-common/modules/common/test/helpers"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
)

var _ = Describe("BaremetalSet Test", func() {
	var baremetalSetName types.NamespacedName
	var bmhName types.NamespacedName
	var deploymentSecretName types.NamespacedName

	BeforeEach(func() {
		baremetalSetName = types.NamespacedName{
			Name:      "edpm-compute-baremetalset",
			Namespace: namespace,
		}
		bmhName = types.NamespacedName{
			Name:      "compute-0",
			Namespace: namespace,
		}
		deploymentSecretName = types.NamespacedName{
			Name:      "mysecret",
			Namespace: namespace,
		}
	})

	When("A BaremetalSet resource created", func() {
		BeforeEach(func() {
			DeferCleanup(th.DeleteInstance, CreateBaremetalHost(bmhName))
			bmh := GetBaremetalHost(bmhName)
			Eventually(func(g Gomega) {
				// OpenStackBaremetalSet has the same name as OpenStackDataPlaneNodeSet
				bmh.Status.Provisioning.State = metal3v1.StateAvailable
				g.Expect(th.K8sClient.Status().Update(th.Ctx, bmh)).To(Succeed())

			}, th.Timeout, th.Interval).Should(Succeed())
			DeferCleanup(th.DeleteInstance, CreateBaremetalSet(baremetalSetName, DefaultBaremetalSetSpec(bmhName, false)))
		})
		It("should have the Spec fields initialized", func() {
			baremetalSetInstance := GetBaremetalSet(baremetalSetName)
			spec := baremetalv1.OpenStackBaremetalSetSpec{
				BaremetalHosts: map[string]baremetalv1.InstanceSpec{
					"compute-0": {
						CtlPlaneIP:                     "10.0.0.1",
						UserData:                       nil,
						NetworkData:                    nil,
						PreprovisioningNetworkDataName: "",
						BmhLabelSelector:               nil,
					},
				},
				OSImage:               "",
				OSContainerImageURL:   "",
				ApacheImageURL:        "",
				AgentImageURL:         "",
				UserData:              nil,
				NetworkData:           nil,
				AutomatedCleaningMode: "metadata",
				ProvisionServerName:   "",
				ProvisioningInterface: "",
				DeploymentSSHSecret:   "mysecret",
				CtlplaneInterface:     "eth0",
				CtlplaneGateway:       "",
				CtlplaneNetmask:       "255.255.255.0",
				BmhNamespace:          baremetalSetName.Namespace,
				BmhLabelSelector:      map[string]string{"app": "openstack"},
				HardwareReqs: baremetalv1.HardwareReqs{
					CPUReqs: baremetalv1.CPUReqs{
						Arch:     "",
						CountReq: baremetalv1.CPUCountReq{Count: 0, ExactMatch: false},
						MhzReq:   baremetalv1.CPUMhzReq{Mhz: 0, ExactMatch: false},
					},
					MemReqs: baremetalv1.MemReqs{
						GbReq: baremetalv1.MemGbReq{Gb: 0, ExactMatch: false},
					},
					DiskReqs: baremetalv1.DiskReqs{
						GbReq:  baremetalv1.DiskGbReq{Gb: 0, ExactMatch: false},
						SSDReq: baremetalv1.DiskSSDReq{SSD: false, ExactMatch: false},
					},
				},
				PasswordSecret:   nil,
				CloudUserName:    "cloud-admin",
				DomainName:       "",
				BootstrapDNS:     nil,
				DNSSearchDomains: nil,
			}
			Expect(baremetalSetInstance.Spec).Should(Equal(spec))
		})
		It("should have Conditions initialized", func() {
			th.ExpectCondition(
				baremetalSetName,
				ConditionGetterFunc(BaremetalSetConditionGetter),
				condition.ReadyCondition,
				corev1.ConditionFalse,
			)
			th.ExpectCondition(
				baremetalSetName,
				ConditionGetterFunc(BaremetalSetConditionGetter),
				condition.InputReadyCondition,
				corev1.ConditionFalse,
			)
			th.ExpectCondition(
				baremetalSetName,
				ConditionGetterFunc(BaremetalSetConditionGetter),
				baremetalv1.OpenStackBaremetalSetProvServerReadyCondition,
				corev1.ConditionUnknown,
			)
			th.ExpectCondition(
				baremetalSetName,
				ConditionGetterFunc(BaremetalSetConditionGetter),
				baremetalv1.OpenStackBaremetalSetBmhProvisioningReadyCondition,
				corev1.ConditionUnknown,
			)
		})
	})

	When("A deployment ssh secret is created", func() {
		BeforeEach(func() {
			DeferCleanup(th.DeleteInstance, CreateBaremetalHost(bmhName))
			bmh := GetBaremetalHost(bmhName)
			Eventually(func(g Gomega) {
				bmh.Status.Provisioning.State = metal3v1.StateAvailable
				g.Expect(th.K8sClient.Status().Update(th.Ctx, bmh)).To(Succeed())

			}, th.Timeout, th.Interval).Should(Succeed())
			DeferCleanup(th.DeleteInstance, CreateSSHSecret(deploymentSecretName))
			DeferCleanup(th.DeleteInstance, CreateBaremetalSet(baremetalSetName, DefaultBaremetalSetSpec(bmhName, false)))
		})
		It("Should set Input Ready", func() {
			th.ExpectCondition(
				baremetalSetName,
				ConditionGetterFunc(BaremetalSetConditionGetter),
				condition.InputReadyCondition,
				corev1.ConditionTrue,
			)

		})
	})

	When("Provisioning interface provided", func() {
		BeforeEach(func() {
			DeferCleanup(th.DeleteInstance, CreateBaremetalHost(bmhName))
			bmh := GetBaremetalHost(bmhName)
			Eventually(func(g Gomega) {
				bmh.Status.Provisioning.State = metal3v1.StateAvailable
				g.Expect(th.K8sClient.Status().Update(th.Ctx, bmh)).To(Succeed())

			}, th.Timeout, th.Interval).Should(Succeed())
			DeferCleanup(th.DeleteInstance, CreateSSHSecret(deploymentSecretName))
			DeferCleanup(th.DeleteInstance, CreateBaremetalSet(baremetalSetName, DefaultBaremetalSetSpec(bmhName, true)))
		})
		It("Prov Server should have the Spec fields initialized", func() {
			osImageDir := "/usr/local/apache2/htdocs"

			provServer := GetProvisionServer(baremetalSetName)
			spec := baremetalv1.OpenStackProvisionServerSpec{
				Port:                6190,
				Interface:           "eth1",
				OSImage:             "edpm-hardened-uefi.qcow2",
				OSImageDir:          &osImageDir,
				OSContainerImageURL: "quay.io/podified-antelope-centos9/edpm-hardened-uefi@latest",
				ApacheImageURL:      "registry.redhat.io/rhel8/httpd-24@latest",
				AgentImageURL:       "quay.io/openstack-k8s-operators/openstack-baremetal-operator-agent@latest",
				NodeSelector:        nil,
				Resources:           corev1.ResourceRequirements{Limits: nil, Requests: nil, Claims: nil},
			}
			Expect(provServer.Spec).Should(Equal(spec))
		})

		It("Should set Provision Server Ready", func() {
			th.ExpectCondition(
				baremetalSetName,
				ConditionGetterFunc(BaremetalSetConditionGetter),
				baremetalv1.OpenStackBaremetalSetProvServerReadyCondition,
				corev1.ConditionFalse,
			)

		})
	})

})
