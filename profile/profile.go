package profile

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

type OpcodeInstruction struct {
	Opcode string   `yaml:"opcode"`
	Funct  []string `yaml:"funct"`
}

// VMProfile represents the configuration for a specific VM.
type VMProfile struct {
	VMName         string              `yaml:"vm"`
	GOOS           string              `yaml:"goos"`
	GOARCH         string              `yaml:"goarch"`
	AllowedOpcodes []OpcodeInstruction `yaml:"allowed_opcodes"`
	AllowedSycalls []int               `yaml:"allowed_syscalls"`
	NOOPSyscalls   []int               `yaml:"noop_syscalls"`
}

func (p *VMProfile) SetDefaults() {
	if p.GOOS == "" {
		p.GOOS = "linux"
	}
	if p.GOARCH == "" {
		p.GOARCH = "mips"
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
	if err = yaml.NewDecoder(file).Decode(&profile); err != nil {
		return nil, fmt.Errorf("failed to parse profile: %w", err)
	}
	return &profile, nil
}
