package main

import (
	"fmt"
)

var (
	gitCommit string
	gitTag string
	builtAt string
	builtBy string
)

type Manifest struct {}

func (m *Manifest) GetRevision() string {
	return gitCommit
}

func (m *Manifest) GetVersion() string {
	return gitTag
}

func (m *Manifest) String() (string, bool) {
	ok := false
	s := ""

	position := m.GetVersion()
	if len(position) == 0 {
		position = m.GetRevision()
	}

	if len(position) > 0 {
		ok = true
		s += fmt.Sprintf(" revision[%s]", position)
	}

	if len(builtAt) > 0 {
		ok = true
		s += fmt.Sprintf(" built @ %s", builtAt)
	}

	if len(builtBy) > 0 {
		ok = true
		s += fmt.Sprintf(" by '%s'", builtBy)
	}

	return s, ok
}
