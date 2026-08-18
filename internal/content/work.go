package content

import (
	"fmt"

	"gopkg.in/yaml.v3"
)

// Job is one entry of the pack's work section: one role, held at one employer,
// with the highlights written for it and the links that back it up.
//
// Nothing here is a summary of the section. The WORK page's count of roles is
// derived from how many of these the pack carries, which is the rule the whole
// surface is built on - an aggregate claim is counted from the itemized facts,
// never written down beside them.
type Job struct {
	Slug     string `yaml:"slug"`
	Company  string `yaml:"company"`
	Role     string `yaml:"role"`
	Start    string `yaml:"start"`
	End      string `yaml:"end"` // empty for a role still held
	Location string `yaml:"location"`

	Highlights []string   `yaml:"highlights"`
	Links      NamedLinks `yaml:"links"`

	// Project is the projects-section slug this role's work is also written up
	// under, where the pack carries one. The work pages do not follow it - work
	// drills into the employer, and the spec routes project detail from the
	// projects and awards sections instead.
	Project string `yaml:"project"`
}

// parseWork parses raw work.yaml bytes into the roles it lists. Like
// parseIdentity and parseLinks it is factored out of Load so tests can run
// against a fixture rather than the fetched pack.
func parseWork(b []byte) ([]Job, error) {
	var doc struct {
		Work []Job `yaml:"work"`
	}
	if err := yaml.Unmarshal(b, &doc); err != nil {
		return nil, fmt.Errorf("parse work section: %w", err)
	}
	return doc.Work, nil
}
