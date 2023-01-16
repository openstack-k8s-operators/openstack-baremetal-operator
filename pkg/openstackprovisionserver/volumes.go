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
	corev1 "k8s.io/api/core/v1"
)

// getVolumes - general provisioning service volumes
func getVolumes() []corev1.Volume {
	return []corev1.Volume{
		{
			Name: "image-data",
			VolumeSource: corev1.VolumeSource{
				EmptyDir: &corev1.EmptyDirVolumeSource{},
			},
		},
		{
			Name: "httpd-config",
			VolumeSource: corev1.VolumeSource{
				ConfigMap: &corev1.ConfigMapVolumeSource{
					LocalObjectReference: corev1.LocalObjectReference{
						Name: "httpd-config",
					},
				},
			},
		},
	}
}

// getInitVolumeMounts - general init task VolumeMounts
func getInitVolumeMounts() []corev1.VolumeMount {
	return []corev1.VolumeMount{
		{
			Name:      "image-data",
			MountPath: "/usr/local/apache2/htdocs",
		},
		{
			Name:      "httpd-config",
			MountPath: HttpdConfPath,
			SubPath:   "httpd.conf",
			ReadOnly:  false,
		},
	}
}

// getVolumeMounts - general VolumeMounts
func getVolumeMounts() []corev1.VolumeMount {
	return []corev1.VolumeMount{
		{
			Name:      "image-data",
			MountPath: "/usr/local/apache2/htdocs",
		},
		{
			Name:      "httpd-config",
			MountPath: HttpdConfPath,
			SubPath:   "httpd.conf",
			ReadOnly:  true,
		},
	}
}
