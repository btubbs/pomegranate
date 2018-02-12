package pomegranate

import (
	"bytes"
	"fmt"
	"go/format"
	"io/ioutil"
	"os"
	"path"
	"regexp"
	"strconv"
	"strings"
	"text/template"
)

// IngestMigrations reads all the migrations in the given directory and writes
// them to a Go source file in the same directory.  The generateTag argument
// determins whether the new Go file will contain a "//go:generate" comment to
// tag it for automatic regeneration by "go generate".
func IngestMigrations(dir, goFile, packageName string, generateTag bool) error {
	migs, err := ReadMigrationFiles(dir)
	if err != nil {
		return err
	}
	err = writeGoMigrations(dir, goFile, packageName, migs, generateTag)
	if err != nil {
		return err
	}
	fmt.Printf("Migrations written to %s\n", path.Join(dir, goFile))
	return nil
}

// InitMigration creates a new 00001_init migration in the given directory.
// This migration will contain the SQL commands necessary to create the
// migration_history table.
func InitMigration(dir string) error {
	name := makeStubName(1, "init")
	forwardSQL := fmt.Sprintf(initForwardTmpl, name)
	backwardSQL := fmt.Sprintf(initBackwardTmpl, name)
	err := writeStubs(dir, name, forwardSQL, backwardSQL)
	if err != nil {
		return err
	}
	return nil
}

// NewMigration creates a new directory containing forward.sql and backward.sql
// stubs.  The directory created will use the name provided to the function,
// prepended by an auto-incrementing zero-padded number.
func NewMigration(dir, name string) error {
	names, err := getMigrationFileNames(dir)
	if err != nil {
		return fmt.Errorf("error making new migration: %v", err)
	}
	latestNum, err := getLatestMigrationFileNumber(names)
	if err != nil {
		return fmt.Errorf("error making new migration: %v", err)
	}
	newName := makeStubName(latestNum+1, name)
	forwardSQL := fmt.Sprintf(forwardTmpl, newName)
	backwardSQL := fmt.Sprintf(backwardTmpl, newName)
	err = writeStubs(dir, newName, forwardSQL, backwardSQL)
	if err != nil {
		return fmt.Errorf("error making new migration: %v", err)
	}
	return nil
}

// ReadMigrationFiles reads all the migration files in the given directory and
// returns an array of Migration objects.
func ReadMigrationFiles(dir string) ([]Migration, error) {
	names, err := getMigrationFileNames(dir)
	if err != nil {
		return nil, err
	}

	migs := []Migration{}
	for _, name := range names {
		m, err := readMigration(dir, name)
		if err != nil {
			return nil, err
		}
		migs = append(migs, m)
	}
	return migs, nil
}

// return a sorted list of subdirs that match our pattern
func getMigrationFileNames(dir string) ([]string, error) {
	names := []string{}
	files, err := ioutil.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("error listing migration files: %v", err)
	}

	for _, file := range files {
		name := file.Name()
		if err != nil {
			return nil, err
		}
		if file.IsDir() && isMigration(name) {
			names = append(names, name)
		}
	}
	return names, nil
}

func getLatestMigrationFileNumber(names []string) (int, error) {
	if len(names) == 0 {
		return 0, nil
	}
	last := names[len(names)-1]
	parts := strings.Split(last, "_")
	num, err := strconv.Atoi(parts[0])
	if err != nil {
		return 0, fmt.Errorf("error getting migration number: %v", err)
	}
	return num, nil
}

func writeStubs(dir, name, forwardSQL, backwardSQL string) error {
	newFolder := path.Join(dir, name)
	err := os.Mkdir(newFolder, 0755)
	if err != nil {
		return fmt.Errorf("error creating migration directory %s: %v", newFolder, err)
	}

	err = ioutil.WriteFile(path.Join(newFolder, "forward.sql"), []byte(forwardSQL), 0644)
	if err != nil {
		return fmt.Errorf("error writing migration file: %v", err)
	}
	err = ioutil.WriteFile(path.Join(newFolder, "backward.sql"), []byte(backwardSQL), 0644)
	if err != nil {
		return fmt.Errorf("error writing migration file: %v", err)
	}
	fmt.Printf("Migration stubs written to %s\n", newFolder)
	return nil
}

func makeStubName(numPart int, namePart string) string {
	return fmt.Sprintf("%s_%s", zeroPad(numPart, leadingDigits), namePart)
}

func readMigration(dir, name string) (Migration, error) {
	m := Migration{Name: name}
	fwd, err := ioutil.ReadFile(path.Join(dir, name, forwardFile))
	if err != nil {
		return m, err
	}

	back, err := ioutil.ReadFile(path.Join(dir, name, backwardFile))
	if err != nil {
		return m, err
	}
	m.ForwardSQL = string(fwd)
	m.BackwardSQL = string(back)
	return m, nil
}

func writeGoMigrations(dir, goFile, packageName string, migs []Migration, generateTag bool) error {
	tmpl, err := template.New("migrations").Parse(srcTmpl)
	if err != nil {
		return nil
	}

	tmplData := srcContext{
		PackageName: packageName,
		Migrations:  migs,
		GenerateTag: generateTag,
		GoFile:      path.Base(goFile),
	}
	var buf bytes.Buffer
	err = tmpl.Execute(&buf, tmplData)
	if err != nil {
		return err
	}
	formatted, err := format.Source(buf.Bytes())
	if err != nil {
		return err
	}
	fname := path.Join(dir, goFile)
	return ioutil.WriteFile(fname, formatted, 0644)
}

func zeroPad(num, digits int) string {
	return fmt.Sprintf("%"+fmt.Sprintf("0%dd", digits), num)
}

func isMigration(dir string) bool {
	pat := fmt.Sprintf(`^[\d]{%d}_.*$`, leadingDigits)
	match, err := regexp.MatchString(pat, dir)
	if err != nil {
		return false
	}
	return match
}
