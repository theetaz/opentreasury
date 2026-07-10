// Package connector maps source-system CSV exports onto the OpenTreasury
// interchange format using declarative YAML mapping profiles, and delivers
// them to the staging API with at-least-once semantics.
package connector

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

// Profile is a declarative mapping from a source file's columns to the
// interchange format. Profiles are versioned artifacts reviewed like code —
// domain experts can author them without touching Go.
type Profile struct {
	SourceSystem string `yaml:"source_system"`
	Version      int    `yaml:"version"`
	Columns      struct {
		SourceRef   string `yaml:"source_ref"`
		OccurredAt  string `yaml:"occurred_at"`
		Institution string `yaml:"institution"`
		Amount      string `yaml:"amount"`
		Currency    string `yaml:"currency"`
	} `yaml:"columns"`
	// v1 account resolution: every row becomes one balanced entry between a
	// fixed debit and credit account of the reference chart.
	DebitAccount  string `yaml:"debit_account"`
	CreditAccount string `yaml:"credit_account"`
}

func (p Profile) Name() string {
	return fmt.Sprintf("%s@%d", p.SourceSystem, p.Version)
}

func (p Profile) validate() error {
	switch {
	case p.SourceSystem == "":
		return fmt.Errorf("profile missing source_system")
	case p.Version <= 0:
		return fmt.Errorf("profile missing version")
	case p.Columns.SourceRef == "" || p.Columns.OccurredAt == "" || p.Columns.Institution == "" ||
		p.Columns.Amount == "" || p.Columns.Currency == "":
		return fmt.Errorf("profile missing column mappings")
	case p.DebitAccount == "" || p.CreditAccount == "":
		return fmt.Errorf("profile missing debit_account/credit_account")
	}
	return nil
}

func LoadProfile(path string) (Profile, error) {
	contents, err := os.ReadFile(path) // #nosec G304 -- operator-supplied config path
	if err != nil {
		return Profile{}, err
	}

	var profile Profile
	if err := yaml.Unmarshal(contents, &profile); err != nil {
		return Profile{}, fmt.Errorf("parsing profile: %w", err)
	}
	if err := profile.validate(); err != nil {
		return Profile{}, err
	}

	return profile, nil
}
