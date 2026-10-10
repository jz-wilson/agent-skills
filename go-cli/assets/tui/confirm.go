package main

import (
	"os"

	"charm.land/huh/v2"
)

// confirm asks a yes/no question. ACCESSIBLE=1 swaps the TUI for plain line prompts.
func confirm(title string) (bool, error) {
	var ok bool
	err := huh.NewForm(huh.NewGroup(
		huh.NewConfirm().Title(title).Value(&ok),
	)).WithAccessible(os.Getenv("ACCESSIBLE") != "").Run()
	return ok, err
}
