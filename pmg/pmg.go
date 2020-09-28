package main

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/btubbs/pomegranate"
	"github.com/urfave/cli/v2"
)

func main() {
	app := cli.NewApp()
	app.Name = "pmg"
	app.Usage = "Create and run Postgres migrations"
	app.Version = "0.0.10"

	// dirFlag and dbFlag are declared once up here and used in multiple places
	// below.  Single-use flags will be declared inline.
	dirFlag := &cli.StringFlag{
		Name:  "dir",
		Value: ".",
		Usage: "Migrations directory",
	}
	dbFlag := &cli.StringFlag{
		Name:    "dburl",
		Usage:   "Database URL",
		EnvVars: []string{"DATABASE_URL"},
	}
	timestampFlag := &cli.BoolFlag{
		Name:  "ts",
		Usage: "To use timestamps for the number part of the migration name",
	}

	app.Commands = []*cli.Command{
		{
			Name:  "init",
			Usage: "Create initial migration",
			Flags: []cli.Flag{dirFlag, timestampFlag},
			Action: func(c *cli.Context) error {
				dir := c.String("dir")
				if c.Bool("ts") {
					err := pomegranate.InitMigrationTimestamp(dir, time.Now().UTC())
					if err != nil {
						return cli.NewExitError(err, 1)
					}
				} else {
					err := pomegranate.InitMigration(dir)
					if err != nil {
						return cli.NewExitError(err, 1)
					}
				}
				return nil
			},
		},
		{
			Name:  "new",
			Usage: "Create new (not initial) migration with given name",
			Flags: []cli.Flag{dirFlag, timestampFlag},
			Action: func(c *cli.Context) error {
				name, err := getArg(c, 0, "migration name")
				if err != nil {
					return cli.NewExitError(err, 1)
				}
				if name == "" {
					return cli.NewExitError("empty name not permitted", 1)
				}
				dir := c.String("dir")
				if c.Bool("ts") {
					err = pomegranate.NewMigrationTimestamp(dir, name, time.Now().UTC())
					if err != nil {
						return cli.NewExitError(err, 1)
					}
				} else {
					err = pomegranate.NewMigration(dir, name)
					if err != nil {
						return cli.NewExitError(err, 1)
					}
				}
				return nil
			},
		},
		{
			Name:  "ingest",
			Usage: "Convert .sql migrations to migrations.go file",
			Flags: []cli.Flag{
				dirFlag,
				&cli.StringFlag{
					Name:  "gofile",
					Value: "migrations.go",
					Usage: "filename to be written",
				},
				&cli.StringFlag{
					Name:  "package",
					Value: "migrations",
					Usage: "Go package name for file to be written",
				},
				&cli.BoolFlag{
					Name:  "nogenerate",
					Usage: "Don't include a go:generate tag inside file",
				},
			},
			Action: func(c *cli.Context) error {
				err := pomegranate.IngestMigrations(
					c.String("dir"),
					c.String("gofile"),
					c.String("package"),
					!c.Bool("nogenerate"),
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
			Name:  "fakeforwardto",
			Usage: "Fake migrating forward to specified migration",
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
				err = pomegranate.FakeMigrateForwardTo(migrateTo, db, allMigrations, true)
				if err != nil {
					return cli.NewExitError(err, 1)
				}
				fmt.Println("Done")
				return nil
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
			Name:  "state",
			Usage: "Show the migration state",
			Flags: []cli.Flag{dbFlag},
			Action: func(c *cli.Context) error {
				db, err := pomegranate.Connect(c.String("dburl"))
				if err != nil {
					return cli.NewExitError(err, 1)
				}
				migs, err := pomegranate.GetMigrationState(db)
				if err != nil {
					return cli.NewExitError(err, 1)
				}
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
		{
			Name:  "log",
			Usage: "Show the migration log",
			Flags: []cli.Flag{dbFlag},
			Action: func(c *cli.Context) error {
				db, err := pomegranate.Connect(c.String("dburl"))
				if err != nil {
					return cli.NewExitError(err, 1)
				}
				migs, err := pomegranate.GetMigrationLog(db)
				if err != nil {
					return cli.NewExitError(err, 1)
				}
				w := new(tabwriter.Writer)
				w.Init(os.Stdout, 5, 0, 1, ' ', tabwriter.Debug)
				fmt.Fprintln(w, "ID\t TIME\t NAME\t OP\t WHO")
				for _, m := range migs {
					fmt.Fprintf(w, "%d\t %s\t %s\t %s\t %s\n", m.ID, m.Time, m.Name, m.Op, m.Who)
				}
				w.Flush()
				return nil
			},
		},
	}
	err := app.Run(os.Args)
	if err != nil {
		log.Fatal(err)
	}
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
