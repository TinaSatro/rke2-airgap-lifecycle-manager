package config

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const configFile = ".lifecycle.conf"

// Config holds the credentials required by both install and upgrade commands.
// Values are read from ~/.lifecycle.conf — never hardcoded.
type Config struct {
	ArtifactoryKey string // X-JFrog-Art-Api header value
	DockerUser     string // Docker Hub username for OCI registry auth
	DockerToken    string // Docker Hub token / PAT
}

// Load reads ~/.lifecycle.conf and returns a populated Config.
// Returns an error if the file is missing or any required key is absent.
func Load() (*Config, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("cannot determine home directory: %w", err)
	}

	path := filepath.Join(home, configFile)
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf(
			"config file not found: %s\n\nCreate it with:\n  ARTIFACTORY_KEY=...\n  DOCKER_USER=...\n  DOCKER_TOKEN=...",
			path,
		)
	}
	defer f.Close()

	cfg := &Config{}
	scanner := bufio.NewScanner(f)

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		// skip blank lines and comments
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			continue
		}

		key := strings.TrimSpace(parts[0])
		val := strings.TrimSpace(parts[1])

		switch key {
		case "ARTIFACTORY_KEY":
			cfg.ArtifactoryKey = val
		case "DOCKER_USER":
			cfg.DockerUser = val
		case "DOCKER_TOKEN":
			cfg.DockerToken = val
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("error reading config: %w", err)
	}

	if err := cfg.validate(); err != nil {
		return nil, err
	}

	return cfg, nil
}

// validate checks that all required fields are present.
func (c *Config) validate() error {
	missing := []string{}

	if c.ArtifactoryKey == "" {
		missing = append(missing, "ARTIFACTORY_KEY")
	}
	if c.DockerUser == "" {
		missing = append(missing, "DOCKER_USER")
	}
	if c.DockerToken == "" {
		missing = append(missing, "DOCKER_TOKEN")
	}

	if len(missing) > 0 {
		return fmt.Errorf("missing required config values: %s", strings.Join(missing, ", "))
	}

	return nil
}
