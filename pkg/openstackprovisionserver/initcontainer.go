package openstackprovisionserver

import (
	"github.com/openstack-k8s-operators/lib-common/modules/common/env"
	corev1 "k8s.io/api/core/v1"
)

// InitContainerDetails information
type InitContainerDetails struct {
	ContainerImage string
	RhelImage      string
	TransportURL   string
	Privileged     bool
	VolumeMounts   []corev1.VolumeMount
}

// InitContainer - init container for provision server pods
func InitContainer(init InitContainerDetails) []corev1.Container {
	envs := []corev1.EnvVar{
		{
			Name:  "RHEL_IMAGE_URL",
			Value: init.RhelImage,
		},
	}
	envs = env.MergeEnvs(envs, map[string]env.Setter{})

	return []corev1.Container{
		{
			Name:  "init",
			Image: init.ContainerImage,
			SecurityContext: &corev1.SecurityContext{
				Privileged: &init.Privileged,
			},
			Env:          envs,
			VolumeMounts: init.VolumeMounts,
		},
	}
}
