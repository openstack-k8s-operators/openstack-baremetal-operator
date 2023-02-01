package openstackbaremetalset

import (
	baremetalv1 "github.com/openstack-k8s-operators/openstack-baremetal-operator/api/v1beta1"
	corev1 "k8s.io/api/core/v1"
	k8s_errors "k8s.io/apimachinery/pkg/api/errors"
)

// GetBmhRefStatus ...
func GetBmhRefStatus(
	instance *baremetalv1.OpenStackBaremetalSet,
	bmh string,
) (baremetalv1.HostStatus, error) {

	for _, bmhStatus := range instance.Status.DeepCopy().BaremetalHosts {
		if bmhStatus.BmhRef == bmh {
			return bmhStatus, nil
		}
	}

	return baremetalv1.HostStatus{}, k8s_errors.NewNotFound(corev1.Resource("OpenStackBaremetalHostStatus"), "not found")
}
