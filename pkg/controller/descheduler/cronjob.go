package descheduler

import (
	"context"
	"fmt"
	"log"
	"reflect"
	"strings"

	deschedulerv1alpha1 "github.com/skckadiyala/descheduler-operator/pkg/apis/descheduler/v1alpha1"
	batch "k8s.io/api/batch/v1"
	batchv1beta1 "k8s.io/api/batch/v1beta1"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

// generateDeschedulerJob generates Descheduler job.
func (r *ReconcileDescheduler) generateDeschedulerJob(Descheduler *deschedulerv1alpha1.Descheduler) error {
	// Check if the cron job already exists
	DeschedulerCronJob := &batchv1beta1.CronJob{}
	err := r.client.Get(context.TODO(), types.NamespacedName{Name: Descheduler.Name, Namespace: Descheduler.Namespace}, DeschedulerCronJob)
	if err != nil && errors.IsNotFound(err) {
		// Create Descheduler cronjob
		dj, err := r.createCronJob(Descheduler)
		if err != nil {
			log.Printf(" error while creating job %v", err)
			return err
		}
		log.Printf("Creating a new cron job %s/%s\n", dj.Namespace, dj.Name)
		err = r.client.Create(context.TODO(), dj)
		if err != nil {
			log.Printf(" error while creating cron job %v", err)
			return err
		}
		// Cronjob created successfully - don't requeue
		return nil
	} else if DeschedulerCronJob.Spec.Schedule != Descheduler.Spec.Schedule {
		// Descheduler schedule mismatch. Let's delete it and in the next reconcilation loop, we will create a new one.
		log.Printf("Schedule mismatch in cron job. Delete it")
		err = r.client.Delete(context.TODO(), DeschedulerCronJob, client.PropagationPolicy(metav1.DeletePropagationOrphan))
		if err != nil {
			log.Printf("Error while deleting cronjob")
			return err
		}
		return r.updateDeschedulerStatus(Descheduler, Updating)
	} else if !CheckIfFlagsChanged(Descheduler.Spec.Flags, DeschedulerCronJob.Spec.JobTemplate.Spec.
		Template.Spec.Containers[0].Command) {
		//By the time we reach here, job would have been created, so no need to check for nil pointers anywhere
		// till command
		log.Printf("Flags mismatch for Descheduler. Delete cronjob")

		err = r.client.Delete(context.TODO(), DeschedulerCronJob, client.PropagationPolicy(metav1.DeletePropagationOrphan))
		if err != nil {
			log.Printf("Error while deleting cronjob")
			return err
		}
		return r.updateDeschedulerStatus(Descheduler, Updating)
	} else if Descheduler.Spec.Image != DeschedulerCronJob.Spec.JobTemplate.Spec.Template.Spec.Containers[0].Image {
		log.Printf("Image mismatch within Descheduler. Delete cronjob")
		err = r.client.Delete(context.TODO(), DeschedulerCronJob, client.PropagationPolicy(metav1.DeletePropagationOrphan))
		if err != nil {
			log.Printf("Error while deleting cronjob")
			return err
		}
		return r.updateDeschedulerStatus(Descheduler, Updating)
	} else if err != nil {
		return err
	}
	return nil
}

// CheckIfFlagsChanged checks if any of the flags changed.
func CheckIfFlagsChanged(newFlags []deschedulerv1alpha1.Param, oldFlags []string) bool {
	latestFlags, err := ValidateFlags(newFlags)
	if err != nil {
		log.Printf("Invalid flags detected")
		return false
	}
	if latestFlags == nil {
		latestFlags = DeschedulerCommand
		return reflect.DeepEqual(latestFlags, oldFlags)
	}
	return reflect.DeepEqual(latestFlags, oldFlags)
}

// ValidateFlags validates flags for descheduler. We don't validate the values here in descheduler operator.
func ValidateFlags(flags []deschedulerv1alpha1.Param) ([]string, error) {
	log.Printf("Validating descheduler flags")
	if len(flags) == 0 {
		return nil, nil
	}
	deschedulerFlags := make([]string, 0)
	validFlags := []string{"descheduling-interval", "dry-run", "node-selector"}
	// deschedulerFlags = DeschedulerCommand
	// log.Printf("deschedulerFlags %v ", deschedulerFlags)
	for _, flag := range flags {
		allowedFlag := false
		for _, validFlag := range validFlags {
			if flag.Name == validFlag {
				allowedFlag = true
			}
		}
		if allowedFlag {
			deschedulerFlags = append(deschedulerFlags, []string{"--" + flag.Name, flag.Value}...)
		} else {
			return nil, fmt.Errorf("descheduler allows only following flags %v but found %v", strings.Join(validFlags, ","), flag.Name)
		}
	}
	// log.Printf("deschedulerFlags %v ", deschedulerFlags)
	return deschedulerFlags, nil
}

// createCronJob creates a descheduler job.
func (r *ReconcileDescheduler) createCronJob(descheduler *deschedulerv1alpha1.Descheduler) (*batchv1beta1.CronJob, error) {
	log.Printf("Creating descheduler job")
	// ttl := int32(100)
	// TTLSecondsAfterFinished: &ttl,
	flags, err := ValidateFlags(descheduler.Spec.Flags)
	if err != nil {
		return nil, err
	}
	if len(descheduler.Spec.Image) == 0 {
		// Set the default image here
		descheduler.Spec.Image = DefaultImage // No need to update the CR here making it opaque to end-user
	}

	flags = append(DeschedulerCommand, flags...)

	job := &batchv1beta1.CronJob{
		TypeMeta: metav1.TypeMeta{
			Kind:       "CronJob",
			APIVersion: batch.SchemeGroupVersion.String(),
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      descheduler.Name,
			Namespace: descheduler.Namespace,
		},
		Spec: batchv1beta1.CronJobSpec{
			Schedule: descheduler.Spec.Schedule,
			JobTemplate: batchv1beta1.JobTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Name: "descheduler-job-spec",
				},
				Spec: batch.JobSpec{
					// TTLSecondsAfterFinished: &ttl,
					Template: v1.PodTemplateSpec{
						Spec: v1.PodSpec{
							Volumes: []v1.Volume{{
								Name: "policy-volume",
								VolumeSource: v1.VolumeSource{
									ConfigMap: &v1.ConfigMapVolumeSource{
										LocalObjectReference: v1.LocalObjectReference{
											Name: descheduler.Name,
										},
									},
								},
							},
							},
							PriorityClassName: "system-cluster-critical",
							RestartPolicy:     "Never",
							Containers: []v1.Container{{
								Name:  "descheduler-axway",
								Image: descheduler.Spec.Image,
								Resources: v1.ResourceRequirements{
									Limits: v1.ResourceList{
										v1.ResourceCPU:    resource.MustParse("100m"),
										v1.ResourceMemory: resource.MustParse("500Mi"),
									},
									Requests: v1.ResourceList{
										v1.ResourceCPU:    resource.MustParse("100m"),
										v1.ResourceMemory: resource.MustParse("500Mi"),
									},
								},
								Command: flags,
								VolumeMounts: []v1.VolumeMount{{
									MountPath: "/policy-dir",
									Name:      "policy-volume",
								}},
							}},
							ServiceAccountName: "descheduler-operator", // TODO: This is hardcoded as of now, find a way to reference it from rbac.yaml.
						},
					},
				},
			},
		},
	}
	err = controllerutil.SetControllerReference(descheduler, job, r.scheme)
	if err != nil {
		return nil, fmt.Errorf("error setting owner references %v", err)
	}
	return job, nil
}
