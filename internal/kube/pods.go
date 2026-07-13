package kube

import (
	"fmt"
	"os/exec"
	"strings"
)

// PodStatus holds the essential fields from kubectl get pods output.
type PodStatus struct {
	Namespace string
	Name      string
	Ready     string
	Status    string
}

// GetPods returns the status of all pods across all namespaces.
func GetPods() ([]PodStatus, error) {
	out, err := exec.Command("kubectl", "get", "pods", "-A").Output()
	if err != nil {
		return nil, fmt.Errorf("kubectl get pods failed: %w", err)
	}

	var pods []PodStatus
	lines := strings.Split(string(out), "\n")

	// skip header line
	for _, line := range lines[1:] {
		fields := strings.Fields(line)
		if len(fields) < 4 {
			continue
		}
		pods = append(pods, PodStatus{
			Namespace: fields[0],
			Name:      fields[1],
			Ready:     fields[2],
			Status:    fields[3],
		})
	}
	return pods, nil
}

// PrintPods writes pod health to stdout.
// Only unhealthy pods are printed individually — if all pods are healthy
// a single summary line is shown instead.
func PrintPods(pods []PodStatus) {
	fmt.Printf("\n=== Pod Status ===\n")

	allOk := true
	for _, p := range pods {
		if p.Status != "Running" && p.Status != "Completed" {
			fmt.Printf("⚠  %-40s %-20s %s\n", p.Name, p.Namespace, p.Status)
			allOk = false
		}
	}

	if allOk {
		fmt.Printf("All %d pods healthy\n", len(pods))
	}
}