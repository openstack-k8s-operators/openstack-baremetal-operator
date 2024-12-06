package openstackprovisionserver

import (
	"github.com/openstack-k8s-operators/lib-common/modules/common/env"
	corev1 "k8s.io/api/core/v1"
	"path/filepath"
	"strings"
)

// InitContainerDetails information
type InitContainerDetails struct {
	ContainerImage     string
	ContainerImageType string
	OsImageDir         string
	OsImage            string
	Privileged         bool
	VolumeMounts       []corev1.VolumeMount
}

// InitContainer - init container for provision server pods
func InitContainer(init InitContainerDetails) []corev1.Container {
	if init.ContainerImageType == "bootc" {

		osImage := init.OsImage
		osImageExtension := filepath.Ext(osImage)
		osImageNoExtension := strings.TrimSuffix(osImage, osImageExtension)
		osImagePathRaw := filepath.Join(init.OsImageDir, osImageNoExtension+".raw")

		// TODO(sbaker) if the extension is qcow2 add an init container which runs "qemu-img convert" on the raw image

		return []corev1.Container{
			{
				Name:    "init",
				Image:   init.ContainerImage,
				Command: []string{"bootc", "install", "to-disk", "--generic-image", "--via-loopback", osImagePathRaw},
				SecurityContext: &corev1.SecurityContext{
					Privileged: &init.Privileged,
					SELinuxOptions: &corev1.SELinuxOptions{
						Type: "unconfined_t",
					},
				},
				VolumeMounts: init.VolumeMounts,
			},
		}
	}
	envs := []corev1.EnvVar{
		{
			Name:  "DEST_DIR",
			Value: init.OsImageDir,
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
