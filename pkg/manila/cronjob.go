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

package manila

import (
	manilav1 "github.com/openstack-k8s-operators/manila-operator/api/v1beta1"

	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"strconv"
	"strings"
)

// DBPurgeCommandBase -
var DBPurgeCommandBase = [...]string{"/usr/bin/manila-manage", "--debug", "--config-dir /etc/manila/manila.conf.d", "db purge "}

// CronJob func
func CronJob(
	instance *manilav1.Manila,
	labels map[string]string,
	annotations map[string]string,
) *batchv1.CronJob {
	runAsUser := int64(0)
	var config0644AccessMode int32 = 0644
	var DBPurgeCommand []string = DBPurgeCommandBase[:]
	args := []string{"-c"}

	if !instance.Spec.Debug.DBPurge {
		// If debug mode is not requested, remove the --debug option
		DBPurgeCommand = append(DBPurgeCommandBase[:1], DBPurgeCommandBase[2:]...)
	}
	// Build the resulting command
	DBPurgeCommandString := strings.Join(DBPurgeCommand, " ")

	// Extend the resulting command with the DBPurgeAge int
	args = append(args, DBPurgeCommandString+strconv.Itoa(DBPurgeAge))

	parallelism := int32(1)
	completions := int32(1)

	cronJobVolume := []corev1.Volume{
		{
			Name: "db-purge-config-data",
			VolumeSource: corev1.VolumeSource{
				Secret: &corev1.SecretVolumeSource{
					DefaultMode: &config0644AccessMode,
					SecretName:  instance.Name + "-config-data",
					Items: []corev1.KeyToPath{
						{
							Key:  DefaultsConfigFileName,
							Path: DefaultsConfigFileName,
						},
					},
				},
			},
		},
	}

	cronJobVolumeMounts := []corev1.VolumeMount{
		{
			Name:      "db-purge-config-data",
			MountPath: "/etc/manila/manila.conf.d",
			ReadOnly:  true,
		},
	}

	cronjob := &batchv1.CronJob{
		ObjectMeta: metav1.ObjectMeta{
			Name:      ServiceName + "-cron",
			Namespace: instance.Namespace,
		},
		Spec: batchv1.CronJobSpec{
			Schedule:          DBPurgeDefaultSchedule,
			ConcurrencyPolicy: batchv1.ForbidConcurrent,
			JobTemplate: batchv1.JobTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: annotations,
					Labels:      labels,
				},
				Spec: batchv1.JobSpec{
					Parallelism: &parallelism,
					Completions: &completions,
					Template: corev1.PodTemplateSpec{
						Spec: corev1.PodSpec{
							Containers: []corev1.Container{
								{
									Name:  ServiceName + "-cron",
									Image: instance.Spec.ManilaAPI.ContainerImage,
									Command: []string{
										"/bin/bash",
									},
									Args:         args,
									VolumeMounts: cronJobVolumeMounts,
									SecurityContext: &corev1.SecurityContext{
										RunAsUser: &runAsUser,
									},
								},
							},
							Volumes:            cronJobVolume,
							RestartPolicy:      corev1.RestartPolicyNever,
							ServiceAccountName: instance.RbacResourceName(),
						},
					},
				},
			},
		},
	}
	if instance.Spec.NodeSelector != nil && len(instance.Spec.NodeSelector) > 0 {
		cronjob.Spec.JobTemplate.Spec.Template.Spec.NodeSelector = instance.Spec.NodeSelector
	}

	return cronjob
}
