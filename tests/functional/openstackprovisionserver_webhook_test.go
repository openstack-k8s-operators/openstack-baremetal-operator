package functional

import (
	"errors"
	"fmt"

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
			Name:      "test-provision-server",
			Namespace: namespace,
		}
	})

	When("When validating ProvisionServer name", func() {
		It("should reject invalid RFC1123 names", func() {
			invalidNames := []string{
				"Invalid_Name_With_Underscores",
				"name-with-UPPERCASE",
				"name with spaces",
				"name-ending-with-",
			}

			for _, invalidName := range invalidNames {
				provisionServerName := types.NamespacedName{
					Name:      invalidName,
					Namespace: namespace,
				}
				spec := DefaultProvisionServerSpec()
				object := DefaultProvisionServerTemplate(provisionServerName, spec)
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
			validNames := []string{
				"valid-name",
				"name123",
				"a-b-c",
				"test-prov-server",
				"provision-01",
			}

			for _, validName := range validNames {
				provisionServerName := types.NamespacedName{
					Name:      validName,
					Namespace: namespace,
				}
				spec := DefaultProvisionServerSpec()
				object := DefaultProvisionServerTemplate(provisionServerName, spec)
				unstructuredObj := &unstructured.Unstructured{Object: object}

				_, err := controllerutil.CreateOrPatch(
					th.Ctx, th.K8sClient, unstructuredObj, func() error { return nil })
				Expect(err).ShouldNot(HaveOccurred())

				// Clean up after each test
				th.DeleteInstance(unstructuredObj)
			}
		})
	})

	When("When validating port conflicts", func() {
		var firstProvisionServer *unstructured.Unstructured

		BeforeEach(func() {
			// Create first provision server with a specific port
			spec := DefaultProvisionServerSpec()
			spec["port"] = 6200 // Use different port to avoid conflicts
			object := DefaultProvisionServerTemplate(provisionServerName, spec)
			firstProvisionServer = &unstructured.Unstructured{Object: object}
			DeferCleanup(th.DeleteInstance, firstProvisionServer)

			_, err := controllerutil.CreateOrPatch(
				th.Ctx, th.K8sClient, firstProvisionServer, func() error { return nil })
			Expect(err).ShouldNot(HaveOccurred())
		})

		It("should reject duplicate ports", func() {
			// Try to create second provision server with same port
			secondProvisionServerName := types.NamespacedName{
				Name:      "second-provision-server",
				Namespace: namespace,
			}
			spec := DefaultProvisionServerSpec()
			spec["port"] = 6200 // Same port as first server
			object := DefaultProvisionServerTemplate(secondProvisionServerName, spec)
			unstructuredObj := &unstructured.Unstructured{Object: object}

			_, err := controllerutil.CreateOrPatch(
				th.Ctx, th.K8sClient, unstructuredObj, func() error { return nil })
			Expect(err).Should(HaveOccurred())
			var statusError *k8s_errors.StatusError
			Expect(errors.As(err, &statusError)).To(BeTrue())
			Expect(statusError.ErrStatus.Message).To(
				ContainSubstring("port 6200 is already in use"))
		})

		It("should allow different ports", func() {
			// Try to create second provision server with different port
			secondProvisionServerName := types.NamespacedName{
				Name:      "second-provision-server",
				Namespace: namespace,
			}
			spec := DefaultProvisionServerSpec()
			spec["port"] = 6201 // Different port
			object := DefaultProvisionServerTemplate(secondProvisionServerName, spec)
			unstructuredObj := &unstructured.Unstructured{Object: object}
			DeferCleanup(th.DeleteInstance, unstructuredObj)

			_, err := controllerutil.CreateOrPatch(
				th.Ctx, th.K8sClient, unstructuredObj, func() error { return nil })
			Expect(err).ShouldNot(HaveOccurred())
		})
	})

	When("When testing defaulting webhook", func() {
		It("should set default values when not specified", func() {
			spec := map[string]interface{}{
				"osImage": "test-image.qcow2",
				// No port, container URLs, etc. - should get defaults
			}
			object := DefaultProvisionServerTemplate(provisionServerName, spec)
			unstructuredObj := &unstructured.Unstructured{Object: object}
			DeferCleanup(th.DeleteInstance, unstructuredObj)

			_, err := controllerutil.CreateOrPatch(
				th.Ctx, th.K8sClient, unstructuredObj, func() error { return nil })
			Expect(err).ShouldNot(HaveOccurred())

			// Check that defaults were applied
			updatedObj := &unstructured.Unstructured{}
			updatedObj.SetAPIVersion("baremetal.openstack.org/v1beta1")
			updatedObj.SetKind("OpenStackProvisionServer")
			Eventually(func(g Gomega) {
				g.Expect(th.K8sClient.Get(th.Ctx, provisionServerName, updatedObj)).To(Succeed())

				// Check that port was assigned (should be non-zero)
				port, found, err := unstructured.NestedInt64(updatedObj.Object, "spec", "port")
				g.Expect(err).ShouldNot(HaveOccurred())
				g.Expect(found).To(BeTrue())
				g.Expect(port).To(BeNumerically(">", 0))

				// Note: Image URLs might be empty if defaults weren't initialized in test environment
				// This is expected in test environment where defaults may not be configured
			}, th.Timeout, th.Interval).Should(Succeed())
		})

		It("should not override explicitly set values", func() {
			customPort := int64(6208)
			customOSContainerURL := "custom.registry.com/os-container:latest"
			customApacheURL := "custom.registry.com/apache:latest"
			customAgentURL := "custom.registry.com/agent:latest"

			spec := map[string]interface{}{
				"osImage":             "test-image.qcow2",
				"port":                customPort,
				"osContainerImageUrl": customOSContainerURL,
				"apacheImageUrl":      customApacheURL,
				"agentImageUrl":       customAgentURL,
			}
			object := DefaultProvisionServerTemplate(provisionServerName, spec)
			unstructuredObj := &unstructured.Unstructured{Object: object}
			DeferCleanup(th.DeleteInstance, unstructuredObj)

			_, err := controllerutil.CreateOrPatch(
				th.Ctx, th.K8sClient, unstructuredObj, func() error { return nil })
			Expect(err).ShouldNot(HaveOccurred())

			// Check that custom values were preserved
			updatedObj := &unstructured.Unstructured{}
			updatedObj.SetAPIVersion("baremetal.openstack.org/v1beta1")
			updatedObj.SetKind("OpenStackProvisionServer")
			Eventually(func(g Gomega) {
				g.Expect(th.K8sClient.Get(th.Ctx, provisionServerName, updatedObj)).To(Succeed())

				port, found, err := unstructured.NestedInt64(updatedObj.Object, "spec", "port")
				g.Expect(err).ShouldNot(HaveOccurred())
				g.Expect(found).To(BeTrue())
				g.Expect(port).To(Equal(customPort))

				osContainerImageURL, found, err := unstructured.NestedString(updatedObj.Object, "spec", "osContainerImageUrl")
				g.Expect(err).ShouldNot(HaveOccurred())
				g.Expect(found).To(BeTrue())
				g.Expect(osContainerImageURL).To(Equal(customOSContainerURL))

				apacheImageURL, found, err := unstructured.NestedString(updatedObj.Object, "spec", "apacheImageUrl")
				g.Expect(err).ShouldNot(HaveOccurred())
				g.Expect(found).To(BeTrue())
				g.Expect(apacheImageURL).To(Equal(customApacheURL))

				agentImageURL, found, err := unstructured.NestedString(updatedObj.Object, "spec", "agentImageUrl")
				g.Expect(err).ShouldNot(HaveOccurred())
				g.Expect(found).To(BeTrue())
				g.Expect(agentImageURL).To(Equal(customAgentURL))
			}, th.Timeout, th.Interval).Should(Succeed())
		})
	})

	When("When updating ProvisionServer", func() {
		var provisionServer *unstructured.Unstructured

		BeforeEach(func() {
			spec := DefaultProvisionServerSpec()
			spec["port"] = 6202 // Use different port to avoid conflicts
			object := DefaultProvisionServerTemplate(provisionServerName, spec)
			provisionServer = &unstructured.Unstructured{Object: object}
			DeferCleanup(th.DeleteInstance, provisionServer)

			_, err := controllerutil.CreateOrPatch(
				th.Ctx, th.K8sClient, provisionServer, func() error { return nil })
			Expect(err).ShouldNot(HaveOccurred())
		})

		It("should allow updating non-port fields", func() {
			// Update osImage field
			newSpec := DefaultProvisionServerSpec()
			newSpec["port"] = 6202 // Keep same port
			newSpec["osImage"] = "updated-image.qcow2"
			provisionServer.Object = DefaultProvisionServerTemplate(provisionServerName, newSpec)

			_, err := controllerutil.CreateOrPatch(
				th.Ctx, th.K8sClient, provisionServer, func() error { return nil })
			Expect(err).ShouldNot(HaveOccurred())
		})

		It("should prevent port conflicts during updates", func() {
			// Create another provision server with port 6203
			anotherProvisionServerName := types.NamespacedName{
				Name:      "another-provision-server",
				Namespace: namespace,
			}
			anotherSpec := DefaultProvisionServerSpec()
			anotherSpec["port"] = 6203
			anotherObject := DefaultProvisionServerTemplate(anotherProvisionServerName, anotherSpec)
			anotherProvisionServer := &unstructured.Unstructured{Object: anotherObject}
			DeferCleanup(th.DeleteInstance, anotherProvisionServer)

			_, err := controllerutil.CreateOrPatch(
				th.Ctx, th.K8sClient, anotherProvisionServer, func() error { return nil })
			Expect(err).ShouldNot(HaveOccurred())

			// Ensure both provision servers exist before attempting update
			Eventually(func(g Gomega) {
				firstPS := &unstructured.Unstructured{}
				firstPS.SetAPIVersion("baremetal.openstack.org/v1beta1")
				firstPS.SetKind("OpenStackProvisionServer")
				g.Expect(th.K8sClient.Get(th.Ctx, provisionServerName, firstPS)).To(Succeed())

				secondPS := &unstructured.Unstructured{}
				secondPS.SetAPIVersion("baremetal.openstack.org/v1beta1")
				secondPS.SetKind("OpenStackProvisionServer")
				g.Expect(th.K8sClient.Get(th.Ctx, anotherProvisionServerName, secondPS)).To(Succeed())
			}, th.Timeout, th.Interval).Should(Succeed())

			// Get the existing first provision server and update its port to conflict
			existingPS := &unstructured.Unstructured{}
			existingPS.SetAPIVersion("baremetal.openstack.org/v1beta1")
			existingPS.SetKind("OpenStackProvisionServer")
			Expect(th.K8sClient.Get(th.Ctx, provisionServerName, existingPS)).To(Succeed())

			// Update the port to conflict with the second provision server
			currentSpec := existingPS.Object["spec"].(map[string]interface{})
			currentSpec["port"] = int64(6203) // Conflicting port
			existingPS.Object["spec"] = currentSpec

			// Try to update - this should fail
			err = th.K8sClient.Update(th.Ctx, existingPS)
			Expect(err).Should(HaveOccurred())
			var statusError *k8s_errors.StatusError
			Expect(errors.As(err, &statusError)).To(BeTrue())
			Expect(statusError.ErrStatus.Message).To(
				ContainSubstring("port 6203 is already in use"))
		})
	})

	When("When deleting ProvisionServer", func() {
		It("should allow deletion", func() {
			spec := DefaultProvisionServerSpec()
			object := DefaultProvisionServerTemplate(provisionServerName, spec)
			provisionServer := &unstructured.Unstructured{Object: object}

			_, err := controllerutil.CreateOrPatch(
				th.Ctx, th.K8sClient, provisionServer, func() error { return nil })
			Expect(err).ShouldNot(HaveOccurred())

			// Delete should succeed (webhook just logs and returns nil)
			err = th.K8sClient.Delete(th.Ctx, provisionServer)
			Expect(err).ShouldNot(HaveOccurred())
		})
	})

	When("When validating port range", func() {
		It("should reject ports outside valid range", func() {
			invalidPorts := []int64{6189, 6221, 1000, 8080}

			for _, invalidPort := range invalidPorts {
				spec := DefaultProvisionServerSpec()
				spec["port"] = invalidPort
				object := DefaultProvisionServerTemplate(provisionServerName, spec)
				unstructuredObj := &unstructured.Unstructured{Object: object}

				_, err := controllerutil.CreateOrPatch(
					th.Ctx, th.K8sClient, unstructuredObj, func() error { return nil })
				Expect(err).Should(HaveOccurred())
				// This validation comes from OpenAPI schema, not webhook
			}
		})

		It("should accept ports within valid range", func() {
			validPorts := []int64{6204, 6205, 6206, 6207}

			for i, validPort := range validPorts {
				provisionServerName := types.NamespacedName{
					Name:      fmt.Sprintf("test-provision-server-%d", i),
					Namespace: namespace,
				}
				spec := DefaultProvisionServerSpec()
				spec["port"] = validPort
				object := DefaultProvisionServerTemplate(provisionServerName, spec)
				unstructuredObj := &unstructured.Unstructured{Object: object}

				_, err := controllerutil.CreateOrPatch(
					th.Ctx, th.K8sClient, unstructuredObj, func() error { return nil })
				Expect(err).ShouldNot(HaveOccurred())

				// Clean up after each test
				th.DeleteInstance(unstructuredObj)
			}
		})
	})
})
