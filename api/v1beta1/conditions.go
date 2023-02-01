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
)

// OpenStack Baremetal Condition Types used by API objects.
const (
	//
	// OpenStackProvisionServer conditions
	//
	// OpenStackProvisionServerReadyCondition Status=True condition which indicates if the OpenStackProvisionServer is configured and operational
	OpenStackProvisionServerReadyCondition condition.Type = "OpenStackProvisionServerReady"

	// OpenStackProvisionServerProvIntfReadyCondition Status=True condition which indicates if the OpenStackProvisionServer was provided or otherwise able to find the provisioning interface
	OpenStackProvisionServerProvIntfReadyCondition condition.Type = "OpenStackProvisionServerProvIntfReady"

	// OpenStackProvisionServerLocalImageUrlReadyCondition Status=True condition which indicates if the OpenStackProvisionServer's LocalImageUrl has been successfully acquired from the provisioning agent
	OpenStackProvisionServerLocalImageURLReadyCondition condition.Type = "OpenStackProvisionServerLocalImageUrlReady"

	//
	// OpenStackBaremetalSet conditions
	//
	// OpenStackBaremetalSetReadyCondition Status=True condition which indicates if the OpenStackBaremetalSet is fully provisioned
	OpenStackBaremetalSetReadyCondition condition.Type = "OpenStackBaremetalSetReady"

	// OpenStackBaremetalSetProvServerReadyCondition Status=True condition which indicates if the OpenStackBaremetalSet's OpenStackProvisionServer is ready to serve its image
	OpenStackBaremetalSetProvServerReadyCondition condition.Type = "OpenStackBaremetalSetProvServerReady"

	// OpenStackBaremetalSetBmhProvisioningReadyCondition Status=True condition which indicates if the OpenStackBaremetalSet's requested BMHs have been provisioned
	OpenStackBaremetalSetBmhProvisioningReadyCondition condition.Type = "OpenStackBaremetalSetBmhProvisioningReady"
)

// OpenStack Baremetal Reasons used by API objects.
const ()

// Common Messages used by API objects.
const (
	//
	// OpenStackProvisionServerReady condition messages
	//
	// OpenStackProvisionServerReadyInitMessage
	OpenStackProvisionServerReadyInitMessage = "OpenStackProvisionServer not started"

	// OpenStackProvisionServerReadyErrorMessage
	OpenStackProvisionServerReadyErrorMessage = "OpenStackProvisionServer error occured %s"

	//
	// OpenStackProvisionServerProvIntfReady condition messages
	//
	// OpenStackProvisionServerProvIntfReadyInitMessage
	OpenStackProvisionServerProvIntfReadyInitMessage = "OpenStackProvisionServerProvIntf not started"

	// OpenStackProvisionServerProvIntfReadyErrorMessage
	OpenStackProvisionServerProvIntfReadyErrorMessage = "OpenStackProvisionServerProvIntf error occured %s"

	// OpenStackProvisionServerProvIntfReadyMessage
	OpenStackProvisionServerProvIntfReadyMessage = "OpenStackProvisionServerProvIntf found"

	//
	// OpenStackProvisionServerLocalImageURLReady condition messages
	//
	// OpenStackProvisionServerLocalImageURLReadyInitMessage
	OpenStackProvisionServerLocalImageURLReadyInitMessage = "OpenStackProvisionServerLocalImageUrl not started"

	// OpenStackProvisionServerLocalImageURLReadyErrorMessage
	OpenStackProvisionServerLocalImageURLReadyErrorMessage = "OpenStackProvisionServerLocalImageUrl error occured %s"

	// OpenStackProvisionServerLocalImageURLReadyRunningMessage
	OpenStackProvisionServerLocalImageURLReadyRunningMessage = "OpenStackProvisionServerLocalImageUrl generation in progress"

	// OpenStackProvisionServerLocalImageURLReadyMessage
	OpenStackProvisionServerLocalImageURLReadyMessage = "OpenStackProvisionServerLocalImageUrl generated"

	//
	// OpenStackBaremetalSetReady condition messages
	//
	// OpenStackBaremetalSetReadyInitMessage
	OpenStackBaremetalSetReadyInitMessage = "OpenStackBaremetalSet not started"

	// OpenStackBaremetalSetReadyErrorMessage
	OpenStackBaremetalSetReadyErrorMessage = "OpenStackBaremetalSet error occured %s"

	//
	// OpenStackBaremetalSetProvServerReady condition messages
	//
	// OpenStackBaremetalSetProvServerReadyInitMessage
	OpenStackBaremetalSetProvServerReadyInitMessage = "OpenStackBaremetalSet provision server not started"

	// OpenStackBaremetalSetProvServerReadyWaitingMessage
	OpenStackBaremetalSetProvServerReadyWaitingMessage = "OpenStackBaremetalSet waiting for provision server creation"

	// OpenStackBaremetalSetProvServerReadyRunningMessage
	OpenStackBaremetalSetProvServerReadyRunningMessage = "OpenStackBaremetalSet provision server deployment in progress"

	// OpenStackBaremetalSetProvServerReadyErrorMessage
	OpenStackBaremetalSetProvServerReadyErrorMessage = "OpenStackBaremetalSet provision server error occured %s"

	// OpenStackBaremetalSetProvServerReadyMessage
	OpenStackBaremetalSetProvServerReadyMessage = "OpenStackBaremetalSet provision server ready"

	//
	// OpenStackBaremetalSetBmhProvisioningReady condition messages
	//
	// OpenStackBaremetalSetBmhProvisioningReadyInitMessage
	OpenStackBaremetalSetBmhProvisioningReadyInitMessage = "OpenStackBaremetalSet BMH provisioning not started"

	// OpenStackBaremetalSetBmhProvisioningReadyRunningMessage
	OpenStackBaremetalSetBmhProvisioningReadyRunningMessage = "OpenStackBaremetalSet BMH provisioning in progress"

	// OpenStackBaremetalSetBmhProvisioningReadyErrorMessage
	OpenStackBaremetalSetBmhProvisioningReadyErrorMessage = "OpenStackBaremetalSet BMH provisioning error occured %s"

	// OpenStackBaremetalSetBmhProvisioningReadyMessage
	OpenStackBaremetalSetBmhProvisioningReadyMessage = "OpenStackBaremetalSet BMH provisioning completed"
)
