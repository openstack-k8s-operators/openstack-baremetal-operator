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
	metal3v1 "github.com/metal3-io/baremetal-operator/apis/metal3.io/v1alpha1"
	condition "github.com/openstack-k8s-operators/lib-common/modules/common/condition"
	"github.com/openstack-k8s-operators/lib-common/modules/common/util"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// ProvisioningNetwork is the boot mode of the system
// +kubebuilder:validation:Enum=Managed;Unmanaged;Disabled
type ProvisioningNetwork string

const (
	// ProvisioningNetwork modes
	ProvisioningNetworkManaged   ProvisioningNetwork = "Managed"
	ProvisioningNetworkUnmanaged ProvisioningNetwork = "Unmanaged"
	ProvisioningNetworkDisabled  ProvisioningNetwork = "Disabled"
	// Checksum job hash
	ChecksumHash = "checksum"
	// DefaultProvisionPort - The starting default port for the OpenStackProvisionServer's Apache container
	ProvisionServerPortStart = 6190
	ProvisionServerPortEnd   = 6210
)

const (
	// OSContainerImage - default fall-back image for OpenStackProvisionServer int container
	OSContainerImage = "quay.io/podified-antelope-centos9/edpm-hardened-uefi:current-podified"
	// OSContainerImageType - default fall-back image type for OSContainerImage
	OSContainerImageType = "self-extracting"
	// AgentImage - default fall-back image for OpenStackProvisionServer agent
	AgentImage = "quay.io/openstack-k8s-operators/openstack-baremetal-operator-agent:latest"
	// ApacheImage - default fall-back image for Apache
	ApacheImage = "registry.redhat.io/ubi9/httpd-24:latest"
	// OSImage - default fall-back image name for qcow2 image found inside OSContainerImage
	OSImage = "edpm-hardened-uefi.qcow2"
)

// OpenStackProvisionServerSpec defines the desired state of OpenStackProvisionServer
type OpenStackProvisionServerSpec struct {
	// Port - The port on which the Apache server should listen
	// +kubebuilder:validation:Optional
	// +kubebuilder:validation:Minimum=6190
	// +kubebuilder:validation:Maximum=6220
	Port int32 `json:"port,omitempty"`
	// +kubebuilder:validation:Optional
	// Interface - An optional interface to use instead of the cluster's default provisioning interface (if any)
	Interface string `json:"interface,omitempty"`
	// +kubebuilder:validation:Optional
	// OSImage - For osContainerImageType=self-extracting, the name of the OS qcow2 file to extract from the image. For osContainerImageType=bootc, the name of the qcow2 file to write to for bootc install.
	OSImage string `json:"osImage"`
	// +kubebuilder:validation:Required
	// +kubebuilder:default=/usr/local/apache2/htdocs
	// OSImageDir - Directory on the container which holds the OS qcow2 image and checksum
	OSImageDir *string `json:"osImageDir"`
	// OSContainerImageURL - Container image URL for init with the OS qcow2 image (osImage)
	OSContainerImageURL string `json:"osContainerImageUrl"`
	// +kubebuilder:validation:Enum=self-extracting;bootc
	// +kubebuilder:default=self-extracting
	// OSContainerImageType - Whether the OS container image is a self-extracting or a bootc based image
	OSContainerImageType string `json:"osContainerImageType,omitempty"`
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
	// +kubebuilder:validation:Optional
	// +kubebuilder:default=false
	// PreserveJobs - do not delete jobs after they finished e.g. to check logs
	PreserveJobs bool `json:"preserveJobs"`
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
	// Filename of OSImage checksum
	OSImageChecksumFilename string `json:"osImageChecksumFilename,omitempty"`
	// OSImage checksum type
	OSImageChecksumType metal3v1.ChecksumType `json:"osImageChecksumType,omitempty"`
	// URL of provisioning image checksum on underlying Apache web server
	LocalImageChecksumURL string `json:"localImageChecksumUrl,omitempty"`
}

// IsReady - returns true if OpenStackProvisionServer is reconciled successfully
func (instance *OpenStackProvisionServer) IsReady() bool {
	return instance.Status.Conditions.IsTrue(condition.ReadyCondition)
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
	OSContainerImageURL  string
	OSContainerImageType string
	AgentImageURL        string
	ApacheImageURL       string
	OSImage              string
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

// SetupDefaults - initializes any CRD field defaults based on environment variables (the defaulting mechanism itself is implemented via webhooks)
// TODO: Move this to a common location if OpenStackBaremetalSets ever get added as well
func SetupDefaults() {
	// Acquire environmental defaults and initialize OpenStackProvisionServer defaults with them
	openstackProvisionServerDefaults := OpenStackProvisionServerDefaults{
		OSContainerImageURL:  util.GetEnvVar("RELATED_IMAGE_OS_CONTAINER_IMAGE_URL_DEFAULT", OSContainerImage),
		OSContainerImageType: util.GetEnvVar("RELATED_IMAGE_OS_CONTAINER_IMAGE_TYPE_DEFAULT", OSContainerImageType),
		AgentImageURL:        util.GetEnvVar("RELATED_IMAGE_AGENT_IMAGE_URL_DEFAULT", AgentImage),
		ApacheImageURL:       util.GetEnvVar("RELATED_IMAGE_APACHE_IMAGE_URL_DEFAULT", ApacheImage),
		OSImage:              util.GetEnvVar("OS_IMAGE_DEFAULT", OSImage),
	}

	SetupOpenStackProvisionServerDefaults(openstackProvisionServerDefaults)
}
