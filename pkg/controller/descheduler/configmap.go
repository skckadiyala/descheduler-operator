package descheduler

import (
	"context"
	"fmt"
	"log"
	"strconv"
	"strings"

	deschedulerv1alpha1 "github.com/skckadiyala/descheduler-operator/pkg/apis/descheduler/v1alpha1"
	"gopkg.in/yaml.v2"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

// Policy Struct for the policy.yaml file
type Policy struct {
	APIVersion string `yaml:"apiVersion"`
	Kind       string `yaml:"kind"`
	Strategies struct {
		LowNodeUtilization struct {
			Enabled bool `yaml:"enabled"`
			Params  struct {
				NodeResourceUtilizationThresholds struct {
					NumberOfNodes    int `yaml:"numberOfNodes"`
					TargetThresholds struct {
						CPU    int `yaml:",omitempty"`
						Memory int `yaml:",omitempty"`
						Pods   int `yaml:",omitempty"`
					} `yaml:"targetThresholds"`
					Thresholds struct {
						CPU    int `yaml:",omitempty"`
						Memory int `yaml:",omitempty"`
						Pods   int `yaml:",omitempty"`
					} `yaml:"thresholds"`
				} `yaml:"nodeResourceUtilizationThresholds"`
			} `yaml:"params"`
		} `yaml:"LowNodeUtilization"`
		RemoveDuplicates struct {
			Enabled bool `yaml:"enabled"`
		} `yaml:"RemoveDuplicates"`
		RemovePodsViolatingInterPodAntiAffinity struct {
			Enabled bool `yaml:"enabled"`
		} `yaml:"RemovePodsViolatingInterPodAntiAffinity"`
		RemovePodsViolatingNodeAffinity struct {
			Enabled bool `yaml:"enabled"`
			Params  struct {
				NodeAffinityType []string `yaml:"nodeAffinityType"`
			} `yaml:"params"`
		} `yaml:"RemovePodsViolatingNodeAffinity"`
	} `yaml:"strategies"`
}

// generateConfigMap generates configmap needed for the descheduler from CR
func (r *ReconcileDescheduler) generateConfigMap(descheduler *deschedulerv1alpha1.Descheduler) error {
	deschedulerConfigMap := &v1.ConfigMap{}
	err := r.client.Get(context.TODO(), types.NamespacedName{Name: descheduler.Name, Namespace: descheduler.Namespace}, deschedulerConfigMap)
	if err != nil && errors.IsNotFound(err) {
		//Create a new ConfigMap
		cm, err := r.createConfigMap(descheduler)
		if err != nil {
			log.Fatalf("%v", err)
			return err
		}
		err = r.client.Create(context.TODO(), cm)
		if err != nil {
			log.Fatalf("%v", err)
			return err
		}
	} else if !CheckIfPropertyChanges(descheduler.Spec.Strategies, deschedulerConfigMap.Data) {
		fmt.Println("Strategy mismatch in configmap, Delete it")
		err = r.client.Delete(context.TODO(), deschedulerConfigMap)
		if err != nil {
			log.Printf("Error while deleteing configmap")
			return err
		}
		return r.updateDeschedulerStatus(descheduler, Updating)
	} else if err != nil {
		return err
	}
	return nil
}

func (r *ReconcileDescheduler) updateDeschedulerStatus(descheduler *deschedulerv1alpha1.Descheduler, desiredPhase string) error {
	currentPhase := descheduler.Status.Phase
	descheduler.Status.Phase = desiredPhase
	if currentPhase == desiredPhase {
		return nil
	}
	log.Printf("Updating descheduler status ")
	err := r.client.Update(context.TODO(), descheduler)
	if err != nil {
		log.Printf("Failed to update descheduler status from %v to %v", currentPhase, desiredPhase)
		return err
	}
	return nil
}

func (r *ReconcileDescheduler) createConfigMap(descheduler *deschedulerv1alpha1.Descheduler) (*v1.ConfigMap, error) {
	log.Printf("Creating config map")
	deschedulerPolicy := &Policy{}

	strategiesPolicyString := generateConfigMapString(descheduler.Spec.Strategies, *deschedulerPolicy)
	log.Printf("strategiesPolicy: %v", strategiesPolicyString)

	cm := &v1.ConfigMap{
		TypeMeta: metav1.TypeMeta{
			Kind:       "ConfigMap",
			APIVersion: "v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      descheduler.Name,
			Namespace: descheduler.Namespace,
		},
		Data: map[string]string{
			"policy.yaml": strategiesPolicyString,
		},
	}
	err := controllerutil.SetControllerReference(descheduler, cm, r.scheme)
	if err != nil {
		return nil, fmt.Errorf("error setting owner references %v", err)
	}
	return cm, nil
}

func generateConfigMapString(requestedStrategies []deschedulerv1alpha1.Strategy, policy Policy) string {
	// There is no need to do validation here. By the time, we reach here, validation would have already happened.
	policy.APIVersion = "descheduler/v1alpha1"
	policy.Kind = "DeschedulerPolicy"
	for _, strategy := range requestedStrategies {
		switch strings.ToLower(strategy.Name) {
		case "duplicates":
			policy.Strategies.RemoveDuplicates.Enabled = true
		case "interpodantiaffinity":
			policy.Strategies.RemovePodsViolatingInterPodAntiAffinity.Enabled = true
		case "lownodeutilization":
			policy.Strategies.LowNodeUtilization.Enabled = true
			if len(strategy.Params) > 0 {
				for _, param := range strategy.Params {
					if !strings.Contains(strings.ToUpper(param.Name), strings.ToUpper("target")) {
						switch param.Name {
						case "cputhreshold":
							policy.Strategies.LowNodeUtilization.Params.NodeResourceUtilizationThresholds.Thresholds.CPU, _ = strconv.Atoi(param.Value)
						case "memorythreshold":
							policy.Strategies.LowNodeUtilization.Params.NodeResourceUtilizationThresholds.Thresholds.Memory, _ = strconv.Atoi(param.Value)
						case "podsthreshold":
							policy.Strategies.LowNodeUtilization.Params.NodeResourceUtilizationThresholds.Thresholds.Pods, _ = strconv.Atoi(param.Value)
						}
					} else {
						switch param.Name {
						case "cputargetthreshold":
							policy.Strategies.LowNodeUtilization.Params.NodeResourceUtilizationThresholds.TargetThresholds.CPU, _ = strconv.Atoi(param.Value)
						case "memorytargetthreshold":
							policy.Strategies.LowNodeUtilization.Params.NodeResourceUtilizationThresholds.TargetThresholds.Memory, _ = strconv.Atoi(param.Value)
						case "podstargetthreshold":
							policy.Strategies.LowNodeUtilization.Params.NodeResourceUtilizationThresholds.TargetThresholds.Pods, _ = strconv.Atoi(param.Value)
						}
					}
					if param.Name == "nodes" {
						policy.Strategies.LowNodeUtilization.Params.NodeResourceUtilizationThresholds.NumberOfNodes, _ = strconv.Atoi(param.Value)
					}
				}
			}
		case "nodeaffinity":
			policy.Strategies.RemovePodsViolatingNodeAffinity.Enabled = true
			nodeAffinity := []string{"requiredDuringSchedulingIgnoredDuringExecution"}
			policy.Strategies.RemovePodsViolatingNodeAffinity.Params.NodeAffinityType = append(nodeAffinity, policy.Strategies.RemovePodsViolatingNodeAffinity.Params.NodeAffinityType...)
		default:
			// Accept no other strategy except for the valid ones.
		}
	}
	policyContent, err := yaml.Marshal(&policy)
	if err != nil {
		log.Fatalf("error: %v", err)
	}
	return string(policyContent)
}

// CheckIfPropertyChanges checks if there is any chnage in the config map
func CheckIfPropertyChanges(strategies []deschedulerv1alpha1.Strategy, existingStrategies map[string]string) bool {
	policyString := existingStrategies["policy.yaml"]
	policy := &Policy{}
	// currentPolicyString := "apiVersion: \"descheduler/v1alpha1\"\nkind:  \"DeschedulerPolicy\"\nstrategies:\n" + generateConfigMapString(strategies, policy)
	currentPolicyString := generateConfigMapString(strategies, *policy)
	log.Printf("\n%v, \n%v", policyString, currentPolicyString)
	return policyString == currentPolicyString
}
