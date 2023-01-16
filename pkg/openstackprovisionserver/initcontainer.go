package openstackprovisionserver

import (
	"github.com/openstack-k8s-operators/lib-common/modules/common/env"
	corev1 "k8s.io/api/core/v1"
)

// // InitContainer information
// type InitContainer struct {
// 	Args           []string
// 	Commands       []string
// 	ContainerImage string
// 	Env            []corev1.EnvVar
// 	Privileged     bool
// 	VolumeMounts   []corev1.VolumeMount
// }

// // GetInitContainers - init containers for ProvisionServers
// func GetInitContainers(inits []InitContainer) []corev1.Container {
// 	trueVar := true

// 	securityContext := &corev1.SecurityContext{}
// 	initContainers := []corev1.Container{}

// 	for index, init := range inits {
// 		if init.Privileged {
// 			securityContext.Privileged = &trueVar
// 		}

// 		container := corev1.Container{
// 			Name:            fmt.Sprintf("init-%d", index),
// 			Image:           init.ContainerImage,
// 			ImagePullPolicy: corev1.PullAlways,
// 			SecurityContext: securityContext,
// 			VolumeMounts:    init.VolumeMounts,
// 			Env:             init.Env,
// 		}

// 		if len(init.Args) != 0 {
// 			container.Args = init.Args
// 		}

// 		if len(init.Commands) != 0 {
// 			container.Command = init.Commands
// 		}

// 		initContainers = append(initContainers, container)
// 	}

// 	return initContainers
// }

// InitContainerDetails information
type InitContainerDetails struct {
	ContainerImage string
	RhelImage      string
	TransportURL   string
	Privileged     bool
	VolumeMounts   []corev1.VolumeMount
}

const (
// InitContainerCommand -
// InitContainerCommand = "/usr/local/bin/container-scripts/init.sh"
)

// InitContainer - init container for provision server pods
func InitContainer(init InitContainerDetails) []corev1.Container {
	// runAsUser := int64(0)

	// args := []string{
	// 	"-c",
	// 	InitContainerCommand,
	// }

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
