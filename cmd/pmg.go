package main

import (
	"bufio"
	"fmt"
	"os"
	"strings"
	"text/tabwriter"

	"github.com/btubbs/pomegranate"
	cli "gopkg.in/urfave/cli.v1"
)

func main() {
	app := cli.NewApp()
	app.Name = "pmg"
	app.Usage = "Create and run Postgres migrations"
	app.Version = "0.0.1"

	// dirFlag and dbFlag are declared once up here and used in multiple places
	// below.  Single-use flags will be declared inline.
	dirFlag := cli.StringFlag{
		Name:  "dir",
		Value: ".",
		Usage: "migrations directory",
	}
	dbFlag := cli.StringFlag{
		Name:   "dburl",
		Usage:  "database url",
		EnvVar: "DATABASE_URL",
	}

	app.Commands = []cli.Command{
		{
			Name:  "init",
			Usage: "create initial migration",
			Flags: []cli.Flag{dirFlag},
			Action: func(c *cli.Context) error {
				dir := c.String("dir")
				err := pomegranate.InitMigration(dir)
				if err != nil {
					return cli.NewExitError(err, 1)
				}
				return nil
			},
		},
		{
			Name:  "new",
			Usage: "create new (not initial) migration with given name",
			Flags: []cli.Flag{dirFlag},
			Action: func(c *cli.Context) error {
				name, err := getArg(c, 0, "migration name")
				if err != nil {
					return cli.NewExitError(err, 1)
				}
				if name == "" {
					return cli.NewExitError("empty name not permitted", 1)
				}
				dir := c.String("dir")
				err = pomegranate.NewMigration(dir, name)
				if err != nil {
					return cli.NewExitError(err, 1)
				}
				return nil
			},
		},
		{
			Name:  "ingest",
			Usage: "convert .sql migrations to migrations.go file",
			Flags: []cli.Flag{
				dirFlag,
				cli.StringFlag{
					Name:  "gofile",
					Value: "migrations.go",
					Usage: "filename to be written",
				},
				cli.StringFlag{
					Name:  "package",
					Value: "migrations",
					Usage: "Go package name for file to be written",
				},
				cli.BoolTFlag{
					Name:  "gogenerate",
					Usage: "Whether to include a go:generate tag inside file",
				},
			},
			Action: func(c *cli.Context) error {
				err := pomegranate.IngestMigrations(
					c.String("dir"),
					c.String("gofile"),
					c.String("package"),
					c.Bool("gogenerate"),
				)
				if err != nil {
					return cli.NewExitError(err, 1)
				}
				return nil
			},
		},
		{
			Name:  "forward",
			Usage: "Migrate forward to latest migration",
			Flags: []cli.Flag{dirFlag, dbFlag},
			Action: func(c *cli.Context) error {
				return forward(c, "")
			},
		},
		{
			Name:  "forwardto",
			Usage: "Migrate forward to specified migration",
			Flags: []cli.Flag{dirFlag, dbFlag},
			Action: func(c *cli.Context) error {
				migrateTo, err := getArg(c, 0, "migration name")
				if err != nil {
					return cli.NewExitError(err, 1)
				}
				return forward(c, migrateTo)
			},
		},
		{
			Name:  "backwardto",
			Usage: "Migrate backward to specified migration",
			Flags: []cli.Flag{dirFlag, dbFlag},
			Action: func(c *cli.Context) error {
				migrateTo, err := getArg(c, 0, "migration name")
				if err != nil {
					return cli.NewExitError(err, 1)
				}
				db, err := pomegranate.Connect(c.String("dburl"))
				if err != nil {
					return cli.NewExitError(err, 1)
				}
				dir := c.String("dir")
				allMigrations, err := pomegranate.ReadMigrationFiles(dir)
				if err != nil {
					return cli.NewExitError(err, 1)
				}
				err = pomegranate.MigrateBackwardTo(migrateTo, db, allMigrations, true)
				if err != nil {
					return cli.NewExitError(err, 1)
				}
				fmt.Println("Done")
				return nil
			},
		},
		{
			Name:  "history",
			Usage: "show the migration history",
			Flags: []cli.Flag{dbFlag},
			Action: func(c *cli.Context) error {
				db, err := pomegranate.Connect(c.String("dburl"))
				if err != nil {
					return cli.NewExitError(err, 1)
				}
				migs, err := pomegranate.GetMigrationHistory(db)
				w := new(tabwriter.Writer)
				w.Init(os.Stdout, 5, 0, 1, ' ', tabwriter.Debug)
				fmt.Fprintln(w, "NAME\t WHEN\t WHO")
				for _, m := range migs {
					fmt.Fprintf(w, "%s\t %s\t %s\n", m.Name, m.Time, m.Who)
				}
				w.Flush()
				return nil
			},
		},
	}
	app.Run(os.Args)
}

// forward takes the cli context, a migration name to migrate to, and makes it
// happen.  It's used by both the `forward` and `forwardto` commands.
func forward(c *cli.Context, name string) error {
	db, err := pomegranate.Connect(c.String("dburl"))
	if err != nil {
		return cli.NewExitError(err, 1)
	}
	dir := c.String("dir")
	allMigrations, err := pomegranate.ReadMigrationFiles(dir)
	if err != nil {
		return cli.NewExitError(err, 1)
	}
	err = pomegranate.MigrateForwardTo(name, db, allMigrations, true)
	if err != nil {
		return cli.NewExitError(err, 1)
	}
	fmt.Println("Done")
	return nil
}

// get arg from position specified by idx. If empty, then prompt for it.
func getArg(c *cli.Context, idx int, prompt string) (string, error) {
	arg := c.Args().Get(0)
	if arg != "" {
		return arg, nil
	}
	fmt.Printf("%s: ", prompt)
	reader := bufio.NewReader(os.Stdin)
	arg, err := reader.ReadString('\n')
	if err != nil {
		return arg, err
	}
	arg = strings.TrimSpace(arg)
	return arg, nil
}
