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
	// OpenStackProvisionServerReadyCondition Status=True condition which indicates if the OpenStackProvisionServer is configured and operational
	OpenStackProvisionServerReadyCondition condition.Type = "OpenStackProvisionServerReady"

	// OpenStackProvisionServerProvIntfReadyCondition Status=True condition which indicates if the OpenStackProvisionServer was provided or otherwise able to find the provisioning interface
	OpenStackProvisionServerProvIntfReadyCondition condition.Type = "OpenStackProvisionServerProvIntfReady"
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
	OpenStackProvisionServerProvIntfReadyMessage = "OpenStackProvisionServerProvIntf completed"
)
