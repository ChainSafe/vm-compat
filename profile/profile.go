package profile

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// VMProfile represents the configuration for a specific VM.
type VMProfile struct {
	VMName         string   `json:"vm"`
	GOOS           string   `json:"goos"`
	GOARCH         string   `json:"goarch"`
	AllowedOpcodes []string `json:"allowed_opcodes"`
	AllowedSycalls []int    `json:"allowed_syscalls"`
}

func (p *VMProfile) SetDefaults() {
	if p.GOOS == "" {
		p.GOOS = "linux"
	}
	if p.GOARCH == "" {
		p.GOARCH = "mips32"
	}
}

// LoadProfile loads a VM profile from a JSON file.
func LoadProfile(filename string) (*VMProfile, error) {
	path, err := filepath.Abs(filename)
	if err != nil {
		return nil, fmt.Errorf("failed to get absolute path of profile: %w", err)
	}

	file, err := os.Open(path)
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
