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
	"fmt"

	baremetalv1 "github.com/openstack-k8s-operators/openstack-baremetal-operator/api/v1beta1"

	"github.com/openstack-k8s-operators/lib-common/modules/common/env"

	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	// ChecksumCommand -
	ChecksumCommand = "/openstack-baremetal-agent checksum-discovery"
)

// ChecksumJob func
func ChecksumJob(
	instance *baremetalv1.OpenStackProvisionServer,
	labels map[string]string,
	annotations map[string]string,
) *batchv1.Job {
	args := []string{"-c", ChecksumCommand}

	envVars := map[string]env.Setter{}
	envVars["OS_IMAGE_DIR"] = env.SetValue(*instance.Spec.OSImageDir)
	envVars["PROV_SERVER_NAME"] = env.SetValue(instance.Name)
	envVars["PROV_SERVER_NAMESPACE"] = env.SetValue(instance.Namespace)

	// We actually use init volumes and mounts for this job
	volumes := getInitVolumes()
	volumeMounts := getInitVolumeMounts(instance)

	job := &batchv1.Job{
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("%s-checksum-discovery", instance.Name),
			Namespace: instance.Namespace,
			Labels:    labels,
		},
		Spec: batchv1.JobSpec{
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: annotations,
				},
				Spec: corev1.PodSpec{
					RestartPolicy:      corev1.RestartPolicyOnFailure,
					ServiceAccountName: instance.RbacResourceName(),
					Containers: []corev1.Container{
						{
							Name: fmt.Sprintf("%s-checksum-discovery", instance.Name),
							Command: []string{
								"/bin/bash",
							},
							Args:         args,
							Image:        instance.Spec.AgentImageURL,
							Env:          env.MergeEnvs([]corev1.EnvVar{}, envVars),
							VolumeMounts: volumeMounts,
						},
					},
					Volumes: volumes,
				},
			},
		},
	}

	if len(instance.Spec.NodeSelector) > 0 {
		job.Spec.Template.Spec.NodeSelector = instance.Spec.NodeSelector
	}

	initContainerDetails := InitContainerDetails{
		OsImageDir:     *instance.Spec.OSImageDir,
		ContainerImage: instance.Spec.OSContainerImageURL,
		VolumeMounts:   getInitVolumeMounts(instance),
	}
	job.Spec.Template.Spec.InitContainers = InitContainer(initContainerDetails)

	return job
}
