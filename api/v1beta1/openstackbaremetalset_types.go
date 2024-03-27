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

package v1beta1

import (
	condition "github.com/openstack-k8s-operators/lib-common/modules/common/condition"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// AutomatedCleaningMode is the interface to enable/disable automated cleaning
// +kubebuilder:validation:Enum=metadata;disabled
type AutomatedCleaningMode string

// InstanceSpec Instance specific attributes
type InstanceSpec struct {
	// +kubebuilder:validation:Optional
	// BmhLabelSelector allows for the selection of a particular BaremetalHost based on arbitrary labels
	BmhLabelSelector map[string]string `json:"bmhLabelSelector,omitempty"`
	// +kubebuilder:validation:Optional
	// CtlPlaneIP - Control Plane IP in CIDR notation
	CtlPlaneIP string `json:"ctlPlaneIP"`
	// +kubebuilder:validation:Optional
	// UserData - Host User Data
	UserData *corev1.SecretReference `json:"userData,omitempty"`
	// +kubebuilder:validation:Optional
	// NetworkData - Host Network Data
	NetworkData *corev1.SecretReference `json:"networkData,omitempty"`
	// +kubebuilder:validation:Optional
	// PreprovisioningNetworkDataName - NetwoData Secret name for Preprovisining in the local namespace
	PreprovisioningNetworkDataName string `json:"preprovisioningNetworkDataName,omitempty"`
}

// Allowed automated cleaning modes
const (
	CleaningModeDisabled AutomatedCleaningMode = "disabled"
	CleaningModeMetadata AutomatedCleaningMode = "metadata"
)

// OpenStackBaremetalSetSpec defines the desired state of OpenStackBaremetalSet
type OpenStackBaremetalSetSpec struct {
	// +kubebuilder:validation:Optional
	// BaremetalHosts - Map of hostname to Instance Spec for all nodes to provision
	BaremetalHosts map[string]InstanceSpec `json:"baremetalHosts,omitempty"`
	// +kubebuilder:validation:Optional
	// OSImage - OS qcow2 image Name
	OSImage string `json:"osImage,omitempty"`
	// +kubebuilder:validation:Optional
	// OSContainerImageURL - Container image URL for init with the OS qcow2 image (osImage)
	OSContainerImageURL string `json:"osContainerImageUrl,omitempty"`
	// +kubebuilder:validation:Optional
	// ApacheImageURL - Container image URL for the main container that serves the downloaded OS qcow2 image (osImage)
	ApacheImageURL string `json:"apacheImageUrl,omitempty"`
	// +kubebuilder:validation:Optional
	// AgentImageURL - Container image URL for the sidecar container that discovers provisioning network IPs
	AgentImageURL string `json:"agentImageUrl,omitempty"`
	// +kubebuilder:validation:Optional
	// UserData holds the reference to the Secret containing the user
	// data to be passed to the host before it boots. UserData can be
	// set per host in BaremetalHosts or here. If none of these are
	// provided it will use a default cloud-config.
	UserData *corev1.SecretReference `json:"userData,omitempty"`
	// +kubebuilder:validation:Optional
	// NetworkData holds the reference to the Secret containing network
	// data to be passed to the hosts. NetworkData can be set per host in
	// BaremetalHosts or here. If none of these are provided it will use
	// default NetworkData to configure CtlPlaneIP.
	NetworkData *corev1.SecretReference `json:"networkData,omitempty"`
	// When set to disabled, automated cleaning will be avoided
	// during provisioning and deprovisioning.
	// +kubebuilder:default=metadata
	// +kubebuilder:validation:Optional
	AutomatedCleaningMode AutomatedCleaningMode `json:"automatedCleaningMode,omitempty"`
	// ProvisionServerName - Optional. If supplied will be used as the base Image for the baremetalset instead of baseImageURL.
	// +kubebuilder:validation:Optional
	ProvisionServerName string `json:"provisionServerName,omitempty"`
	// ProvisioningInterface - Optional. If not provided along with ProvisionServerName, it would be discovered from CBO.  This is the provisioning interface on the OCP masters/workers.
	// +kubebuilder:validation:Optional
	ProvisioningInterface string `json:"provisioningInterface,omitempty"`
	// DeploymentSSHSecret - Name of secret holding the cloud-admin ssh keys
	DeploymentSSHSecret string `json:"deploymentSSHSecret"`
	// CtlplaneInterface - Interface on the provisioned nodes to use for ctlplane network
	CtlplaneInterface string `json:"ctlplaneInterface"`
	// CtlplaneGateway - IP of gateway for ctrlplane network (TODO: acquire this is another manner?)
	// +kubebuilder:validation:Optional
	CtlplaneGateway string `json:"ctlplaneGateway,omitempty"`
	// +kubebuilder:validation:Optional
	// +kubebuilder:default="255.255.255.0"
	// CtlplaneNetmask - Netmask to use for ctlplane network (TODO: acquire this is another manner?)
	CtlplaneNetmask string `json:"ctlplaneNetmask,omitempty"`
	// +kubebuilder:default=openshift-machine-api
	// +kubebuilder:validation:Optional
	// BmhNamespace Namespace to look for BaremetalHosts(default: openshift-machine-api)
	BmhNamespace string `json:"bmhNamespace,omitempty"`
	// +kubebuilder:validation:Optional
	// BmhLabelSelector allows for a sub-selection of BaremetalHosts based on arbitrary labels
	BmhLabelSelector map[string]string `json:"bmhLabelSelector,omitempty"`
	// +kubebuilder:validation:Optional
	// Hardware requests for sub-selection of BaremetalHosts with certain hardware specs
	HardwareReqs HardwareReqs `json:"hardwareReqs,omitempty"`
	// +kubebuilder:validation:Optional
	// PasswordSecret the name of the secret used to optionally set the root pwd by adding
	// NodeRootPassword: <base64 enc pwd>
	// to the secret data
	PasswordSecret *corev1.SecretReference `json:"passwordSecret,omitempty"`
	// +kubebuilder:default=cloud-admin
	// CloudUser to be configured for remote access
	CloudUserName string `json:"cloudUserName"`
	// DomainName is the domain name that will be set on the underlying Metal3 BaremetalHosts (TODO: acquire this is another manner?)
	// +kubebuilder:validation:Optional
	DomainName string `json:"domainName,omitempty"`
	// +kubebuilder:validation:Optional
	// BootstrapDNS - initial DNS nameserver values to set on the BaremetalHosts when they are provisioned.
	// Note that subsequent deployment will overwrite these values
	BootstrapDNS []string `json:"bootstrapDns,omitempty"`
	// +kubebuilder:validation:Optional
	// DNSSearchDomains - initial DNS nameserver values to set on the BaremetalHosts when they are provisioned.
	// Note that subsequent deployment will overwrite these values
	DNSSearchDomains []string `json:"dnsSearchDomains,omitempty"`
}

// OpenStackBaremetalSetStatus defines the observed state of OpenStackBaremetalSet
type OpenStackBaremetalSetStatus struct {
	// Conditions
	Conditions condition.Conditions `json:"conditions,omitempty" optional:"true"`
	// Map of hashes to track e.g. job status
	Hash map[string]string `json:"hash,omitempty"`
	// BaremetalHosts that are being processed or have been processed for this OpenStackBaremetalSet
	BaremetalHosts map[string]HostStatus `json:"baremetalHosts,omitempty" optional:"true"`
	// ObservedGeneration - the most recent generation observed for this
	// service. If the observed generation is less than the spec generation,
	// then the controller has not processed the latest changes injected by
	// the opentack-operator in the top-level CR (e.g. the ContainerImage)
	ObservedGeneration int64 `json:"observedGeneration,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:shortName=osbmset;osbmsets;osbms
// +operator-sdk:csv:customresourcedefinitions:displayName="OpenStack BaremetalSet"
// +kubebuilder:printcolumn:name="Status",type="string",JSONPath=".status.conditions[0].status",description="Status"
// +kubebuilder:printcolumn:name="Message",type="string",JSONPath=".status.conditions[0].message",description="Message"

// OpenStackBaremetalSet is the Schema for the openstackbaremetalsets API
type OpenStackBaremetalSet struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   OpenStackBaremetalSetSpec   `json:"spec,omitempty"`
	Status OpenStackBaremetalSetStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// OpenStackBaremetalSetList contains a list of OpenStackBaremetalSet
type OpenStackBaremetalSetList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []OpenStackBaremetalSet `json:"items"`
}

// ProvisioningState - the overall state of a BMH
type ProvisioningState string

// IPStatus represents the hostname and IP info for a specific host
type IPStatus struct {
	Hostname string `json:"hostname"`

	// +kubebuilder:default=unassigned
	BmhRef string `json:"bmhRef"`

	// +kubebuilder:validation:Optional
	IPAddresses map[string]string `json:"ipAddresses"`
}

// HostStatus represents the IPStatus and provisioning state + deployment information
type HostStatus struct {

	// +kubebuilder:validation:Required
	// IPStatus -
	IPStatus `json:",inline"`

	ProvisioningState ProvisioningState `json:"provisioningState"`

	// +kubebuilder:default=false
	// Host annotated for deletion
	AnnotatedForDeletion bool `json:"annotatedForDeletion"`

	UserDataSecretName    string `json:"userDataSecretName"`
	NetworkDataSecretName string `json:"networkDataSecretName"`
}

// HardwareReqs defines request hardware attributes for the BaremetalHost replicas
type HardwareReqs struct {
	CPUReqs  CPUReqs  `json:"cpuReqs,omitempty"`
	MemReqs  MemReqs  `json:"memReqs,omitempty"`
	DiskReqs DiskReqs `json:"diskReqs,omitempty"`
}

// CPUReqs defines specific CPU hardware requests
type CPUReqs struct {
	// Arch is a scalar (string) because it wouldn't make sense to give it an "exact-match" option
	// Can be either "x86_64" or "ppc64le" if included
	// +kubebuilder:validation:Enum=x86_64;ppc64le
	Arch     string      `json:"arch,omitempty"`
	CountReq CPUCountReq `json:"countReq,omitempty"`
	MhzReq   CPUMhzReq   `json:"mhzReq,omitempty"`
}

// CPUCountReq defines a specific hardware request for CPU core count
type CPUCountReq struct {
	// +kubebuilder:validation:Minimum=1
	Count int `json:"count,omitempty"`
	// If ExactMatch == false, actual count > Count will match
	ExactMatch bool `json:"exactMatch,omitempty"`
}

// CPUMhzReq defines a specific hardware request for CPU clock speed
type CPUMhzReq struct {
	// +kubebuilder:validation:Minimum=1
	Mhz int `json:"mhz,omitempty"`
	// If ExactMatch == false, actual mhz > Mhz will match
	ExactMatch bool `json:"exactMatch,omitempty"`
}

// MemReqs defines specific memory hardware requests
type MemReqs struct {
	GbReq MemGbReq `json:"gbReq,omitempty"`
}

// MemGbReq defines a specific hardware request for memory size
type MemGbReq struct {
	// +kubebuilder:validation:Minimum=1
	Gb int `json:"gb,omitempty"`
	// If ExactMatch == false, actual GB > Gb will match
	ExactMatch bool `json:"exactMatch,omitempty"`
}

// DiskReqs defines specific disk hardware requests
type DiskReqs struct {
	GbReq DiskGbReq `json:"gbReq,omitempty"`
	// SSD is scalar (bool) because it wouldn't make sense to give it an "exact-match" option
	SSDReq DiskSSDReq `json:"ssdReq,omitempty"`
}

// DiskGbReq defines a specific hardware request for disk size
type DiskGbReq struct {
	// +kubebuilder:validation:Minimum=1
	Gb int `json:"gb,omitempty"`
	// If ExactMatch == false, actual GB > Gb will match
	ExactMatch bool `json:"exactMatch,omitempty"`
}

// DiskSSDReq defines a specific hardware request for disk of type SSD (true) or rotational (false)
type DiskSSDReq struct {
	SSD bool `json:"ssd,omitempty"`
	// We only actually care about SSD flag if it is true or ExactMatch is set to true.
	// This second flag is necessary as SSD's bool zero-value (false) is indistinguishable
	// from it being explicitly set to false
	ExactMatch bool `json:"exactMatch,omitempty"`
}

//
// BEGIN - functions
// NOTE: Eventually we will need to move certain functions from the main module's "pkg" dir
// into this module/package instead.  This is because we will be adding necessary webhooks that
// will need some of those functions to exist in this module/package to avoid import cycle errors
//

// IsReady - returns true if OpenStackBaremetalSet is reconciled successfully
func (instance *OpenStackBaremetalSet) IsReady() bool {
	return instance.Status.Conditions.IsTrue(condition.ReadyCondition)
}

//
// END - functions
//

func init() {
	SchemeBuilder.Register(&OpenStackBaremetalSet{}, &OpenStackBaremetalSetList{})
}
