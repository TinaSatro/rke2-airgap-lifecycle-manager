package kube

import (
	"encoding/base64"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"
)

const (
	// secretWaitTimeout is the maximum time to wait for the storage-user
	// secret to appear after the platform's storage operator initialises.
	secretWaitTimeout = 5 * time.Minute

	// secretPollInterval is how often we check for the secret.
	secretPollInterval = 10 * time.Second
)

// WaitAndFixSecrets waits for the authoritative storage-user secret to appear,
// reads the real credentials from it, and patches the bootstrap secret that
// pods actually mount.
//
// Background: the object-storage access secret is written early in the
// bootstrap sequence with placeholder credentials. The real credentials only
// exist once the storage operator finishes its own init cycle. Pods that mount
// the bootstrap secret crash immediately because they cannot authenticate.
//
// Fix:
//  1. Poll until the authoritative secret appears (deadline loop).
//  2. Read real access key + secret key from it.
//  3. JSON-patch all affected fields in the bootstrap secret.
//  4. Delete the crashed pods so they restart with the correct credentials.
func WaitAndFixSecrets(namespace, storageUserSecret, bootstrapSecret string, dependentApps []string) error {
	fmt.Println("\n→ Waiting for storage-user secret to appear...")

	if err := waitForSecret(namespace, storageUserSecret); err != nil {
		return err
	}

	accessKey, secretKey, endpoint, err := getStorageCredentials(namespace, storageUserSecret)
	if err != nil {
		return fmt.Errorf("cannot read storage credentials: %w", err)
	}
	fmt.Printf("  access key : %s\n", accessKey)
	fmt.Printf("  endpoint   : %s\n", endpoint)

	if err := patchBootstrapSecret(namespace, bootstrapSecret, accessKey, secretKey, endpoint); err != nil {
		return fmt.Errorf("cannot patch bootstrap secret: %w", err)
	}

	restartDependentPods(namespace, dependentApps)
	return nil
}

// waitForSecret polls until the named secret exists in the given namespace
// or the deadline is exceeded.
func waitForSecret(namespace, secretName string) error {
	deadline := time.Now().Add(secretWaitTimeout)
	for time.Now().Before(deadline) {
		out, err := exec.Command("kubectl", "get", "secret",
			secretName, "-n", namespace, "--no-headers").Output()
		if err == nil && len(strings.TrimSpace(string(out))) > 0 {
			fmt.Println()
			return nil
		}
		fmt.Print(".")
		time.Sleep(secretPollInterval)
	}
	fmt.Println()
	return fmt.Errorf("timed out waiting for secret %s/%s", namespace, secretName)
}

// getStorageCredentials reads the real object-storage access and secret keys
// from the authoritative secret created by the storage operator.
func getStorageCredentials(namespace, secretName string) (accessKey, secretKey, endpoint string, err error) {
	out, err := exec.Command("kubectl", "get", "secret", secretName,
		"-n", namespace,
		"-o", "jsonpath={.data.accesskey} {.data.secretkey}").Output()
	if err != nil {
		return "", "", "", fmt.Errorf("kubectl get secret failed: %w", err)
	}

	parts := strings.Fields(string(out))
	if len(parts) < 2 {
		return "", "", "", fmt.Errorf("unexpected secret format")
	}

	ak, err := base64.StdEncoding.DecodeString(parts[0])
	if err != nil {
		return "", "", "", fmt.Errorf("cannot decode access key: %w", err)
	}
	sk, err := base64.StdEncoding.DecodeString(parts[1])
	if err != nil {
		return "", "", "", fmt.Errorf("cannot decode secret key: %w", err)
	}

	// Endpoint is the cluster-internal object storage service address.
	// Defined here as a constant because it does not change between installs.
	endpoint = "<object-storage-svc>:<port>"

	return string(ak), string(sk), endpoint, nil
}

// patchBootstrapSecret overwrites the placeholder credentials in the bootstrap
// secret with the real values using a JSON patch.
// All Secret .data values must be base64-encoded.
func patchBootstrapSecret(namespace, secretName, accessKey, secretKey, endpoint string) error {
	patch := fmt.Sprintf(`[
		{"op":"replace","path":"/data/accesskey",          "value":"%s"},
		{"op":"replace","path":"/data/awsaccesskeyid",     "value":"%s"},
		{"op":"replace","path":"/data/secretkey",          "value":"%s"},
		{"op":"replace","path":"/data/awssecretaccesskey", "value":"%s"},
		{"op":"replace","path":"/data/awsendpoint",        "value":"%s"}
	]`,
		b64(accessKey), b64(accessKey),
		b64(secretKey), b64(secretKey),
		b64(endpoint),
	)

	cmd := exec.Command("kubectl", "patch", "secret", secretName,
		"-n", namespace, "--type=json", "-p", patch)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("kubectl patch secret failed: %w", err)
	}

	fmt.Printf("  %s patched\n", secretName)
	return nil
}

// restartDependentPods deletes pods matching each app label so they restart
// and pick up the patched secret. Kubernetes will reschedule them immediately.
func restartDependentPods(namespace string, apps []string) {
	for _, app := range apps {
		err := exec.Command("kubectl", "delete", "pod",
			"-n", namespace, "-l", "app="+app).Run()
		if err != nil {
			fmt.Printf("  WARNING: could not restart %s: %v\n", app, err)
		} else {
			fmt.Printf("  restarted %s\n", app)
		}
	}
}

// b64 returns the base64-encoded representation of s.
func b64(s string) string {
	return base64.StdEncoding.EncodeToString([]byte(s))
}