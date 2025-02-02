package sshmgmt

import (
	"fmt"
	"sync"
	"time"

	"yourproject/oqs_vault"
)

// SSHConnection represents a saved or temporary SSH connection.
type SSHConnection struct {
	Name       string
	Address    string
	Username   string
	PrivateKey string
	CreatedAt  time.Time
	Temporary  bool
}

// SSHManager encapsulates SSH connection management.
type SSHManager struct {
	mu          sync.Mutex
	connections map[string]*SSHConnection
	vault       *oqs_vault.VaultManager // Post-Quantum secure vault
}

// NewSSHManager initializes the SSHManager with a secure vault.
func NewSSHManager(vault *oqs_vault.VaultManager) *SSHManager {
	return &SSHManager{
		connections: make(map[string]*SSHConnection),
		vault:       vault,
	}
}

// SaveConnection saves a new or updated SSH connection securely.
func (m *SSHManager) SaveConnection(name, address, username, privateKey string, temporary bool) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	conn := &SSHConnection{
		Name:       name,
		Address:    address,
		Username:   username,
		PrivateKey: privateKey,
		CreatedAt:  time.Now(),
		Temporary:  temporary,
	}

	// Encrypt and store the connection securely in the vault
	encryptedKey := fmt.Sprintf("ssh_conn_%s", name)
	if err := m.vault.Store(encryptedKey, conn); err != nil {
		return fmt.Errorf("failed to store connection in vault: %w", err)
	}

	m.connections[name] = conn
	return nil
}

// GetConnection retrieves an SSH connection by name.
func (m *SSHManager) GetConnection(name string) (*SSHConnection, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	conn, exists := m.connections[name]
	if !exists {
		return nil, fmt.Errorf("connection '%s' not found", name)
	}
	return conn, nil
}

// ListConnections lists all saved SSH connections.
func (m *SSHManager) ListConnections() []SSHConnection {
	m.mu.Lock()
	defer m.mu.Unlock()

	var connections []SSHConnection
	for _, conn := range m.connections {
		connections = append(connections, *conn)
	}
	return connections
}

// DeleteConnection deletes an SSH connection by name.
func (m *SSHManager) DeleteConnection(name string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, exists := m.connections[name]; !exists {
		return fmt.Errorf("connection '%s' not found", name)
	}

	encryptedKey := fmt.Sprintf("ssh_conn_%s", name)
	if err := m.vault.Delete(encryptedKey); err != nil {
		return fmt.Errorf("failed to delete connection from vault: %w", err)
	}

	delete(m.connections, name)
	return nil
}

// CreateTempConnection creates a temporary SSH connection (not saved to disk).
func (m *SSHManager) CreateTempConnection(address, username, privateKey string) (*SSHConnection, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	tempName := fmt.Sprintf("temp_%d", time.Now().UnixNano())
	conn := &SSHConnection{
		Name:       tempName,
		Address:    address,
		Username:   username,
		PrivateKey: privateKey,
		CreatedAt:  time.Now(),
		Temporary:  true,
	}

	m.connections[tempName] = conn
	return conn, nil
}

// CleanupTempConnections removes all temporary connections.
func (m *SSHManager) CleanupTempConnections() {
	m.mu.Lock()
	defer m.mu.Unlock()

	for name, conn := range m.connections {
		if conn.Temporary {
			delete(m.connections, name)
		}
	}
}

// GenerateReport generates a report of all active SSH connections.
func (m *SSHManager) GenerateReport() []string {
	m.mu.Lock()
	defer m.mu.Unlock()

	report := []string{"=== SSH Connections Report ==="}
	for _, conn := range m.connections {
		report = append(report, fmt.Sprintf(
			"Name: %s\nAddress: %s\nUsername: %s\nCreated At: %s\nTemporary: %v\n---",
			conn.Name, conn.Address, conn.Username, conn.CreatedAt.Format(time.RFC3339), conn.Temporary,
		))
	}
	return report
}
