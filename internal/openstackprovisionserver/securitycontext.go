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

package openstackprovisionserver

import (
	"github.com/openstack-k8s-operators/lib-common/modules/common/pod"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/utils/ptr"
)

// hardenedSecurityContext returns a SecurityContext based on lib-common's
// RestrictiveSecurityContext with SeccompProfile set to nil — OpenShift's
// hostnetwork SCC (required for HostNetwork: true) rejects seccomp annotations
// — and ReadOnlyRootFilesystem enabled.
func hardenedSecurityContext() *corev1.SecurityContext {
	sc := pod.RestrictiveSecurityContext(1001, 0)
	sc.SeccompProfile = nil
	// hostnetwork SCC allocates UID/GID from its own range; explicit values
	// are rejected. RunAsNonRoot (set above) is sufficient to enforce non-root.
	sc.RunAsUser = nil
	sc.RunAsGroup = nil
	sc.ReadOnlyRootFilesystem = ptr.To(true)
	return sc
}
