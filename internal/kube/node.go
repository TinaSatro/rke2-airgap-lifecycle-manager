// Package kube provides helpers for interacting with the Kubernetes cluster
// via kubectl. All functions assume kubectl is installed and kubeconfig is
// configured at ~/.kube/config.
package kube

import (
	"fmt"
	"os/exec"
	"strconv"
	"strings"
)

// NodeInfo holds the hardware summary parsed from kubectl describe node.
type NodeInfo struct {
	CPU     int
	MemGB   float64
	DiskGB  float64
	Profile string // edge | small | medium | large
}

// GetNodeInfo runs kubectl describe node and parses cpu / memory / storage.
func GetNodeInfo() (*NodeInfo, error) {
	out, err := exec.Command("kubectl", "describe", "node").Output()
	if err != nil {
		return nil, fmt.Errorf("kubectl describe node failed: %w", err)
	}

	info := &NodeInfo{}
	for _, line := range strings.Split(string(out), "\n") {
		line = strings.TrimSpace(line)
		parts := strings.Fields(line)
		if len(parts) < 2 {
			continue
		}
		switch {
		case strings.HasPrefix(line, "cpu:"):
			info.CPU, _ = strconv.Atoi(parts[1])
		case strings.HasPrefix(line, "memory:"):
			info.MemGB = parseKiToGB(parts[1])
		case strings.HasPrefix(line, "ephemeral-storage:"):
			info.DiskGB = parseKiToGB(parts[1])
		}
	}

	info.Profile = detectProfile(info)
	return info, nil
}

// GetNodeIP returns the primary IP address of the first cluster node.
func GetNodeIP() (string, error) {
	out, err := exec.Command("kubectl", "get", "nodes",
		"-o", "jsonpath={.items[0].status.addresses[0].address}").Output()
	if err != nil {
		return "", fmt.Errorf("kubectl get nodes failed: %w", err)
	}
	return strings.TrimSpace(string(out)), nil
}

// PrintNodeInfo writes a human-readable cluster summary to stdout.
func PrintNodeInfo(n *NodeInfo) {
	fmt.Printf("\n=== Cluster Info ===\n")
	fmt.Printf("Profile : %s\n", n.Profile)
	fmt.Printf("CPU     : %d cores\n", n.CPU)
	fmt.Printf("Memory  : %.1f GB\n", n.MemGB)

	if n.Profile == "edge" {
		// Edge nodes can optionally run the observability stack if they
		// meet the minimum hardware bar (6 CPU / 15 GB RAM).
		insufficient := []string{}
		if n.CPU < 6 {
			insufficient = append(insufficient, fmt.Sprintf("CPU: have %d, need 6", n.CPU))
		}
		if n.MemGB < 14.5 {
			insufficient = append(insufficient, fmt.Sprintf("RAM: have %.1fGB, need 15GB", n.MemGB))
		}
		if len(insufficient) == 0 {
			fmt.Println("Observability : available")
		} else {
			fmt.Printf("Observability : not available (%s)\n", strings.Join(insufficient, ", "))
		}
	}
}

// detectProfile maps CPU core count to a named deployment size profile.
// Profiles gate optional platform components and storage sizing.
func detectProfile(n *NodeInfo) string {
	switch {
	case n.CPU <= 6:
		return "edge"
	case n.CPU <= 8:
		return "small"
	case n.CPU <= 12:
		return "medium"
	default:
		return "large"
	}
}

// parseKiToGB converts a Kubernetes resource string in Ki to gigabytes.
func parseKiToGB(s string) float64 {
	if strings.HasSuffix(s, "Ki") {
		v, _ := strconv.ParseFloat(strings.TrimSuffix(s, "Ki"), 64)
		return v / 1024 / 1024
	}
	return 0
}