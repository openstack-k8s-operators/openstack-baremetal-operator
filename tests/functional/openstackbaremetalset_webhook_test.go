package functional

import (
	"errors"

	metal3v1 "github.com/metal3-io/baremetal-operator/apis/metal3.io/v1alpha1"
	. "github.com/onsi/ginkgo/v2" //revive:disable:dot-imports
	. "github.com/onsi/gomega"    //revive:disable:dot-imports
	v1 "k8s.io/api/core/v1"
	k8s_errors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/types"

	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

var (
	// ErrConflictRetry is used in Eventually blocks to signal retry on resource conflicts
	ErrConflictRetry = errors.New("conflict error, will retry")
)

var _ = Describe("OpenStackBaremetalSet Webhook", func() {

	var baremetalSetName types.NamespacedName
	var bmhName types.NamespacedName
	var bmhName1 types.NamespacedName
	var bmhName2 types.NamespacedName

	BeforeEach(func() {
		baremetalSetName = types.NamespacedName{
			Name:      "edpm-compute-baremetalset",
			Namespace: namespace,
		}
		bmhName = types.NamespacedName{
			Name:      "compute-0",
			Namespace: namespace,
		}
		bmhName1 = types.NamespacedName{
			Name:      "compute-1",
			Namespace: namespace,
		}
		bmhName2 = types.NamespacedName{
			Name:      "compute-3",
			Namespace: namespace,
		}

	})
	When("When creating BaremetalSet", func() {
		BeforeEach(func() {
			DeferCleanup(th.DeleteInstance, CreateBaremetalHost(bmhName))
			bmh := GetBaremetalHost(bmhName)
			Eventually(func(g Gomega) {
				// OpenStackBaremetalSet has the same name as OpenStackDataPlaneNodeSet
				bmh.Status.Provisioning.State = metal3v1.StateAvailable
				g.Expect(th.K8sClient.Status().Update(th.Ctx, bmh)).To(Succeed())

			}, th.Timeout, th.Interval).Should(Succeed())

		})
		It("It should not fail if enough bmhs are available", func() {
			spec := DefaultBaremetalSetSpec(baremetalSetName, false)
			object := DefaultBaremetalSetTemplate(baremetalSetName, spec)
			unstructuredObj := &unstructured.Unstructured{Object: object}
			_, err := controllerutil.CreateOrPatch(
				th.Ctx, th.K8sClient, unstructuredObj, func() error { return nil })
			Expect(err).ShouldNot(HaveOccurred())
		})

		It("It should fail if not enough bmhs are available", func() {
			spec := TwoNodeBaremetalSetSpec(baremetalSetName.Namespace)
			object := DefaultBaremetalSetTemplate(baremetalSetName, spec)
			unstructuredObj := &unstructured.Unstructured{Object: object}
			_, err := controllerutil.CreateOrPatch(
				th.Ctx, th.K8sClient, unstructuredObj, func() error { return nil })
			Expect(err).Should(HaveOccurred())
			var statusError *k8s_errors.StatusError
			Expect(errors.As(err, &statusError)).To(BeTrue())
			Expect(statusError.ErrStatus.Message).To(
				ContainSubstring(
					"unable to find 2 requested BaremetalHosts"),
			)
		})

		It("It should fail when suffiecient BMHs are not offine", func() {
			bmh := GetBaremetalHost(bmhName)
			Eventually(func(g Gomega) {
				// OpenStackBaremetalSet has the same name as OpenStackDataPlaneNodeSet
				bmh.Spec.Online = true
				g.Expect(th.K8sClient.Status().Update(th.Ctx, bmh)).To(Succeed())

			}, th.Timeout, th.Interval).Should(Succeed())
			spec := DefaultBaremetalSetSpec(baremetalSetName, false)
			object := DefaultBaremetalSetTemplate(baremetalSetName, spec)
			unstructuredObj := &unstructured.Unstructured{Object: object}
			_, err := controllerutil.CreateOrPatch(
				th.Ctx, th.K8sClient, unstructuredObj, func() error { return nil })
			Expect(err).ShouldNot(HaveOccurred())
		})
		It("It should fail when all BMHs have consumerRef", func() {
			bmh := GetBaremetalHost(bmhName)
			Eventually(func(g Gomega) {
				// OpenStackBaremetalSet has the same name as OpenStackDataPlaneNodeSet
				bmh.Spec.ConsumerRef = &v1.ObjectReference{}
				g.Expect(th.K8sClient.Status().Update(th.Ctx, bmh)).To(Succeed())

			}, th.Timeout, th.Interval).Should(Succeed())
			spec := DefaultBaremetalSetSpec(baremetalSetName, false)
			object := DefaultBaremetalSetTemplate(baremetalSetName, spec)
			unstructuredObj := &unstructured.Unstructured{Object: object}
			_, err := controllerutil.CreateOrPatch(
				th.Ctx, th.K8sClient, unstructuredObj, func() error { return nil })
			Expect(err).ShouldNot(HaveOccurred())
		})
	})

	When("When creating BaremetalSet with a node selector", func() {
		BeforeEach(func() {
			DeferCleanup(th.DeleteInstance, CreateBaremetalHostWithNodeLabel(
				bmhName, map[string]string{"nodeName": "compute-0"}))
			bmh := GetBaremetalHost(bmhName)
			DeferCleanup(th.DeleteInstance, CreateBaremetalHostWithNodeLabel(
				bmhName1, map[string]string{"nodeName": "compute-1"}))
			bmh1 := GetBaremetalHost(bmhName1)
			Eventually(func(g Gomega) {
				// OpenStackBaremetalSet has the same name as OpenStackDataPlaneNodeSet
				bmh.Status.Provisioning.State = metal3v1.StateAvailable
				bmh1.Status.Provisioning.State = metal3v1.StateAvailable
				g.Expect(th.K8sClient.Status().Update(th.Ctx, bmh)).To(Succeed())
				g.Expect(th.K8sClient.Status().Update(th.Ctx, bmh1)).To(Succeed())
			}, th.Timeout, th.Interval).Should(Succeed())

		})

		It("It should pass if node labels match", func() {
			spec := TwoNodeBaremetalSetSpecWithNodeLabel(baremetalSetName.Namespace)
			object := DefaultBaremetalSetTemplate(baremetalSetName, spec)
			unstructuredObj := &unstructured.Unstructured{Object: object}
			_, err := controllerutil.CreateOrPatch(
				th.Ctx, th.K8sClient, unstructuredObj, func() error { return nil })
			Expect(err).ShouldNot(HaveOccurred())
		})

		It("It should fail if node labels don't match", func() {
			spec := TwoNodeBaremetalSetSpecWithWrongNodeLabel(baremetalSetName.Namespace)
			object := DefaultBaremetalSetTemplate(baremetalSetName, spec)
			unstructuredObj := &unstructured.Unstructured{Object: object}
			_, err := controllerutil.CreateOrPatch(
				th.Ctx, th.K8sClient, unstructuredObj, func() error { return nil })
			Expect(err).Should(HaveOccurred())
			var statusError *k8s_errors.StatusError
			Expect(errors.As(err, &statusError)).To(BeTrue())
			Expect(statusError.ErrStatus.Message).To(
				ContainSubstring(
					"unable to find 2 requested BaremetalHosts"),
			)
		})
	})

	When("When creating BaremetalSet with node selectors", func() {
		BeforeEach(func() {
			DeferCleanup(th.DeleteInstance, CreateBaremetalHostWithNodeLabel(
				bmhName, map[string]string{"nodeType": "compute", "dummyLabel": "dummy", "nodeName": "compute-1"}))
			bmh := GetBaremetalHost(bmhName)
			DeferCleanup(th.DeleteInstance, CreateBaremetalHostWithNodeLabel(
				bmhName1, map[string]string{"nodeType": "compute", "nodeName": "compute-0", "dummyLabel": "dummy"}))
			bmh1 := GetBaremetalHost(bmhName1)
			DeferCleanup(th.DeleteInstance, CreateBaremetalHostWithNodeLabel(
				bmhName2, map[string]string{"nodeType": "compute", "dummyLabel": "dummy"}))
			bmh2 := GetBaremetalHost(bmhName2)

			Eventually(func(g Gomega) {
				// OpenStackBaremetalSet has the same name as OpenStackDataPlaneNodeSet
				bmh.Status.Provisioning.State = metal3v1.StateAvailable
				bmh1.Status.Provisioning.State = metal3v1.StateAvailable
				bmh2.Status.Provisioning.State = metal3v1.StateAvailable
				g.Expect(th.K8sClient.Status().Update(th.Ctx, bmh)).To(Succeed())
				g.Expect(th.K8sClient.Status().Update(th.Ctx, bmh1)).To(Succeed())
				g.Expect(th.K8sClient.Status().Update(th.Ctx, bmh2)).To(Succeed())
			}, th.Timeout, th.Interval).Should(Succeed())

		})

		It("It should pass if node labels are same", func() {
			spec := MultiNodeBaremetalSetSpecWithSameNodeLabel(baremetalSetName.Namespace)
			object := DefaultBaremetalSetTemplate(baremetalSetName, spec)
			unstructuredObj := &unstructured.Unstructured{Object: object}
			_, err := controllerutil.CreateOrPatch(
				th.Ctx, th.K8sClient, unstructuredObj, func() error { return nil })
			Expect(err).ShouldNot(HaveOccurred())
		})

		It("It should pass if overlapping node labels match", func() {
			spec := MultiNodeBaremetalSetSpecWithOverlappingNodeLabels(baremetalSetName.Namespace)
			object := DefaultBaremetalSetTemplate(baremetalSetName, spec)
			unstructuredObj := &unstructured.Unstructured{Object: object}
			_, err := controllerutil.CreateOrPatch(
				th.Ctx, th.K8sClient, unstructuredObj, func() error { return nil })
			Expect(err).ShouldNot(HaveOccurred())
		})

	})

	When("When validating BaremetalSet name", func() {
		It("should reject invalid RFC1123 names", func() {
			// Create required BareMetalHosts first so validation can proceed to name check
			bmhName := types.NamespacedName{Name: "compute-0", Namespace: namespace}
			DeferCleanup(th.DeleteInstance, CreateBaremetalHost(bmhName))

			invalidNames := []string{
				"Invalid_Name_With_Underscores",
				"name-with-UPPERCASE",
				"name with spaces",
				"-invalid-start-with-dash",
				"invalid-end-with-dash-",
			}

			for _, invalidName := range invalidNames {
				baremetalSetName := types.NamespacedName{
					Name:      invalidName,
					Namespace: namespace,
				}
				spec := DefaultBaremetalSetSpec(baremetalSetName, false)
				object := DefaultBaremetalSetTemplate(baremetalSetName, spec)
				unstructuredObj := &unstructured.Unstructured{Object: object}

				_, err := controllerutil.CreateOrPatch(
					th.Ctx, th.K8sClient, unstructuredObj, func() error { return nil })
				Expect(err).Should(HaveOccurred())
				var statusError *k8s_errors.StatusError
				Expect(errors.As(err, &statusError)).To(BeTrue())
				Expect(statusError.ErrStatus.Message).To(
					ContainSubstring("lowercase RFC 1123 subdomain"))
			}
		})

		It("should accept valid RFC1123 names", func() {
			DeferCleanup(th.DeleteInstance, CreateBaremetalHost(bmhName))
			bmh := GetBaremetalHost(bmhName)
			Eventually(func(g Gomega) {
				bmh.Status.Provisioning.State = metal3v1.StateAvailable
				g.Expect(th.K8sClient.Status().Update(th.Ctx, bmh)).To(Succeed())
			}, th.Timeout, th.Interval).Should(Succeed())

			validNames := []string{
				"valid-name",
				"name123",
				"a-b-c",
				"test-compute-set",
			}

			for _, validName := range validNames {
				baremetalSetName := types.NamespacedName{
					Name:      validName,
					Namespace: namespace,
				}
				spec := DefaultBaremetalSetSpec(baremetalSetName, false)
				object := DefaultBaremetalSetTemplate(baremetalSetName, spec)
				unstructuredObj := &unstructured.Unstructured{Object: object}

				_, err := controllerutil.CreateOrPatch(
					th.Ctx, th.K8sClient, unstructuredObj, func() error { return nil })
				Expect(err).ShouldNot(HaveOccurred())

				// Clean up after each test
				th.DeleteInstance(unstructuredObj)
			}
		})
	})

	When("When validating cloud-init secrets", func() {
		var wrongNamespace string

		BeforeEach(func() {
			wrongNamespace = "wrong-namespace"
			DeferCleanup(th.DeleteInstance, CreateBaremetalHost(bmhName))
			bmh := GetBaremetalHost(bmhName)
			Eventually(func(g Gomega) {
				bmh.Status.Provisioning.State = metal3v1.StateAvailable
				g.Expect(th.K8sClient.Status().Update(th.Ctx, bmh)).To(Succeed())
			}, th.Timeout, th.Interval).Should(Succeed())
		})

		It("should reject userData secret in wrong namespace", func() {
			spec := DefaultBaremetalSetSpec(baremetalSetName, false)
			// Set userData secret to wrong namespace
			for hostName := range spec["baremetalHosts"].(map[string]interface{}) {
				hostSpec := spec["baremetalHosts"].(map[string]interface{})[hostName].(map[string]interface{})
				hostSpec["userData"] = map[string]interface{}{
					"name":      "user-data-secret",
					"namespace": wrongNamespace,
				}
			}

			object := DefaultBaremetalSetTemplate(baremetalSetName, spec)
			unstructuredObj := &unstructured.Unstructured{Object: object}

			_, err := controllerutil.CreateOrPatch(
				th.Ctx, th.K8sClient, unstructuredObj, func() error { return nil })
			Expect(err).Should(HaveOccurred())
			var statusError *k8s_errors.StatusError
			Expect(errors.As(err, &statusError)).To(BeTrue())
			Expect(statusError.ErrStatus.Message).To(
				ContainSubstring("should exist in the bmh namespace"))
		})

		It("should reject networkData secret in wrong namespace", func() {
			spec := DefaultBaremetalSetSpec(baremetalSetName, false)
			// Set networkData secret to wrong namespace
			for hostName := range spec["baremetalHosts"].(map[string]interface{}) {
				hostSpec := spec["baremetalHosts"].(map[string]interface{})[hostName].(map[string]interface{})
				hostSpec["networkData"] = map[string]interface{}{
					"name":      "network-data-secret",
					"namespace": wrongNamespace,
				}
			}

			object := DefaultBaremetalSetTemplate(baremetalSetName, spec)
			unstructuredObj := &unstructured.Unstructured{Object: object}

			_, err := controllerutil.CreateOrPatch(
				th.Ctx, th.K8sClient, unstructuredObj, func() error { return nil })
			Expect(err).Should(HaveOccurred())
			var statusError *k8s_errors.StatusError
			Expect(errors.As(err, &statusError)).To(BeTrue())
			Expect(statusError.ErrStatus.Message).To(
				ContainSubstring("should exist in the bmh namespace"))
		})
	})

	When("When updating BaremetalSet", func() {
		var baremetalSet *unstructured.Unstructured

		BeforeEach(func() {
			DeferCleanup(th.DeleteInstance, CreateBaremetalHost(bmhName))
			DeferCleanup(th.DeleteInstance, CreateBaremetalHost(bmhName1))

			bmh := GetBaremetalHost(bmhName)
			bmh1 := GetBaremetalHost(bmhName1)
			Eventually(func(g Gomega) {
				bmh.Status.Provisioning.State = metal3v1.StateAvailable
				bmh1.Status.Provisioning.State = metal3v1.StateAvailable
				g.Expect(th.K8sClient.Status().Update(th.Ctx, bmh)).To(Succeed())
				g.Expect(th.K8sClient.Status().Update(th.Ctx, bmh1)).To(Succeed())
			}, th.Timeout, th.Interval).Should(Succeed())

			// Create initial BaremetalSet
			spec := DefaultBaremetalSetSpec(baremetalSetName, false)
			object := DefaultBaremetalSetTemplate(baremetalSetName, spec)
			baremetalSet = &unstructured.Unstructured{Object: object}
			DeferCleanup(th.DeleteInstance, baremetalSet)

			_, err := controllerutil.CreateOrPatch(
				th.Ctx, th.K8sClient, baremetalSet, func() error { return nil })
			Expect(err).ShouldNot(HaveOccurred())
		})

		It("should allow scaling up when enough BMHs available", func() {
			// Scale up to 2 nodes
			spec := TwoNodeBaremetalSetSpec(baremetalSetName.Namespace)
			baremetalSet.Object = DefaultBaremetalSetTemplate(baremetalSetName, spec)

			_, err := controllerutil.CreateOrPatch(
				th.Ctx, th.K8sClient, baremetalSet, func() error { return nil })
			Expect(err).ShouldNot(HaveOccurred())
		})

		It("should prevent changing bmhLabelSelector when count > 0", func() {
			// Retry to handle version conflicts and get to the actual validation
			Eventually(func(g Gomega) {
				existingObj := &unstructured.Unstructured{}
				existingObj.SetAPIVersion("baremetal.openstack.org/v1beta1")
				existingObj.SetKind("OpenStackBaremetalSet")
				g.Expect(th.K8sClient.Get(th.Ctx, baremetalSetName, existingObj)).To(Succeed())

				// Modify just the bmhLabelSelector
				currentSpec := existingObj.Object["spec"].(map[string]interface{})
				currentSpec["bmhLabelSelector"] = map[string]interface{}{
					"newLabel": "newValue",
				}

				// Try update - should get validation error, not conflict
				err := th.K8sClient.Update(th.Ctx, existingObj)
				g.Expect(err).Should(HaveOccurred())

				// Ensure this is a validation error (webhook), not a conflict error
				var statusError *k8s_errors.StatusError
				g.Expect(errors.As(err, &statusError)).To(BeTrue())

				// Skip conflicts - we want validation errors only
				if statusError.ErrStatus.Reason == "Conflict" {
					g.Expect(ErrConflictRetry).ToNot(HaveOccurred())
				}

				// Should be validation error from webhook
				g.Expect(statusError.ErrStatus.Message).To(
					ContainSubstring("cannot change \"bmhLabelSelector\""))
			}, th.Timeout, th.Interval).Should(Succeed())
		})

		It("should prevent changing hardwareReqs when count > 0", func() {
			// Retry to handle version conflicts and get to the actual validation
			Eventually(func(g Gomega) {
				existingObj := &unstructured.Unstructured{}
				existingObj.SetAPIVersion("baremetal.openstack.org/v1beta1")
				existingObj.SetKind("OpenStackBaremetalSet")
				g.Expect(th.K8sClient.Get(th.Ctx, baremetalSetName, existingObj)).To(Succeed())

				// Modify just the hardwareReqs
				currentSpec := existingObj.Object["spec"].(map[string]interface{})
				currentSpec["hardwareReqs"] = map[string]interface{}{
					"cpuReqs": map[string]interface{}{
						"countReq": map[string]interface{}{
							"count":      8,
							"exactMatch": false,
						},
					},
				}

				// Try update - should get validation error, not conflict
				err := th.K8sClient.Update(th.Ctx, existingObj)
				g.Expect(err).Should(HaveOccurred())

				// Ensure this is a validation error (webhook), not a conflict error
				var statusError *k8s_errors.StatusError
				g.Expect(errors.As(err, &statusError)).To(BeTrue())

				// Skip conflicts - we want validation errors only
				if statusError.ErrStatus.Reason == "Conflict" {
					g.Expect(ErrConflictRetry).ToNot(HaveOccurred())
				}

				// Should be validation error from webhook
				g.Expect(statusError.ErrStatus.Message).To(
					ContainSubstring("cannot change \"bmhLabelSelector\" nor \"hardwareReqs\""))
			}, th.Timeout, th.Interval).Should(Succeed())
		})
	})

	When("When deleting BaremetalSet", func() {
		It("should allow deletion", func() {
			DeferCleanup(th.DeleteInstance, CreateBaremetalHost(bmhName))
			bmh := GetBaremetalHost(bmhName)
			Eventually(func(g Gomega) {
				bmh.Status.Provisioning.State = metal3v1.StateAvailable
				g.Expect(th.K8sClient.Status().Update(th.Ctx, bmh)).To(Succeed())
			}, th.Timeout, th.Interval).Should(Succeed())

			spec := DefaultBaremetalSetSpec(baremetalSetName, false)
			object := DefaultBaremetalSetTemplate(baremetalSetName, spec)
			baremetalSet := &unstructured.Unstructured{Object: object}

			_, err := controllerutil.CreateOrPatch(
				th.Ctx, th.K8sClient, baremetalSet, func() error { return nil })
			Expect(err).ShouldNot(HaveOccurred())

			// Delete should succeed (webhook just logs and returns nil)
			err = th.K8sClient.Delete(th.Ctx, baremetalSet)
			Expect(err).ShouldNot(HaveOccurred())
		})
	})

})
