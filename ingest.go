package pomegranate

import (
	"bytes"
	"fmt"
	"go/format"
	"io/ioutil"
	"path"
	"text/template"
)

type srcContext struct {
	PackageName string
	Migrations  []Migration
}

const srcTmpl = `package {{.PackageName}} 

import "github.com/btubbs/pomegranate"

var All = []pomegranate.Migration{
{{range .Migrations}}  {
  Name: "{{.Name}}",
  ForwardSQL: {{.QuotedForward}},
  BackwardSQL: {{.QuotedBackward}},
  },{{end}}
}
`

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

func readMigrations(dir string) ([]Migration, error) {
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

func writeMigrations(dir, goFile, packageName string, migs []Migration) error {
	tmpl, err := template.New("migrations").Parse(srcTmpl)
	if err != nil {
		return nil
	}

	tmplData := srcContext{
		PackageName: "migrations",
		Migrations:  migs,
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

func IngestMigrations(dir, goFile, packageName string) error {
	migs, err := readMigrations(dir)
	if err != nil {
		fmt.Println(err)
		return err
	}
	return writeMigrations(dir, goFile, packageName, migs)
}
