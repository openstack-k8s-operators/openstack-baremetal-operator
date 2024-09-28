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

var _ = Describe("OpenStackBaremetalSet Webhook", func() {

	var baremetalSetName types.NamespacedName
	var bmhName types.NamespacedName
	var bmhName1 types.NamespacedName

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

	When("When creating BaremetalSet with nodeSelector", func() {
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
})
