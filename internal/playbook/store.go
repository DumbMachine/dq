package playbook

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/dumbmachine/db-cli/internal/config"
	"gopkg.in/yaml.v3"
)

// Meta holds playbook metadata from YAML frontmatter.
type Meta struct {
	Name        string   `yaml:"name" json:"name"`
	Description string   `yaml:"description" json:"description"`
	Tags        []string `yaml:"tags,omitempty" json:"tags,omitempty"`
	Connections []string `yaml:"connections,omitempty" json:"connections,omitempty"`
	Created     string   `yaml:"created" json:"created"`
	Updated     string   `yaml:"updated" json:"updated"`
}

// Playbook holds metadata and markdown content.
type Playbook struct {
	Meta    `yaml:",inline"`
	Content string `json:"content"`
}

// PlaybooksDir returns the directory where playbooks are stored.
func PlaybooksDir() string {
	return filepath.Join(config.ConfigDir(), "playbooks")
}

// PlaybookPath returns the file path for a named playbook.
func PlaybookPath(name string) string {
	return filepath.Join(PlaybooksDir(), name+".md")
}

// List returns metadata for all stored playbooks.
func List() ([]Meta, error) {
	dir := PlaybooksDir()
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("reading playbooks dir: %w", err)
	}

	var playbooks []Meta
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".md") {
			continue
		}
		name := strings.TrimSuffix(e.Name(), ".md")
		pb, err := Load(name)
		if err != nil {
			continue
		}
		playbooks = append(playbooks, pb.Meta)
	}
	return playbooks, nil
}

// Load reads a playbook by name.
func Load(name string) (*Playbook, error) {
	path := PlaybookPath(name)
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("playbook %q not found", name)
		}
		return nil, fmt.Errorf("reading playbook: %w", err)
	}

	return Parse(string(data))
}

// Parse parses a playbook from raw markdown with YAML frontmatter.
func Parse(raw string) (*Playbook, error) {
	fm, body := parseFrontmatter(raw)
	if fm == "" {
		return nil, fmt.Errorf("playbook must have YAML frontmatter (--- delimited)")
	}

	var meta Meta
	if err := yaml.Unmarshal([]byte(fm), &meta); err != nil {
		return nil, fmt.Errorf("parsing playbook frontmatter: %w", err)
	}
	if meta.Name == "" {
		return nil, fmt.Errorf("playbook frontmatter must include 'name'")
	}

	return &Playbook{
		Meta:    meta,
		Content: body,
	}, nil
}

// Save writes a playbook to disk. If the playbook already exists, it updates it.
func Save(pb *Playbook) error {
	dir := PlaybooksDir()
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("creating playbooks dir: %w", err)
	}

	if pb.Created == "" {
		pb.Created = time.Now().Format("2006-01-02")
	}
	pb.Updated = time.Now().Format("2006-01-02")

	fmBytes, err := yaml.Marshal(&pb.Meta)
	if err != nil {
		return fmt.Errorf("marshaling playbook metadata: %w", err)
	}

	content := "---\n" + string(fmBytes) + "---\n" + pb.Content
	return os.WriteFile(PlaybookPath(pb.Name), []byte(content), 0644)
}

// Remove deletes a playbook by name.
func Remove(name string) error {
	err := os.Remove(PlaybookPath(name))
	if errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("playbook %q not found", name)
	}
	return err
}

// Template returns a starter playbook markdown string.
func Template(name string) string {
	return fmt.Sprintf(`---
name: %s
description: ""
tags: []
connections: []
---

# %s

## Overview
Describe the goal — what question does this playbook answer or what workflow does it encode?

## Procedure
1. Connect and discover the schema
2. Run the analysis queries
3. Interpret the results
4. Generate charts if needed
5. Summarize findings

## SQL Templates
`+"```sql"+`
-- Add your SQL queries here
SELECT 1;
`+"```"+`

## Specifications
- Expected output format or postconditions
- Data quality requirements

## Advice
- Tips for getting accurate results
- Known edge cases or data quirks

## Forbidden Actions
- Do not run mutations without confirmation
- Do not expose PII columns in output
`, name, strings.ReplaceAll(name, "-", " "))
}

// parseFrontmatter splits a markdown document into YAML frontmatter and body.
func parseFrontmatter(raw string) (string, string) {
	const sep = "---\n"
	if !strings.HasPrefix(raw, sep) {
		return "", raw
	}
	rest := raw[len(sep):]
	idx := strings.Index(rest, "\n---\n")
	if idx == -1 {
		// Try ending with --- at EOF
		idx = strings.Index(rest, "\n---")
		if idx == -1 {
			return "", raw
		}
		return rest[:idx], ""
	}
	return rest[:idx], rest[idx+len("\n---\n"):]
}
