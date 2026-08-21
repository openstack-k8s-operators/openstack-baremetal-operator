/*
Copyright 2023 Red Hat
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
	"fmt"

	baremetalv1 "github.com/openstack-k8s-operators/openstack-baremetal-operator/api/v1beta1"
	corev1 "k8s.io/api/core/v1"
)

// getVolumes - general provisioning service volumes
func getInitVolumes() []corev1.Volume {
	return []corev1.Volume{
		{
			Name: "image-data",
			VolumeSource: corev1.VolumeSource{
				EmptyDir: &corev1.EmptyDirVolumeSource{},
			},
		},
		scratchVolume("tmp"),
	}
}

// getVolumes - general provisioning service volumes
func getVolumes(name string) []corev1.Volume {
	return append(getInitVolumes(), corev1.Volume{
		Name: "httpd-config",
		VolumeSource: corev1.VolumeSource{
			ConfigMap: &corev1.ConfigMapVolumeSource{
				LocalObjectReference: corev1.LocalObjectReference{
					Name: fmt.Sprintf("%s-httpd-config", name),
				},
			},
		},
	},
		scratchVolume("httpd-conf-etc"),
	)
}

// getInitVolumeMounts - general init task VolumeMounts
func getInitVolumeMounts(instance *baremetalv1.OpenStackProvisionServer) []corev1.VolumeMount {
	return []corev1.VolumeMount{
		{
			Name:      "image-data",
			MountPath: *instance.Spec.OSImageDir,
		},
		{
			Name:      "tmp",
			MountPath: "/tmp",
		},
	}
}

// getScratchVolumeMounts - VolumeMounts for containers that only need writable
// scratch space (e.g. /tmp) and no other data
func getScratchVolumeMounts() []corev1.VolumeMount {
	return []corev1.VolumeMount{
		{
			Name:      "tmp",
			MountPath: "/tmp",
		},
	}
}

// getVolumeMounts - general VolumeMounts
func getVolumeMounts(instance *baremetalv1.OpenStackProvisionServer) []corev1.VolumeMount {
	return []corev1.VolumeMount{
		{
			Name:      "image-data",
			MountPath: *instance.Spec.OSImageDir,
		},
		{
			Name:      "httpd-config",
			MountPath: HttpdConfPath,
			SubPath:   "httpd.conf",
			ReadOnly:  true,
		},
		{
			Name:      "tmp",
			MountPath: "/tmp",
		},
		{
			// writable destination for the "cp httpd.conf /etc/httpd/conf/httpd.conf"
			// startup step, required since the container runs with a read-only
			// root filesystem
			Name:      "httpd-conf-etc",
			MountPath: "/etc/httpd/conf",
		},
	}
}
