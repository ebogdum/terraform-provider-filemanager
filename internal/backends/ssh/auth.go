// SPDX-License-Identifier: MIT

package ssh

import (
	"bytes"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strings"

	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/agent"
	"golang.org/x/crypto/ssh/knownhosts"
)

// ParsePrivateKey parses a private key, optionally with a passphrase.
func ParsePrivateKey(keyData []byte, passphrase string) (ssh.Signer, error) {
	var signer ssh.Signer
	var err error

	if passphrase != "" {
		signer, err = ssh.ParsePrivateKeyWithPassphrase(keyData, []byte(passphrase))
	} else {
		signer, err = ssh.ParsePrivateKey(keyData)
	}

	if err != nil {
		return nil, fmt.Errorf("failed to parse private key: %w", err)
	}

	return signer, nil
}

// LoadPrivateKeyFile loads a private key from a file path.
func LoadPrivateKeyFile(keyPath string, passphrase string) (ssh.Signer, error) {
	// Expand ~ to home directory
	if strings.HasPrefix(keyPath, "~") {
		home, err := os.UserHomeDir()
		if err != nil {
			return nil, fmt.Errorf("failed to get home directory: %w", err)
		}
		keyPath = filepath.Join(home, keyPath[1:])
	}

	keyData, err := os.ReadFile(keyPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read private key file %s: %w", keyPath, err)
	}

	return ParsePrivateKey(keyData, passphrase)
}

// SSHAgentAuth returns an SSH agent authentication method.
func SSHAgentAuth() (ssh.AuthMethod, error) {
	socket := os.Getenv("SSH_AUTH_SOCK")
	if socket == "" {
		return nil, fmt.Errorf("SSH_AUTH_SOCK environment variable not set")
	}

	conn, err := net.Dial("unix", socket)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to SSH agent: %w", err)
	}

	agentClient := agent.NewClient(conn)
	return ssh.PublicKeysCallback(agentClient.Signers), nil
}

// ParseKnownHosts parses known_hosts data and returns a host key callback.
func ParseKnownHosts(data []byte) (ssh.HostKeyCallback, error) {
	// Create a temporary file with the known hosts data
	tmpFile, err := os.CreateTemp("", "known_hosts")
	if err != nil {
		return nil, fmt.Errorf("failed to create temp file: %w", err)
	}
	defer os.Remove(tmpFile.Name())

	if _, err := tmpFile.Write(data); err != nil {
		tmpFile.Close()
		return nil, fmt.Errorf("failed to write known hosts data: %w", err)
	}
	tmpFile.Close()

	callback, err := knownhosts.New(tmpFile.Name())
	if err != nil {
		return nil, fmt.Errorf("failed to parse known hosts: %w", err)
	}

	return callback, nil
}

// LoadKnownHostsFile loads known_hosts from a file path.
func LoadKnownHostsFile(path string) (ssh.HostKeyCallback, error) {
	// Expand ~ to home directory
	if strings.HasPrefix(path, "~") {
		home, err := os.UserHomeDir()
		if err != nil {
			return nil, fmt.Errorf("failed to get home directory: %w", err)
		}
		path = filepath.Join(home, path[1:])
	}

	callback, err := knownhosts.New(path)
	if err != nil {
		return nil, fmt.Errorf("failed to load known hosts from %s: %w", path, err)
	}

	return callback, nil
}

// DefaultKnownHostsCallback returns a host key callback using the default known_hosts file.
// Returns an error if the known_hosts file does not exist.
// Use insecure_skip_host_key = true in service configuration for explicit insecure mode.
func DefaultKnownHostsCallback() (ssh.HostKeyCallback, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("failed to get home directory: %w", err)
	}

	knownHostsPath := filepath.Join(home, ".ssh", "known_hosts")

	// Check if the file exists - SECURITY: Do NOT fall back to insecure mode
	if _, err := os.Stat(knownHostsPath); os.IsNotExist(err) {
		return nil, fmt.Errorf("known_hosts file not found at %s. Either create the file, specify known_hosts_file, or set insecure_skip_host_key = true (not recommended)", knownHostsPath)
	}

	return knownhosts.New(knownHostsPath)
}

// HostKeyCallbackFromString creates a host key callback from a single host key string.
// The format is: "ssh-rsa AAAA..." or "ssh-ed25519 AAAA..."
func HostKeyCallbackFromString(hostKey string) (ssh.HostKeyCallback, error) {
	if hostKey == "" {
		return nil, fmt.Errorf("host key string is empty")
	}

	// Parse the public key
	pubKey, _, _, _, err := ssh.ParseAuthorizedKey([]byte(hostKey))
	if err != nil {
		return nil, fmt.Errorf("failed to parse host key: %w", err)
	}

	// Create a callback that accepts only this specific key
	return func(hostname string, remote net.Addr, key ssh.PublicKey) error {
		if !bytes.Equal(key.Marshal(), pubKey.Marshal()) {
			return fmt.Errorf("host key mismatch for %s", hostname)
		}
		return nil
	}, nil
}

// KeyboardInteractiveAuth creates a keyboard-interactive authentication method.
// The answers map should contain question-answer pairs for the expected prompts.
func KeyboardInteractiveAuth(answers map[string]string) ssh.AuthMethod {
	return ssh.KeyboardInteractive(func(user, instruction string, questions []string, echos []bool) ([]string, error) {
		responses := make([]string, len(questions))
		for i, question := range questions {
			// Try to find a matching answer
			for key, answer := range answers {
				if strings.Contains(strings.ToLower(question), strings.ToLower(key)) {
					responses[i] = answer
					break
				}
			}
		}
		return responses, nil
	})
}

// CombinedAuth creates an authentication method that tries multiple methods in sequence.
func CombinedAuth(methods ...ssh.AuthMethod) ssh.AuthMethod {
	// Return first non-nil method for simplicity
	// In practice, SSH client tries all configured methods
	for _, m := range methods {
		if m != nil {
			return m
		}
	}
	return nil
}

// AuthConfig holds authentication configuration options.
type AuthConfig struct {
	// Private key authentication
	PrivateKey     []byte // PEM-encoded private key
	PrivateKeyPath string // Path to private key file
	Passphrase     string // Passphrase for encrypted private key

	// Password authentication
	Password string

	// SSH agent
	UseAgent bool

	// Host key verification
	HostKey          string // Single host key string
	KnownHostsPath   string // Path to known_hosts file
	KnownHostsData   []byte // Inline known_hosts data
	InsecureNoVerify bool   // Skip host key verification (not recommended)
}

// BuildAuthMethods builds a list of authentication methods from the config.
func BuildAuthMethods(config AuthConfig) ([]ssh.AuthMethod, error) {
	var methods []ssh.AuthMethod

	// Private key from data
	if len(config.PrivateKey) > 0 {
		signer, err := ParsePrivateKey(config.PrivateKey, config.Passphrase)
		if err != nil {
			return nil, fmt.Errorf("failed to parse private key: %w", err)
		}
		methods = append(methods, ssh.PublicKeys(signer))
	}

	// Private key from file
	if config.PrivateKeyPath != "" && len(config.PrivateKey) == 0 {
		signer, err := LoadPrivateKeyFile(config.PrivateKeyPath, config.Passphrase)
		if err != nil {
			return nil, fmt.Errorf("failed to load private key file: %w", err)
		}
		methods = append(methods, ssh.PublicKeys(signer))
	}

	// SSH agent
	if config.UseAgent {
		agentAuth, err := SSHAgentAuth()
		if err == nil {
			methods = append(methods, agentAuth)
		}
		// Ignore agent errors - might not be available
	}

	// Password authentication
	if config.Password != "" {
		methods = append(methods, ssh.Password(config.Password))
	}

	// If no methods configured, try SSH agent as fallback
	if len(methods) == 0 {
		agentAuth, err := SSHAgentAuth()
		if err == nil {
			methods = append(methods, agentAuth)
		}
	}

	return methods, nil
}

// BuildHostKeyCallback builds a host key callback from the config.
func BuildHostKeyCallback(config AuthConfig) (ssh.HostKeyCallback, error) {
	// Insecure mode
	if config.InsecureNoVerify {
		return ssh.InsecureIgnoreHostKey(), nil
	}

	// Single host key
	if config.HostKey != "" {
		return HostKeyCallbackFromString(config.HostKey)
	}

	// Known hosts from data
	if len(config.KnownHostsData) > 0 {
		return ParseKnownHosts(config.KnownHostsData)
	}

	// Known hosts from file
	if config.KnownHostsPath != "" {
		return LoadKnownHostsFile(config.KnownHostsPath)
	}

	// Default known_hosts
	return DefaultKnownHostsCallback()
}

// LoadDefaultPrivateKey attempts to load a private key from default locations.
func LoadDefaultPrivateKey(passphrase string) (ssh.Signer, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("failed to get home directory: %w", err)
	}

	// Default key locations to try
	keyFiles := []string{
		filepath.Join(home, ".ssh", "id_ed25519"),
		filepath.Join(home, ".ssh", "id_ecdsa"),
		filepath.Join(home, ".ssh", "id_rsa"),
		filepath.Join(home, ".ssh", "id_dsa"),
	}

	for _, keyPath := range keyFiles {
		if _, err := os.Stat(keyPath); os.IsNotExist(err) {
			continue
		}

		signer, err := LoadPrivateKeyFile(keyPath, passphrase)
		if err == nil {
			return signer, nil
		}
	}

	return nil, fmt.Errorf("no default private key found")
}
