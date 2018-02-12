package pomegranate

import (
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestWriteInitMigration(t *testing.T) {
	dir, _ := ioutil.TempDir(".", "pmgtest")
	defer os.RemoveAll(dir)
	err := InitMigration(dir)
	assert.Nil(t, err)
	f, _ := ioutil.ReadFile(path.Join(dir, "00001_init", "forward.sql"))
	assert.Contains(t,
		string(f),
		"INSERT INTO migration_state(name) VALUES ('00001_init');",
	)
	b, _ := ioutil.ReadFile(path.Join(dir, "00001_init", "backward.sql"))
	assert.Contains(t,
		string(b),
		"DELETE FROM migration_state WHERE name='00001_init';",
	)
}

func TestWriteNewMigration(t *testing.T) {
	dir, _ := ioutil.TempDir(".", "pmgtest")
	defer os.RemoveAll(dir)
	name := "foo"
	err := NewMigration(dir, name)
	assert.Nil(t, err)
	f, _ := ioutil.ReadFile(path.Join(dir, "00001_"+name, "forward.sql"))
	assert.Contains(t,
		string(f),
		fmt.Sprintf("INSERT INTO migration_state(name) VALUES ('00001_%s');",
			name),
	)
	b, _ := ioutil.ReadFile(path.Join(dir, "00001_"+name, "backward.sql"))
	assert.Contains(t,
		string(b),
		fmt.Sprintf("DELETE FROM migration_state WHERE name='00001_%s';", name),
	)
}

func TestAutoNumber(t *testing.T) {
	dir, _ := ioutil.TempDir(".", "pmgtest")
	defer os.RemoveAll(dir)
	NewMigration(dir, "foo") // 00001_foo
	name := "bar"
	err := NewMigration(dir, name)
	assert.Nil(t, err)
	f, _ := ioutil.ReadFile(path.Join(dir, "00002_"+name, "forward.sql"))
	assert.Contains(t,
		string(f),
		fmt.Sprintf("INSERT INTO migration_state(name) VALUES ('00002_%s');",
			name),
	)
	b, _ := ioutil.ReadFile(path.Join(dir, "00002_"+name, "backward.sql"))
	assert.Contains(t,
		string(b),
		fmt.Sprintf("DELETE FROM migration_state WHERE name='00002_%s';", name),
	)
}

func TestReadMigrations(t *testing.T) {
	dir, _ := ioutil.TempDir(".", "pmgtest")
	defer os.RemoveAll(dir)
	m1 := path.Join(dir, "00001_foo")
	m2 := path.Join(dir, "00002_bar")
	m3 := path.Join(dir, "other_dir") // should be excluded from results
	os.Mkdir(m1, 0755)
	os.Mkdir(m2, 0755)
	ioutil.WriteFile(path.Join(m1, "forward.sql"), []byte("m1 forward"), 0644)
	ioutil.WriteFile(path.Join(m1, "backward.sql"), []byte("m1 backward"), 0644)
	ioutil.WriteFile(path.Join(m2, "forward.sql"), []byte("m2 forward"), 0644)
	ioutil.WriteFile(path.Join(m2, "backward.sql"), []byte("m2 backward"), 0644)
	ioutil.WriteFile(path.Join(m3, "forward.sql"), []byte("m3 forward"), 0644)
	ioutil.WriteFile(path.Join(m3, "backward.sql"), []byte("m3 backward"), 0644)

	expected := []Migration{
		Migration{
			Name:        "00001_foo",
			ForwardSQL:  "m1 forward",
			BackwardSQL: "m1 backward",
		},
		Migration{
			Name:        "00002_bar",
			ForwardSQL:  "m2 forward",
			BackwardSQL: "m2 backward",
		},
	}
	migs, err := ReadMigrationFiles(dir)
	assert.Nil(t, err)
	assert.Equal(t, expected, migs)
}

func TestIngestMigrations(t *testing.T) {
	dir, _ := ioutil.TempDir(".", "pmgtest")
	defer os.RemoveAll(dir)
	NewMigration(dir, "foo") // 00001_foo
	NewMigration(dir, "bar") // 00002_bar
	err := IngestMigrations(dir, "testmigrations.go", "somepackage", true)
	assert.Nil(t, err)
	f, _ := ioutil.ReadFile(path.Join(dir, "testmigrations.go"))
	contents := string(f)
	assert.Contains(t, contents, "package somepackage")
	assert.Contains(
		t,
		contents,
		"//go:generate pmg ingest -package somepackage -gofile testmigrations.go",
	)

	// also check disabling "go generate" tag
	err = IngestMigrations(dir, "testmigrations.go", "somepackage", false)
	assert.Nil(t, err)
	f, _ = ioutil.ReadFile(path.Join(dir, "testmigrations.go"))
	contents = string(f)
	assert.NotContains(
		t,
		contents,
		"//go:generate",
	)
}
