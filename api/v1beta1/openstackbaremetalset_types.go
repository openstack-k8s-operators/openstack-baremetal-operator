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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// OpenStackBaremetalSetSpec defines the desired state of OpenStackBaremetalSet
type OpenStackBaremetalSetSpec struct {
	// +kubebuilder:default={}
	// +kubebuilder:validation:Optional
	// BaremetalHosts - Map of hostname to control plane IP address for all nodes to provision
	BaremetalHosts map[string]string `json:"baremetalHosts"`
	// RhelImageURL - Remote URL pointing to desired RHEL qcow2 image
	RhelImageURL string `json:"rhelImageUrl,omitempty"`
	// +kubebuilder:validation:Optional
	// ProvisionServerName - Optional. If supplied will be used as the base Image for the baremetalset instead of baseImageURL.
	ProvisionServerName string `json:"provisionServerName,omitempty"`
	// DeploymentSSHSecret - Name of secret holding the stack-admin ssh keys
	DeploymentSSHSecret string `json:"deploymentSSHSecret"`
	// CtlplaneInterface - Interface to use for ctlplane network
	CtlplaneInterface string `json:"ctlplaneInterface"`
	// CtlplaneGateway - IP of gateway for ctrlplane network (TODO: acquire this is another manner?)
	CtlplaneGateway string `json:"ctlplaneGateway"`
	// CtlplaneNetmask - Netmask to use for ctlplane network (TODO: acquire this is another manner?)
	CtlplaneNetmask string `json:"ctlplaneNetmask"`
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
	PasswordSecret string `json:"passwordSecret,omitempty"`
	// DomainName is the domain name that will be set on the underlying Metal3 BaremetalHosts (TODO: acquire this is another manner?)
	DomainName string `json:"domainName"`
	// +kubebuilder:validation:Optional
	// BootstrapDNS - initial DNS nameserver values to set on the BaremetalHosts when they are provisioned.
	// Note that subsequent TripleO deployment will overwrite these values
	BootstrapDNS []string `json:"bootstrapDns,omitempty"`
	// +kubebuilder:validation:Optional
	// DNSSearchDomains - initial DNS nameserver values to set on the BaremetalHosts when they are provisioned.
	// Note that subsequent TripleO deployment will overwrite these values
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

// IsReady - returns true if all requested BMHs are provisioned
func (instance *OpenStackBaremetalSet) IsReady() bool {
	return instance.Status.Conditions.IsTrue(OpenStackBaremetalSetBmhProvisioningReadyCondition)
}

//
// END - functions
//

func init() {
	SchemeBuilder.Register(&OpenStackBaremetalSet{}, &OpenStackBaremetalSetList{})
}
