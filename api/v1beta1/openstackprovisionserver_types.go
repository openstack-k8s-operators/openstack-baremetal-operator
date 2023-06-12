/*


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

package v1beta1

import (
	condition "github.com/openstack-k8s-operators/lib-common/modules/common/condition"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// OpenStackProvisionServerSpec defines the desired state of OpenStackProvisionServer
type OpenStackProvisionServerSpec struct {
	// Port - The port on which the Apache server should listen
	Port int32 `json:"port"`
	// +kubebuilder:validation:Optional
	// Interface - An optional interface to use instead of the cluster's default provisioning interface (if any)
	Interface string `json:"interface,omitempty"`
	// OSImage - OS qcow2 image (compressed as gz, or uncompressed)
	OSImage string `json:"osImage"`
	// OSContainerImageURL - Container image URL for init with the OS qcow2 image (osImage)
	OSContainerImageURL string `json:"osContainerImageUrl"`
	// ApacheImageURL - Container image URL for the main container that serves the downloaded OS qcow2 image (osImage)
	ApacheImageURL string `json:"apacheImageUrl"`
	// AgentImageURL - Container image URL for the sidecar container that discovers provisioning network IPs
	AgentImageURL string `json:"agentImageUrl"`
	// +kubebuilder:validation:Optional
	// NodeSelector to target subset of worker nodes running this provision server
	NodeSelector map[string]string `json:"nodeSelector,omitempty"`
	// +kubebuilder:validation:Optional
	// Resources - Compute Resources required by this provision server (Limits/Requests).
	// https://kubernetes.io/docs/concepts/configuration/manage-resources-containers/
	Resources corev1.ResourceRequirements `json:"resources,omitempty"`
	// +kubebuilder:validation:Required
	// ServiceAccount - service account name used internally to provide ProvisionServer the default SA name
	// +kubebuilder:default="provisionserver"
	ServiceAccount string `json:"serviceAccount"`
}

// OpenStackProvisionServerStatus defines the observed state of OpenStackProvisionServer
type OpenStackProvisionServerStatus struct {
	// ReadyCount of provision server Apache instances
	ReadyCount int32 `json:"readyCount,omitempty"`
	// Conditions
	Conditions condition.Conditions `json:"conditions,omitempty" optional:"true"`
	// Map of hashes to track e.g. job status
	Hash map[string]string `json:"hash,omitempty"`
	// IP of the provisioning interface on the node running the ProvisionServer pod
	ProvisionIP string `json:"provisionIp,omitempty"`
	// URL of provisioning image on underlying Apache web server
	LocalImageURL string `json:"localImageUrl,omitempty"`
}

// IsReady - returns true if service is ready to serve requests
func (instance *OpenStackProvisionServer) IsReady() bool {
	return instance.Status.ReadyCount > 0 && instance.Status.LocalImageURL != ""
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:shortName=osprovserver;osprovservers
// +operator-sdk:csv:customresourcedefinitions:displayName="OpenStackProvisionServer"
// +kubebuilder:printcolumn:name="Status",type="string",JSONPath=".status.conditions[0].status",description="Status"
// +kubebuilder:printcolumn:name="Message",type="string",JSONPath=".status.conditions[0].message",description="Message"

// OpenStackProvisionServer used to serve custom images for baremetal provisioning with Metal3
type OpenStackProvisionServer struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   OpenStackProvisionServerSpec   `json:"spec,omitempty"`
	Status OpenStackProvisionServerStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// OpenStackProvisionServerList contains a list of OpenStackProvisionServer
type OpenStackProvisionServerList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []OpenStackProvisionServer `json:"items"`
}

// OpenStackProvisionServerDefaults -
type OpenStackProvisionServerDefaults struct {
	OSContainerImageURL string
	AgentImageURL       string
	ApacheImageURL      string
	OSImage             string
}

func init() {
	SchemeBuilder.Register(&OpenStackProvisionServer{}, &OpenStackProvisionServerList{})
}

// RbacConditionsSet - set the conditions for the rbac object
func (instance OpenStackProvisionServer) RbacConditionsSet(c *condition.Condition) {
	instance.Status.Conditions.Set(c)
}

// RbacNamespace - return the namespace
func (instance OpenStackProvisionServer) RbacNamespace() string {
	return instance.Namespace
}

// RbacResourceName - return the name to be used for rbac objects (serviceaccount, role, rolebinding)
func (instance OpenStackProvisionServer) RbacResourceName() string {
	return "provisionserver-" + instance.Name
}
