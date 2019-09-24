package main

import (
	"fmt"
	"log"
	"os"

	"gopkg.in/yaml.v2"
)

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

func main() {
	desc := Policy{}
	desc.APIVersion = "descheduler/v1alpha1"
	desc.Kind = "DeschedulerPolicy"
	desc.Strategies.RemoveDuplicates.Enabled = true

	desc.Strategies.RemovePodsViolatingNodeAffinity.Enabled = true

	nodeAffinity := []string{"requiredDuringSchedulingIgnoredDuringExecution"}

	desc.Strategies.RemovePodsViolatingNodeAffinity.Params.NodeAffinityType = append(nodeAffinity, desc.Strategies.RemovePodsViolatingNodeAffinity.Params.NodeAffinityType...)

	d, err := yaml.Marshal(&desc)
	if err != nil {
		log.Fatalf("error: %v", err)
	}
	fmt.Printf("--- t dump:\n%s\n\n", string(d))

	f, err := os.Create("deshedular.txt")
	if err != nil {
		fmt.Println(err)
	}
	l, err := f.WriteString(string(d))
	if err != nil {
		fmt.Println(err)
		f.Close()
	}
	fmt.Println(l, "bytes written successfully")
	f.Close()
}
