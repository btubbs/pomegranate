package pomegranate

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"strings"
)

// This file should contain only private, mostly pure functions.  They should
// not interact with the filesystem or database.

func nameInMigrationList(name string, migrations []Migration) bool {
	for _, mig := range migrations {
		if name == mig.Name {
			return true
		}
	}
	return false
}

func nameInState(name string, state []MigrationRecord) bool {
	for _, mig := range state {
		if name == mig.Name {
			return true
		}
	}
	return false
}

func getConfirm(toRun []Migration, forwardBack string, input io.Reader) error {
	names := []string{}
	for _, mig := range toRun {
		names = append(names, mig.Name)
	}
	fmt.Printf(
		"%s migrations that will be run:\n%s\nRun these migrations? (y/n) ",
		forwardBack,
		strings.Join(names, "\n"),
	)
	reader := bufio.NewReader(input)
	resp, err := reader.ReadString('\n')
	if err != nil {
		return err
	}

	switch resp = strings.TrimSpace(resp); resp {
	case "y":
		return nil
	case "n":
		return errors.New("cancelled")
	}
	return fmt.Errorf("Invalid option: %s", resp)
}

// getForwardMigrations takes a state of already run migrations, and the list
// of all migrations, and returns all that haven't been run yet.  Error if the
// state is out of sync with the allMigrations list.
func getForwardMigrations(state []MigrationRecord, allMigrations []Migration) ([]Migration, error) {
	stateCount := len(state)
	migCount := len(allMigrations)
	if stateCount > migCount {
		return nil, errors.New("migration state longer than static list")
	}

	for i := 0; i < stateCount; i++ {
		if state[i].Name != allMigrations[i].Name {
			return nil, fmt.Errorf(
				"migration %d from state (%s) does not match name from static list (%s)",
				i+1, state[i].Name, allMigrations[i].Name,
			)
		}
	}
	return allMigrations[stateCount:], nil
}

func trimMigrationsTail(newtail string, migrations []Migration) ([]Migration, error) {
	trimmed := []Migration{}
	for _, mig := range migrations {
		trimmed = append(trimmed, mig)
		if mig.Name == newtail {
			return trimmed, nil
		}
	}
	return nil, fmt.Errorf("migration %s not found", newtail)
}

// getForwardMigrationsToRun returns all the forward migrations that have not yet been run, up to
// and including the one named in the first argument.
func getForwardMigrationsToRun(name string, state []MigrationRecord, allMigrations []Migration) ([]Migration, error) {
	if len(allMigrations) == 0 {
		return nil, errors.New("no migrations provided")
	}
	if nameInState(name, state) {
		fmt.Printf("migration '%s' has already been run\n", name)
		return []Migration{}, nil
	}
	if name == "" {
		name = allMigrations[len(allMigrations)-1].Name
	}
	forwardMigrations, err := getForwardMigrations(state, allMigrations)
	if err != nil {
		return nil, err
	}
	if len(forwardMigrations) == 0 {
		return []Migration{}, nil
	}
	if !nameInMigrationList(name, forwardMigrations) {
		return nil, fmt.Errorf("migration '%s' not in list of un-run migrations")
	}
	// trim forwardMigrations later than name
	return trimMigrationsTail(name, forwardMigrations)
}

// getMigrationsToReverse takes the name that you're rolling back to, state of
// all migrations run so far, and an ordered list of all possible migrations.
func getMigrationsToReverse(name string, state []MigrationRecord, allMigrations []Migration) ([]Migration, error) {
	// get name of most recent migration
	latest := state[len(state)-1].Name
	// trim allMigrations to ignore anything newer than latest in state.
	reversableMigrations, err := trimMigrationsTail(latest, allMigrations)
	if err != nil {
		return nil, err
	}

	// reversableMigrations and state should now be the same length
	if le, lh := len(reversableMigrations), len(state); le != lh {
		return nil, fmt.Errorf(
			"state in DB has %d migrations, but we have source for %d migrations up to and including %s",
			lh, le, latest,
		)
	}
	// loop backward over state and allmigrations, asserting that names match,
	// and building list of migrations that need running, until we get to the name
	// we're looking for.
	// If we fall off the end, error.
	toRun := []Migration{}
	for i := len(state) - 1; i >= 0; i-- {
		if state[i].Name != reversableMigrations[i].Name {
			return nil, fmt.Errorf(
				"migration %d from state (%s) does not match name from static list (%s)",
				i+1, state[i].Name, reversableMigrations[i].Name,
			)
		}
		toRun = append(toRun, reversableMigrations[i])
		if state[i].Name == name {
			return toRun, nil
		}
	}
	return nil, fmt.Errorf("migration %s not in state", name)
}
