package domainpack

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

// Pack is operator domain configuration loaded from YAML.
type Pack struct {
	Name        string            `yaml:"name"`
	Description string            `yaml:"description"`
	Instruction string            `yaml:"instruction"`
	Roles       []Role            `yaml:"roles"`
	EventTypes  []EventType       `yaml:"event_types"`
	Metadata    map[string]string `yaml:"metadata"`
}

// Role maps an operator role to a recommended curation strategy.
type Role struct {
	Name               string  `yaml:"name"`
	CurationStrategy   int     `yaml:"curation_strategy"`
	DiversityPercent   float64 `yaml:"diversity_percent"`
	DigestSize         int     `yaml:"digest_size"`
}

// EventType defines default severity for a domain event type.
type EventType struct {
	Name             string `yaml:"name"`
	DefaultSeverity  string `yaml:"default_severity"`
	EscalateIfRepeat int    `yaml:"escalate_if_repeat"`
}

// Load reads a domain pack YAML file from disk.
func Load(path string) (*Pack, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("domainpack: read %s: %w", path, err)
	}
	var p Pack
	if err := yaml.Unmarshal(b, &p); err != nil {
		return nil, fmt.Errorf("domainpack: parse %s: %w", path, err)
	}
	if p.Instruction == "" {
		return nil, fmt.Errorf("domainpack: %s missing instruction", path)
	}
	if p.Name == "" {
		p.Name = path
	}
	return &p, nil
}

// DefaultInstruction returns the pack instruction or fallback.
func (p *Pack) DefaultInstruction(fallback string) string {
	if p == nil || p.Instruction == "" {
		return fallback
	}
	return p.Instruction
}
