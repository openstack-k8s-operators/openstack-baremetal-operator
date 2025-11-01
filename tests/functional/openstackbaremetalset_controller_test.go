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
	. "github.com/onsi/ginkgo/v2" //revive:disable:dot-imports
	. "github.com/onsi/gomega"    //revive:disable:dot-imports
	baremetalv1 "github.com/openstack-k8s-operators/openstack-baremetal-operator/api/v1beta1"
	corev1 "k8s.io/api/core/v1"
	k8s_errors "k8s.io/apimachinery/pkg/api/errors"

	//revive:disable-next-line:dot-imports
	"github.com/openstack-k8s-operators/lib-common/modules/common/condition"
	. "github.com/openstack-k8s-operators/lib-common/modules/common/test/helpers"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

var _ = Describe("BaremetalSet Test", func() {
	var baremetalSetName types.NamespacedName
	var baremetalSet2Name types.NamespacedName
	var bmhName types.NamespacedName
	var bmh2Name types.NamespacedName
	var deploymentSecretName types.NamespacedName
	var secondaryDeploymentSecretName types.NamespacedName

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
		baremetalSet2Name = types.NamespacedName{
			Name:      "edpm-compute-baremetalset",
			Namespace: secondaryNamespace,
		}
		bmh2Name = types.NamespacedName{
			Name:      "compute-0",
			Namespace: secondaryNamespace,
		}
		secondaryDeploymentSecretName = types.NamespacedName{
			Name:      "mysecret",
			Namespace: secondaryNamespace,
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
			coreSpec := baremetalv1.OpenStackBaremetalSetTemplateSpec{
				OSImage:               "edpm-hardened-uefi.qcow2",
				OSContainerImageURL:   "",
				ApacheImageURL:        "",
				AgentImageURL:         "",
				AutomatedCleaningMode: "metadata",
				ProvisionServerName:   "",
				ProvisioningInterface: "",
				DeploymentSSHSecret:   "mysecret",
				CtlplaneInterface:     "eth0",
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
				PasswordSecret: nil,
				CloudUserName:  "cloud-admin",
				DomainName:     "",
			}
			spec := baremetalv1.OpenStackBaremetalSetSpec{
				BaremetalHosts: map[string]baremetalv1.InstanceSpec{
					"compute-0": {
						CtlPlaneIP:       "10.0.0.1/24",
						UserData:         nil,
						NetworkData:      nil,
						BmhLabelSelector: nil,
					},
				},
				BootstrapDNS:                      nil,
				DNSSearchDomains:                  nil,
				OpenStackBaremetalSetTemplateSpec: coreSpec,
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

			provServer := GetProvisionServer(baremetalSetName)
			Expect(provServer.Spec.Interface).Should(Equal("eth1"))
		})

		It("Should set Provision Server Ready", func() {
			th.ExpectCondition(
				baremetalSetName,
				ConditionGetterFunc(BaremetalSetConditionGetter),
				baremetalv1.OpenStackBaremetalSetProvServerReadyCondition,
				corev1.ConditionFalse,
			)
		})

		It("Should clean-up its auto-generated Provision Server if provisionServerName is later provided", func() {
			Eventually(func(g Gomega) {
				baremetalSet := GetBaremetalSet(baremetalSetName)
				baremetalSet.Spec.ProvisionServerName = "unimportant"
				g.Expect(th.K8sClient.Update(th.Ctx, baremetalSet)).To(Succeed())
			}, th.Timeout, th.Interval).Should(Succeed())

			Eventually(func(g Gomega) {
				provisionServerName := types.NamespacedName{
					Name:      strings.Join([]string{baremetalSetName.Name, "provisionserver"}, "-"),
					Namespace: namespace,
				}
				instance := &baremetalv1.OpenStackProvisionServer{}
				err := k8sClient.Get(ctx, provisionServerName, instance)
				g.Expect(k8s_errors.IsNotFound(err)).To(BeTrue())
			}, th.Timeout, th.Interval).Should(Succeed())
		})
	})

	When("Two ProvisionServer instances are created with the same name, in different namespaces", func() {
		BeforeEach(func() {

			DeferCleanup(th.DeleteInstance, CreateBaremetalHost(bmhName))
			bmh := GetBaremetalHost(bmhName)
			Eventually(func(g Gomega) {
				bmh.Status.Provisioning.State = metal3v1.StateAvailable
				g.Expect(th.K8sClient.Status().Update(th.Ctx, bmh)).To(Succeed())

			}, th.Timeout, th.Interval).Should(Succeed())

			DeferCleanup(th.DeleteInstance, CreateBaremetalHost(bmh2Name))
			bmh2 := GetBaremetalHost(bmh2Name)
			Eventually(func(g Gomega) {
				bmh2.Status.Provisioning.State = metal3v1.StateAvailable
				g.Expect(th.K8sClient.Status().Update(th.Ctx, bmh2)).To(Succeed())

			}, th.Timeout, th.Interval).Should(Succeed())
			DeferCleanup(th.DeleteInstance, CreateSSHSecret(deploymentSecretName))
			DeferCleanup(th.DeleteInstance, CreateSSHSecret(secondaryDeploymentSecretName))
			DeferCleanup(th.DeleteInstance, CreateBaremetalSet(baremetalSetName, DefaultBaremetalSetSpec(bmhName, true)))
			DeferCleanup(th.DeleteInstance, CreateBaremetalSet(baremetalSet2Name, DefaultBaremetalSetSpec(bmh2Name, true)))
		})
		It("Each ProvisionServer should use different ports", func() {

			provServer := GetProvisionServer(baremetalSetName)
			provServer2 := GetProvisionServer(baremetalSet2Name)
			Expect(provServer.Spec.Port).ShouldNot(Equal(provServer2.Spec.Port))
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

	When("BMH provisioned with generated userdata and networkdata", func() {
		BeforeEach(func() {
			DeferCleanup(th.DeleteInstance, CreateBaremetalHost(bmhName))
			Eventually(func(g Gomega) {
				bmh := GetBaremetalHost(bmhName)
				bmh.Status.Provisioning.State = metal3v1.StateAvailable
				g.Expect(th.K8sClient.Status().Update(th.Ctx, bmh)).To(Succeed())
			}, th.Timeout, th.Interval).Should(Succeed())

			DeferCleanup(th.DeleteInstance, CreateSSHSecret(deploymentSecretName))
			DeferCleanup(th.DeleteInstance, CreateBaremetalSet(baremetalSetName, DefaultBaremetalSetSpec(bmhName, true)))

			// Patch the provision server to have LocalImageURL set
			Eventually(func(g Gomega) {
				provServer := GetProvisionServer(baremetalSetName)
				provServer.Status.LocalImageURL = "http://192.168.1.100:6190/images/edpm-hardened-uefi.qcow2"
				provServer.Status.LocalImageChecksumURL = "http://192.168.1.100:6190/images/edpm-hardened-uefi.qcow2.md5sum"
				provServer.Status.OSImageChecksumType = metal3v1.MD5
				g.Expect(th.K8sClient.Status().Update(th.Ctx, provServer)).To(Succeed())
			}, th.Timeout, th.Interval).Should(Succeed())
		})

		It("Should generate userdata secret with correct content", func() {
			// Wait for BMH to be provisioned
			Eventually(func(g Gomega) {
				baremetalSet := GetBaremetalSet(baremetalSetName)
				g.Expect(baremetalSet.Status.BaremetalHosts).To(HaveKey("compute-0"))
				g.Expect(baremetalSet.Status.BaremetalHosts["compute-0"].UserDataSecretName).ToNot(BeEmpty())
			}, th.Timeout, th.Interval).Should(Succeed())

			baremetalSet := GetBaremetalSet(baremetalSetName)
			userDataSecretName := baremetalSet.Status.BaremetalHosts["compute-0"].UserDataSecretName

			// Verify userdata secret exists and has expected content
			userDataSecret := th.GetSecret(types.NamespacedName{
				Name:      userDataSecretName,
				Namespace: bmhName.Namespace,
			})
			Expect(userDataSecret.Data).To(HaveKey("userData"))
			userData := string(userDataSecret.Data["userData"])
			Expect(userData).To(ContainSubstring("#cloud-config"))
			Expect(userData).To(ContainSubstring("hostname: compute-0"))
			Expect(userData).To(ContainSubstring("cloud-admin"))
		})

		It("Should generate networkdata secret with correct content", func() {
			// Wait for BMH to be provisioned
			Eventually(func(g Gomega) {
				baremetalSet := GetBaremetalSet(baremetalSetName)
				g.Expect(baremetalSet.Status.BaremetalHosts).To(HaveKey("compute-0"))
				g.Expect(baremetalSet.Status.BaremetalHosts["compute-0"].NetworkDataSecretName).ToNot(BeEmpty())
			}, th.Timeout, th.Interval).Should(Succeed())

			baremetalSet := GetBaremetalSet(baremetalSetName)
			networkDataSecretName := baremetalSet.Status.BaremetalHosts["compute-0"].NetworkDataSecretName

			// Verify networkdata secret exists and has expected content
			networkDataSecret := th.GetSecret(types.NamespacedName{
				Name:      networkDataSecretName,
				Namespace: bmhName.Namespace,
			})
			Expect(networkDataSecret.Data).To(HaveKey("networkData"))
			networkData := string(networkDataSecret.Data["networkData"])
			// Verify proper YAML formatting - links: should be on its own line
			Expect(networkData).To(MatchRegexp("(?m)^links:\n"))
			Expect(networkData).To(MatchRegexp("(?m)^networks:\n"))
			Expect(networkData).NotTo(ContainSubstring("links:- "))
			Expect(networkData).NotTo(ContainSubstring("networks:- "))
			Expect(networkData).To(ContainSubstring("links:"))
			Expect(networkData).To(ContainSubstring("name: eth0"))
			Expect(networkData).To(ContainSubstring("ip_address: 10.0.0.1"))
			Expect(networkData).To(ContainSubstring("type: ipv4"))
		})

		It("Should set BMH image URLs from provision server", func() {
			Eventually(func(g Gomega) {
				bmh := GetBaremetalHost(bmhName)
				g.Expect(bmh.Spec.Image).ToNot(BeNil())
				g.Expect(bmh.Spec.Image.URL).To(Equal("http://192.168.1.100:6190/images/edpm-hardened-uefi.qcow2"))
				g.Expect(bmh.Spec.Image.Checksum).To(Equal("http://192.168.1.100:6190/images/edpm-hardened-uefi.qcow2.md5sum"))
			}, th.Timeout, th.Interval).Should(Succeed())
		})

		It("Should set BMH ConsumerRef and UserData/NetworkData references", func() {
			Eventually(func(g Gomega) {
				bmh := GetBaremetalHost(bmhName)
				g.Expect(bmh.Spec.ConsumerRef).ToNot(BeNil())
				g.Expect(bmh.Spec.ConsumerRef.Name).To(Equal(baremetalSetName.Name))
				g.Expect(bmh.Spec.UserData).ToNot(BeNil())
				g.Expect(bmh.Spec.NetworkData).ToNot(BeNil())
			}, th.Timeout, th.Interval).Should(Succeed())
		})
	})

	When("BMH provisioned with VLAN configuration", func() {
		BeforeEach(func() {
			DeferCleanup(th.DeleteInstance, CreateBaremetalHost(bmhName))
			bmh := GetBaremetalHost(bmhName)
			Eventually(func(g Gomega) {
				bmh.Status.Provisioning.State = metal3v1.StateAvailable
				g.Expect(th.K8sClient.Status().Update(th.Ctx, bmh)).To(Succeed())
			}, th.Timeout, th.Interval).Should(Succeed())

			DeferCleanup(th.DeleteInstance, CreateSSHSecret(deploymentSecretName))

			// Create baremetalset with VLAN configuration
			vlanID := 100
			spec := DefaultBaremetalSetSpec(bmhName, true)
			spec["ctlplaneVlan"] = vlanID
			spec["baremetalHosts"] = map[string]any{
				"compute-0": map[string]any{
					"ctlPlaneIP": "10.0.0.1/24",
				},
			}
			DeferCleanup(th.DeleteInstance, CreateBaremetalSet(baremetalSetName, spec))

			// Patch the provision server to have LocalImageURL set
			Eventually(func(g Gomega) {
				provServer := GetProvisionServer(baremetalSetName)
				provServer.Status.LocalImageURL = "http://192.168.1.100:6190/images/edpm-hardened-uefi.qcow2"
				provServer.Status.LocalImageChecksumURL = "http://192.168.1.100:6190/images/edpm-hardened-uefi.qcow2.md5sum"
				provServer.Status.OSImageChecksumType = metal3v1.MD5
				g.Expect(th.K8sClient.Status().Update(th.Ctx, provServer)).To(Succeed())
			}, th.Timeout, th.Interval).Should(Succeed())
		})

		It("Should generate networkdata with VLAN configuration", func() {
			Eventually(func(g Gomega) {
				baremetalSet := GetBaremetalSet(baremetalSetName)
				g.Expect(baremetalSet.Status.BaremetalHosts).To(HaveKey("compute-0"))
				g.Expect(baremetalSet.Status.BaremetalHosts["compute-0"].NetworkDataSecretName).ToNot(BeEmpty())
			}, th.Timeout, th.Interval).Should(Succeed())

			baremetalSet := GetBaremetalSet(baremetalSetName)
			networkDataSecretName := baremetalSet.Status.BaremetalHosts["compute-0"].NetworkDataSecretName

			networkDataSecret := th.GetSecret(types.NamespacedName{
				Name:      networkDataSecretName,
				Namespace: bmhName.Namespace,
			})
			networkData := string(networkDataSecret.Data["networkData"])
			Expect(networkData).To(ContainSubstring("vlan_id: 100"))
			Expect(networkData).To(ContainSubstring("type: vlan"))
			Expect(networkData).To(ContainSubstring("eth0.100"))
		})
	})

	When("BMH provisioned with Gateway and DNS configuration", func() {
		BeforeEach(func() {
			DeferCleanup(th.DeleteInstance, CreateBaremetalHost(bmhName))
			bmh := GetBaremetalHost(bmhName)
			Eventually(func(g Gomega) {
				bmh.Status.Provisioning.State = metal3v1.StateAvailable
				g.Expect(th.K8sClient.Status().Update(th.Ctx, bmh)).To(Succeed())
			}, th.Timeout, th.Interval).Should(Succeed())

			DeferCleanup(th.DeleteInstance, CreateSSHSecret(deploymentSecretName))

			// Create baremetalset with gateway and DNS
			spec := DefaultBaremetalSetSpec(bmhName, true)
			spec["ctlplaneGateway"] = "10.0.0.254"
			spec["bootstrapDns"] = []string{"8.8.8.8", "8.8.4.4"}
			spec["dnsSearchDomains"] = []string{"example.com", "test.local"}
			spec["baremetalHosts"] = map[string]any{
				"compute-0": map[string]any{
					"ctlPlaneIP": "10.0.0.1/24",
				},
			}
			DeferCleanup(th.DeleteInstance, CreateBaremetalSet(baremetalSetName, spec))

			// Patch the provision server to have LocalImageURL set
			Eventually(func(g Gomega) {
				provServer := GetProvisionServer(baremetalSetName)
				provServer.Status.LocalImageURL = "http://192.168.1.100:6190/images/edpm-hardened-uefi.qcow2"
				provServer.Status.LocalImageChecksumURL = "http://192.168.1.100:6190/images/edpm-hardened-uefi.qcow2.md5sum"
				provServer.Status.OSImageChecksumType = metal3v1.MD5
				g.Expect(th.K8sClient.Status().Update(th.Ctx, provServer)).To(Succeed())
			}, th.Timeout, th.Interval).Should(Succeed())
		})

		It("Should generate networkdata with gateway configuration", func() {
			Eventually(func(g Gomega) {
				baremetalSet := GetBaremetalSet(baremetalSetName)
				g.Expect(baremetalSet.Status.BaremetalHosts).To(HaveKey("compute-0"))
				g.Expect(baremetalSet.Status.BaremetalHosts["compute-0"].NetworkDataSecretName).ToNot(BeEmpty())
			}, th.Timeout, th.Interval).Should(Succeed())

			baremetalSet := GetBaremetalSet(baremetalSetName)
			networkDataSecretName := baremetalSet.Status.BaremetalHosts["compute-0"].NetworkDataSecretName

			networkDataSecret := th.GetSecret(types.NamespacedName{
				Name:      networkDataSecretName,
				Namespace: bmhName.Namespace,
			})
			networkData := string(networkDataSecret.Data["networkData"])
			// Verify proper YAML formatting for routes
			Expect(networkData).To(MatchRegexp("(?m)^  routes:\n"))
			Expect(networkData).NotTo(ContainSubstring("routes:  - network:"))
			Expect(networkData).To(ContainSubstring("gateway: 10.0.0.254"))
			Expect(networkData).To(ContainSubstring("routes:"))
		})

		It("Should generate networkdata with DNS configuration", func() {
			Eventually(func(g Gomega) {
				baremetalSet := GetBaremetalSet(baremetalSetName)
				g.Expect(baremetalSet.Status.BaremetalHosts).To(HaveKey("compute-0"))
			}, th.Timeout, th.Interval).Should(Succeed())

			baremetalSet := GetBaremetalSet(baremetalSetName)
			networkDataSecretName := baremetalSet.Status.BaremetalHosts["compute-0"].NetworkDataSecretName

			networkDataSecret := th.GetSecret(types.NamespacedName{
				Name:      networkDataSecretName,
				Namespace: bmhName.Namespace,
			})
			networkData := string(networkDataSecret.Data["networkData"])
			// Verify proper YAML formatting for DNS sections
			Expect(networkData).To(MatchRegexp("(?m)^  dns_nameservers:\n"))
			Expect(networkData).NotTo(ContainSubstring("dns_nameservers:    - "))
			Expect(networkData).To(MatchRegexp("(?m)^  dns_search:\n"))
			Expect(networkData).NotTo(ContainSubstring("dns_search:    - "))
			Expect(networkData).To(ContainSubstring("dns_nameservers:"))
			Expect(networkData).To(ContainSubstring("8.8.8.8"))
			Expect(networkData).To(ContainSubstring("8.8.4.4"))
			Expect(networkData).To(ContainSubstring("dns_search:"))
			Expect(networkData).To(ContainSubstring("example.com"))
			Expect(networkData).To(ContainSubstring("test.local"))
		})
	})

	When("BMH provisioned with per-instance overrides", func() {
		BeforeEach(func() {
			DeferCleanup(th.DeleteInstance, CreateBaremetalHost(bmhName))
			bmh := GetBaremetalHost(bmhName)
			Eventually(func(g Gomega) {
				bmh.Status.Provisioning.State = metal3v1.StateAvailable
				g.Expect(th.K8sClient.Status().Update(th.Ctx, bmh)).To(Succeed())
			}, th.Timeout, th.Interval).Should(Succeed())

			DeferCleanup(th.DeleteInstance, CreateSSHSecret(deploymentSecretName))

			// Create baremetalset with per-instance overrides
			vlanID := 200
			spec := DefaultBaremetalSetSpec(bmhName, true)
			spec["ctlplaneGateway"] = "10.0.0.254"
			spec["baremetalHosts"] = map[string]any{
				"compute-0": map[string]any{
					"ctlPlaneIP":        "10.0.0.1/24",
					"ctlplaneInterface": "ens3",
					"ctlplaneGateway":   "10.0.0.1",
					"ctlplaneVlan":      vlanID,
				},
			}
			DeferCleanup(th.DeleteInstance, CreateBaremetalSet(baremetalSetName, spec))

			// Patch the provision server to have LocalImageURL set
			Eventually(func(g Gomega) {
				provServer := GetProvisionServer(baremetalSetName)
				provServer.Status.LocalImageURL = "http://192.168.1.100:6190/images/edpm-hardened-uefi.qcow2"
				provServer.Status.LocalImageChecksumURL = "http://192.168.1.100:6190/images/edpm-hardened-uefi.qcow2.md5sum"
				provServer.Status.OSImageChecksumType = metal3v1.MD5
				g.Expect(th.K8sClient.Status().Update(th.Ctx, provServer)).To(Succeed())
			}, th.Timeout, th.Interval).Should(Succeed())
		})

		It("Should use per-instance interface override in networkdata", func() {
			Eventually(func(g Gomega) {
				baremetalSet := GetBaremetalSet(baremetalSetName)
				g.Expect(baremetalSet.Status.BaremetalHosts).To(HaveKey("compute-0"))
			}, th.Timeout, th.Interval).Should(Succeed())

			baremetalSet := GetBaremetalSet(baremetalSetName)
			networkDataSecretName := baremetalSet.Status.BaremetalHosts["compute-0"].NetworkDataSecretName

			networkDataSecret := th.GetSecret(types.NamespacedName{
				Name:      networkDataSecretName,
				Namespace: bmhName.Namespace,
			})
			networkData := string(networkDataSecret.Data["networkData"])
			Expect(networkData).To(ContainSubstring("name: ens3"))
			Expect(networkData).ToNot(ContainSubstring("name: eth0"))
		})

		It("Should use per-instance VLAN override in networkdata", func() {
			Eventually(func(g Gomega) {
				baremetalSet := GetBaremetalSet(baremetalSetName)
				g.Expect(baremetalSet.Status.BaremetalHosts).To(HaveKey("compute-0"))
			}, th.Timeout, th.Interval).Should(Succeed())

			baremetalSet := GetBaremetalSet(baremetalSetName)
			networkDataSecretName := baremetalSet.Status.BaremetalHosts["compute-0"].NetworkDataSecretName

			networkDataSecret := th.GetSecret(types.NamespacedName{
				Name:      networkDataSecretName,
				Namespace: bmhName.Namespace,
			})
			networkData := string(networkDataSecret.Data["networkData"])
			Expect(networkData).To(ContainSubstring("vlan_id: 200"))
			Expect(networkData).To(ContainSubstring("ens3.200"))
		})

		It("Should use per-instance gateway override in networkdata", func() {
			Eventually(func(g Gomega) {
				baremetalSet := GetBaremetalSet(baremetalSetName)
				g.Expect(baremetalSet.Status.BaremetalHosts).To(HaveKey("compute-0"))
			}, th.Timeout, th.Interval).Should(Succeed())

			baremetalSet := GetBaremetalSet(baremetalSetName)
			networkDataSecretName := baremetalSet.Status.BaremetalHosts["compute-0"].NetworkDataSecretName

			networkDataSecret := th.GetSecret(types.NamespacedName{
				Name:      networkDataSecretName,
				Namespace: bmhName.Namespace,
			})
			networkData := string(networkDataSecret.Data["networkData"])
			Expect(networkData).To(ContainSubstring("gateway: 10.0.0.1"))
		})
	})

	When("BMH provisioned with password secret", func() {
		var passwordSecretName types.NamespacedName

		BeforeEach(func() {
			passwordSecretName = types.NamespacedName{
				Name:      "password-secret",
				Namespace: namespace,
			}

			DeferCleanup(th.DeleteInstance, CreateBaremetalHost(bmhName))
			bmh := GetBaremetalHost(bmhName)
			Eventually(func(g Gomega) {
				bmh.Status.Provisioning.State = metal3v1.StateAvailable
				g.Expect(th.K8sClient.Status().Update(th.Ctx, bmh)).To(Succeed())
			}, th.Timeout, th.Interval).Should(Succeed())

			DeferCleanup(th.DeleteInstance, CreateSSHSecret(deploymentSecretName))

			// Create password secret
			DeferCleanup(th.DeleteInstance, th.CreateSecret(
				passwordSecretName,
				map[string][]byte{
					"NodeRootPassword": []byte("supersecret"),
				},
			))

			// Create baremetalset with password secret
			spec := DefaultBaremetalSetSpec(bmhName, true)
			spec["passwordSecret"] = map[string]any{
				"name":      passwordSecretName.Name,
				"namespace": passwordSecretName.Namespace,
			}
			DeferCleanup(th.DeleteInstance, CreateBaremetalSet(baremetalSetName, spec))

			// Patch the provision server to have LocalImageURL set
			Eventually(func(g Gomega) {
				provServer := GetProvisionServer(baremetalSetName)
				provServer.Status.LocalImageURL = "http://192.168.1.100:6190/images/edpm-hardened-uefi.qcow2"
				provServer.Status.LocalImageChecksumURL = "http://192.168.1.100:6190/images/edpm-hardened-uefi.qcow2.md5sum"
				provServer.Status.OSImageChecksumType = metal3v1.MD5
				g.Expect(th.K8sClient.Status().Update(th.Ctx, provServer)).To(Succeed())
			}, th.Timeout, th.Interval).Should(Succeed())
		})

		It("Should include root password in userdata", func() {
			Eventually(func(g Gomega) {
				baremetalSet := GetBaremetalSet(baremetalSetName)
				g.Expect(baremetalSet.Status.BaremetalHosts).To(HaveKey("compute-0"))
			}, th.Timeout, th.Interval).Should(Succeed())

			baremetalSet := GetBaremetalSet(baremetalSetName)
			userDataSecretName := baremetalSet.Status.BaremetalHosts["compute-0"].UserDataSecretName

			userDataSecret := th.GetSecret(types.NamespacedName{
				Name:      userDataSecretName,
				Namespace: bmhName.Namespace,
			})
			userData := string(userDataSecret.Data["userData"])
			Expect(userData).To(ContainSubstring("disable_root: false"))
			Expect(userData).To(ContainSubstring("ssh_pwauth:   true"))
			Expect(userData).To(ContainSubstring("chpasswd:"))
			Expect(userData).To(ContainSubstring("root:supersecret"))
		})
	})

	When("BMH provisioned with custom UserData and NetworkData", func() {
		var customUserDataSecret types.NamespacedName
		var customNetworkDataSecret types.NamespacedName

		BeforeEach(func() {
			customUserDataSecret = types.NamespacedName{
				Name:      "custom-userdata",
				Namespace: namespace,
			}
			customNetworkDataSecret = types.NamespacedName{
				Name:      "custom-networkdata",
				Namespace: namespace,
			}

			DeferCleanup(th.DeleteInstance, CreateBaremetalHost(bmhName))
			bmh := GetBaremetalHost(bmhName)
			Eventually(func(g Gomega) {
				bmh.Status.Provisioning.State = metal3v1.StateAvailable
				g.Expect(th.K8sClient.Status().Update(th.Ctx, bmh)).To(Succeed())
			}, th.Timeout, th.Interval).Should(Succeed())

			DeferCleanup(th.DeleteInstance, CreateSSHSecret(deploymentSecretName))

			// Create custom userdata secret
			DeferCleanup(th.DeleteInstance, th.CreateSecret(
				customUserDataSecret,
				map[string][]byte{
					"userData": []byte("#cloud-config\nhostname: custom-host"),
				},
			))

			// Create custom networkdata secret
			DeferCleanup(th.DeleteInstance, th.CreateSecret(
				customNetworkDataSecret,
				map[string][]byte{
					"networkData": []byte("links:\n- name: custom-nic"),
				},
			))

			// Create baremetalset with custom secrets
			spec := DefaultBaremetalSetSpec(bmhName, true)
			spec["baremetalHosts"] = map[string]any{
				"compute-0": map[string]any{
					"ctlPlaneIP": "10.0.0.1/24",
					"userData": map[string]any{
						"name":      customUserDataSecret.Name,
						"namespace": customUserDataSecret.Namespace,
					},
					"networkData": map[string]any{
						"name":      customNetworkDataSecret.Name,
						"namespace": customNetworkDataSecret.Namespace,
					},
				},
			}
			DeferCleanup(th.DeleteInstance, CreateBaremetalSet(baremetalSetName, spec))

			// Patch the provision server to have LocalImageURL set
			Eventually(func(g Gomega) {
				provServer := GetProvisionServer(baremetalSetName)
				provServer.Status.LocalImageURL = "http://192.168.1.100:6190/images/edpm-hardened-uefi.qcow2"
				provServer.Status.LocalImageChecksumURL = "http://192.168.1.100:6190/images/edpm-hardened-uefi.qcow2.md5sum"
				provServer.Status.OSImageChecksumType = metal3v1.MD5
				g.Expect(th.K8sClient.Status().Update(th.Ctx, provServer)).To(Succeed())
			}, th.Timeout, th.Interval).Should(Succeed())
		})

		It("Should use custom UserData secret", func() {
			Eventually(func(g Gomega) {
				baremetalSet := GetBaremetalSet(baremetalSetName)
				g.Expect(baremetalSet.Status.BaremetalHosts).To(HaveKey("compute-0"))
				g.Expect(baremetalSet.Status.BaremetalHosts["compute-0"].UserDataSecretName).To(Equal(customUserDataSecret.Name))
			}, th.Timeout, th.Interval).Should(Succeed())
		})

		It("Should use custom NetworkData secret", func() {
			Eventually(func(g Gomega) {
				baremetalSet := GetBaremetalSet(baremetalSetName)
				g.Expect(baremetalSet.Status.BaremetalHosts).To(HaveKey("compute-0"))
				g.Expect(baremetalSet.Status.BaremetalHosts["compute-0"].NetworkDataSecretName).To(Equal(customNetworkDataSecret.Name))
			}, th.Timeout, th.Interval).Should(Succeed())
		})

		It("Should set BMH with custom secret references", func() {
			Eventually(func(g Gomega) {
				bmh := GetBaremetalHost(bmhName)
				g.Expect(bmh.Spec.UserData).ToNot(BeNil())
				g.Expect(bmh.Spec.UserData.Name).To(Equal(customUserDataSecret.Name))
				g.Expect(bmh.Spec.NetworkData).ToNot(BeNil())
				g.Expect(bmh.Spec.NetworkData.Name).To(Equal(customNetworkDataSecret.Name))
			}, th.Timeout, th.Interval).Should(Succeed())
		})
	})

	When("BMH provisioned with IPv6 control plane", func() {
		BeforeEach(func() {
			DeferCleanup(th.DeleteInstance, CreateBaremetalHost(bmhName))
			bmh := GetBaremetalHost(bmhName)
			Eventually(func(g Gomega) {
				bmh.Status.Provisioning.State = metal3v1.StateAvailable
				g.Expect(th.K8sClient.Status().Update(th.Ctx, bmh)).To(Succeed())
			}, th.Timeout, th.Interval).Should(Succeed())

			DeferCleanup(th.DeleteInstance, CreateSSHSecret(deploymentSecretName))

			// Create baremetalset with IPv6 address
			spec := DefaultBaremetalSetSpec(bmhName, true)
			// Update the baremetalHosts to use IPv6
			spec["baremetalHosts"].(map[string]any)["compute-0"].(map[string]any)["ctlPlaneIP"] = "fd00:1::10/64"
			spec["ctlplaneGateway"] = "fd00:1::1"
			DeferCleanup(th.DeleteInstance, CreateBaremetalSet(baremetalSetName, spec))

			// Wait for provision server to be created, then patch it to have LocalImageURL set
			Eventually(func(g Gomega) {
				provServer := GetProvisionServer(baremetalSetName)
				g.Expect(provServer).ToNot(BeNil())
				provServer.Status.LocalImageURL = "http://192.168.1.100:6190/images/edpm-hardened-uefi.qcow2"
				provServer.Status.LocalImageChecksumURL = "http://192.168.1.100:6190/images/edpm-hardened-uefi.qcow2.md5sum"
				provServer.Status.OSImageChecksumType = metal3v1.MD5
				g.Expect(th.K8sClient.Status().Update(th.Ctx, provServer)).To(Succeed())
			}, th.Timeout, th.Interval).Should(Succeed())
		})

		It("Should generate networkdata with IPv6 configuration", func() {
			Eventually(func(g Gomega) {
				baremetalSet := GetBaremetalSet(baremetalSetName)
				g.Expect(baremetalSet.Status.BaremetalHosts).To(HaveKey("compute-0"))
				g.Expect(baremetalSet.Status.BaremetalHosts["compute-0"].NetworkDataSecretName).ToNot(BeEmpty())
			}, th.Timeout, th.Interval).Should(Succeed())

			baremetalSet := GetBaremetalSet(baremetalSetName)
			networkDataSecretName := baremetalSet.Status.BaremetalHosts["compute-0"].NetworkDataSecretName

			networkDataSecret := th.GetSecret(types.NamespacedName{
				Name:      networkDataSecretName,
				Namespace: bmhName.Namespace,
			})
			networkData := string(networkDataSecret.Data["networkData"])
			Expect(networkData).To(ContainSubstring("type: ipv6"))
			Expect(networkData).To(ContainSubstring("ip_address: fd00:1::10"))
			Expect(networkData).To(ContainSubstring("gateway: fd00:1::1"))
			Expect(networkData).To(ContainSubstring("network: \"::\""))
		})
	})

	When("BMH provisioned with domain name", func() {
		BeforeEach(func() {
			DeferCleanup(th.DeleteInstance, CreateBaremetalHost(bmhName))
			bmh := GetBaremetalHost(bmhName)
			Eventually(func(g Gomega) {
				bmh.Status.Provisioning.State = metal3v1.StateAvailable
				g.Expect(th.K8sClient.Status().Update(th.Ctx, bmh)).To(Succeed())
			}, th.Timeout, th.Interval).Should(Succeed())

			DeferCleanup(th.DeleteInstance, CreateSSHSecret(deploymentSecretName))

			// Create baremetalset with domain name
			spec := DefaultBaremetalSetSpec(bmhName, true)
			spec["domainName"] = "example.com"
			DeferCleanup(th.DeleteInstance, CreateBaremetalSet(baremetalSetName, spec))

			// Wait for provision server to be created, then patch it to have LocalImageURL set
			Eventually(func(g Gomega) {
				provServer := GetProvisionServer(baremetalSetName)
				g.Expect(provServer).ToNot(BeNil())
				provServer.Status.LocalImageURL = "http://192.168.1.100:6190/images/edpm-hardened-uefi.qcow2"
				provServer.Status.LocalImageChecksumURL = "http://192.168.1.100:6190/images/edpm-hardened-uefi.qcow2.md5sum"
				provServer.Status.OSImageChecksumType = metal3v1.MD5
				g.Expect(th.K8sClient.Status().Update(th.Ctx, provServer)).To(Succeed())
			}, th.Timeout, th.Interval).Should(Succeed())
		})

		It("Should generate userdata with FQDN", func() {
			Eventually(func(g Gomega) {
				baremetalSet := GetBaremetalSet(baremetalSetName)
				g.Expect(baremetalSet.Status.BaremetalHosts).To(HaveKey("compute-0"))
				g.Expect(baremetalSet.Status.BaremetalHosts["compute-0"].UserDataSecretName).ToNot(BeEmpty())
			}, th.Timeout, th.Interval).Should(Succeed())

			baremetalSet := GetBaremetalSet(baremetalSetName)
			userDataSecretName := baremetalSet.Status.BaremetalHosts["compute-0"].UserDataSecretName

			userDataSecret := th.GetSecret(types.NamespacedName{
				Name:      userDataSecretName,
				Namespace: bmhName.Namespace,
			})
			userData := string(userDataSecret.Data["userData"])
			Expect(userData).To(ContainSubstring("fqdn: compute-0.example.com"))
		})
	})

	When("BMH provisioning fails with invalid CIDR", func() {
		BeforeEach(func() {
			DeferCleanup(th.DeleteInstance, CreateBaremetalHost(bmhName))
			bmh := GetBaremetalHost(bmhName)
			Eventually(func(g Gomega) {
				bmh.Status.Provisioning.State = metal3v1.StateAvailable
				g.Expect(th.K8sClient.Status().Update(th.Ctx, bmh)).To(Succeed())
			}, th.Timeout, th.Interval).Should(Succeed())

			DeferCleanup(th.DeleteInstance, CreateSSHSecret(deploymentSecretName))

			// Create baremetalset with invalid CIDR (missing network prefix)
			spec := DefaultBaremetalSetSpec(bmhName, true)
			spec["baremetalHosts"] = map[string]any{
				"compute-0": map[string]any{
					"ctlPlaneIP": "10.0.0.1", // Invalid: missing /24
				},
			}
			DeferCleanup(th.DeleteInstance, CreateBaremetalSet(baremetalSetName, spec))
		})

		It("Should report error condition for invalid CIDR", func() {
			th.ExpectCondition(
				baremetalSetName,
				ConditionGetterFunc(BaremetalSetConditionGetter),
				condition.ReadyCondition,
				corev1.ConditionFalse,
			)
		})
	})

	When("BMH provisioning fails without deployment secret", func() {
		BeforeEach(func() {
			DeferCleanup(th.DeleteInstance, CreateBaremetalHost(bmhName))
			bmh := GetBaremetalHost(bmhName)
			Eventually(func(g Gomega) {
				bmh.Status.Provisioning.State = metal3v1.StateAvailable
				g.Expect(th.K8sClient.Status().Update(th.Ctx, bmh)).To(Succeed())
			}, th.Timeout, th.Interval).Should(Succeed())

			// Don't create the deployment secret
			spec := DefaultBaremetalSetSpec(bmhName, true)
			DeferCleanup(th.DeleteInstance, CreateBaremetalSet(baremetalSetName, spec))
		})

		It("Should report InputReady as False", func() {
			th.ExpectCondition(
				baremetalSetName,
				ConditionGetterFunc(BaremetalSetConditionGetter),
				condition.InputReadyCondition,
				corev1.ConditionFalse,
			)
		})

		It("Should not provision BMH without secret", func() {
			Consistently(func(g Gomega) {
				bmh := GetBaremetalHost(bmhName)
				// BMH should not have ConsumerRef set
				g.Expect(bmh.Spec.ConsumerRef).To(BeNil())
			}, "5s", "1s").Should(Succeed())
		})
	})

	When("BMH provisioning without provision server ready", func() {
		BeforeEach(func() {
			DeferCleanup(th.DeleteInstance, CreateBaremetalHost(bmhName))
			bmh := GetBaremetalHost(bmhName)
			Eventually(func(g Gomega) {
				bmh.Status.Provisioning.State = metal3v1.StateAvailable
				g.Expect(th.K8sClient.Status().Update(th.Ctx, bmh)).To(Succeed())
			}, th.Timeout, th.Interval).Should(Succeed())

			DeferCleanup(th.DeleteInstance, CreateSSHSecret(deploymentSecretName))
			DeferCleanup(th.DeleteInstance, CreateBaremetalSet(baremetalSetName, DefaultBaremetalSetSpec(bmhName, true)))

			// Don't patch the provision server with LocalImageURL
		})

		It("Should report provision server not ready", func() {
			th.ExpectCondition(
				baremetalSetName,
				ConditionGetterFunc(BaremetalSetConditionGetter),
				baremetalv1.OpenStackBaremetalSetProvServerReadyCondition,
				corev1.ConditionFalse,
			)
		})

		It("Should not provision BMH without provision server ready", func() {
			Consistently(func(g Gomega) {
				bmh := GetBaremetalHost(bmhName)
				// BMH should not have Image set
				g.Expect(bmh.Spec.Image).To(BeNil())
			}, "5s", "1s").Should(Succeed())
		})
	})

	When("BMH with invalid IPv6 CIDR format", func() {
		BeforeEach(func() {
			DeferCleanup(th.DeleteInstance, CreateBaremetalHost(bmhName))
			bmh := GetBaremetalHost(bmhName)
			Eventually(func(g Gomega) {
				bmh.Status.Provisioning.State = metal3v1.StateAvailable
				g.Expect(th.K8sClient.Status().Update(th.Ctx, bmh)).To(Succeed())
			}, th.Timeout, th.Interval).Should(Succeed())

			DeferCleanup(th.DeleteInstance, CreateSSHSecret(deploymentSecretName))

			// Create baremetalset with malformed IPv6 address
			spec := DefaultBaremetalSetSpec(bmhName, true)
			spec["baremetalHosts"].(map[string]any)["compute-0"].(map[string]any)["ctlPlaneIP"] = "fd00:1::10::20/64" // Invalid: double ::
			DeferCleanup(th.DeleteInstance, CreateBaremetalSet(baremetalSetName, spec))
		})

		It("Should report error for invalid IPv6 format", func() {
			th.ExpectCondition(
				baremetalSetName,
				ConditionGetterFunc(BaremetalSetConditionGetter),
				condition.ReadyCondition,
				corev1.ConditionFalse,
			)
		})
	})

	When("A BaremetalSet with bonding configuration generates network data", func() {
		BeforeEach(func() {
			DeferCleanup(th.DeleteInstance, CreateBaremetalHost(bmhName))
			bmh := GetBaremetalHost(bmhName)
			Eventually(func(g Gomega) {
				bmh.Status.Provisioning.State = metal3v1.StateAvailable
				g.Expect(th.K8sClient.Status().Update(th.Ctx, bmh)).To(Succeed())
			}, th.Timeout, th.Interval).Should(Succeed())

			DeferCleanup(th.DeleteInstance, CreateSSHSecret(deploymentSecretName))
		})

		// Helper to patch provision server to be ready
		patchProvisionServerReady := func() {
			provServerName := types.NamespacedName{
				Name:      strings.Join([]string{baremetalSetName.Name, "provisionserver"}, "-"),
				Namespace: namespace,
			}

			// Wait for provision server to be created by the controller
			// and patch it to be ready
			Eventually(func(g Gomega) {
				provServer := &baremetalv1.OpenStackProvisionServer{}
				g.Expect(th.K8sClient.Get(th.Ctx, provServerName, provServer)).To(Succeed())
				provServer.Status.ProvisionIP = "192.168.122.100"
				provServer.Status.LocalImageURL = "http://192.168.122.100:6190/images/edpm-hardened-uefi.qcow2"
				provServer.Status.LocalImageChecksumURL = "http://192.168.122.100:6190/images/edpm-hardened-uefi.qcow2.sha256sum"
				provServer.Status.ReadyCount = 1

				g.Expect(th.K8sClient.Status().Update(th.Ctx, provServer)).To(Succeed())
			}, th.Timeout, th.Interval).Should(Succeed())
		}

		It("Should generate networkdata secret with bonding configuration", func() {
			bondSpec := DefaultBaremetalSetSpec(bmhName, true) // Need provision server
			bondSpec["ctlplaneInterface"] = "bond0"
			bondSpec["ctlplaneBond"] = map[string]any{
				"bondInterfaces": []string{"eno1", "eno2"},
				"bondMode":       "active-backup",
			}
			DeferCleanup(th.DeleteInstance, CreateBaremetalSet(baremetalSetName, bondSpec))

			// Verify default was applied
			baremetalSetInstance := GetBaremetalSet(baremetalSetName)
			Expect(baremetalSetInstance.Spec.CtlplaneBond.BondMode).Should(Equal("active-backup"))

			//patch provision server to be ready
			patchProvisionServerReady()

			// Wait for network data secret with bonding config
			Eventually(func(g Gomega) {
				secretName := types.NamespacedName{
					Name:      strings.Join([]string{baremetalSetName.Name, "cloudinit-networkdata", "compute-0"}, "-"),
					Namespace: namespace,
				}
				secret := &corev1.Secret{}
				g.Expect(th.K8sClient.Get(th.Ctx, secretName, secret)).To(Succeed())

				networkData := string(secret.Data["networkData"])
				// Verify proper YAML formatting
				g.Expect(networkData).To(MatchRegexp("(?m)^links:\n"))
				g.Expect(networkData).NotTo(ContainSubstring("links:- "))
				g.Expect(networkData).To(MatchRegexp("(?m)^  bond_interfaces:\n"))
				g.Expect(networkData).NotTo(ContainSubstring("bond_interfaces:    - "))
				g.Expect(networkData).To(MatchRegexp("bond_mode: [^\n]+\n  "))
				g.Expect(networkData).NotTo(MatchRegexp("bond_mode: [^\n]+  params:"))
				g.Expect(networkData).Should(ContainSubstring("type: bond"))
				g.Expect(networkData).Should(ContainSubstring("bond_mode: active-backup"))
				g.Expect(networkData).Should(ContainSubstring("eno1"))
				g.Expect(networkData).Should(ContainSubstring("eno2"))
			}, th.Timeout, th.Interval).Should(Succeed())
		})

		It("Should reject bonding with less than 2 interfaces", func() {
			bondSpec := DefaultBaremetalSetSpec(bmhName, true)
			bondSpec["ctlplaneInterface"] = "bond0"
			bondSpec["ctlplaneBond"] = map[string]any{
				"bondInterfaces": []string{"eno1"}, // Only one interface - should fail
				"bondMode":       "active-backup",
			}

			object := DefaultBaremetalSetTemplate(baremetalSetName, bondSpec)
			unstructuredObj := &unstructured.Unstructured{Object: object}
			_, err := controllerutil.CreateOrPatch(
				th.Ctx, th.K8sClient, unstructuredObj, func() error { return nil })
			Expect(err).Should(HaveOccurred())
			Expect(err.Error()).Should(ContainSubstring("bondInterfaces"))
		})

		It("Should generate networkdata without bonding (backward compatibility)", func() {
			noBondSpec := DefaultBaremetalSetSpec(bmhName, true)
			noBondSpec["ctlplaneInterface"] = "eth0"

			DeferCleanup(th.DeleteInstance, CreateBaremetalSet(baremetalSetName, noBondSpec))

			// Verify provision server to be ready
			patchProvisionServerReady()

			// Wait for network data secret without bonding
			Eventually(func(g Gomega) {
				secretName := types.NamespacedName{
					Name:      strings.Join([]string{baremetalSetName.Name, "cloudinit-networkdata", "compute-0"}, "-"),
					Namespace: namespace,
				}
				secret := &corev1.Secret{}
				g.Expect(th.K8sClient.Get(th.Ctx, secretName, secret)).To(Succeed())

				networkData := string(secret.Data["networkData"])
				g.Expect(networkData).ShouldNot(ContainSubstring("type: bond"))
				g.Expect(networkData).Should(ContainSubstring("type: vif"))
			}, th.Timeout, th.Interval).Should(Succeed())
		})

		It("Should generate networkdata with bonding and VLAN", func() {
			bondVlanSpec := DefaultBaremetalSetSpec(bmhName, true)
			bondVlanSpec["ctlplaneInterface"] = "bond0"
			bondVlanSpec["ctlplaneVlan"] = 100
			bondVlanSpec["ctlplaneBond"] = map[string]any{
				"bondInterfaces": []string{"eno1", "eno2"},
				"bondMode":       "802.3ad",
			}

			DeferCleanup(th.DeleteInstance, CreateBaremetalSet(baremetalSetName, bondVlanSpec))

			// Verify provision server to be ready
			patchProvisionServerReady()

			// Wait for network data secret with bonding + VLAN
			Eventually(func(g Gomega) {
				secretName := types.NamespacedName{
					Name:      strings.Join([]string{baremetalSetName.Name, "cloudinit-networkdata", "compute-0"}, "-"),
					Namespace: namespace,
				}
				secret := &corev1.Secret{}
				g.Expect(th.K8sClient.Get(th.Ctx, secretName, secret)).To(Succeed())

				networkData := string(secret.Data["networkData"])
				g.Expect(networkData).Should(ContainSubstring("type: bond"))
				g.Expect(networkData).Should(ContainSubstring("bond_mode: 802.3ad"))
				g.Expect(networkData).Should(ContainSubstring("type: vlan"))
				g.Expect(networkData).Should(ContainSubstring("vlan_id: 100"))
				g.Expect(networkData).Should(ContainSubstring("bond0.100"))
			}, th.Timeout, th.Interval).Should(Succeed())
		})
	})
})
