package profile

import (
	"encoding/json"
	"fmt"
	"os"
)

// VMProfile represents the configuration for a specific VM.
type VMProfile struct {
	VMName             string   `json:"vm"`
	AllowedOpcodes     []string `json:"allowed_opcodes"`
	RestrictedSyscalls []string `json:"restricted_syscalls"`
}

// LoadProfile loads a VM profile from a JSON file.
func LoadProfile(filename string) (*VMProfile, error) {
	file, err := os.Open(filename)
	if err != nil {
		return nil, fmt.Errorf("failed to open profile: %w", err)
	}
	defer file.Close()

	var profile VMProfile
	if err := json.NewDecoder(file).Decode(&profile); err != nil {
		return nil, fmt.Errorf("failed to parse profile: %w", err)
	}
	return &profile, nil
}
