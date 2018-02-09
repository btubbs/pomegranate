package main

import (
	"bufio"
	"database/sql"
	"fmt"
	"os"
	"strings"
	"text/tabwriter"

	"github.com/btubbs/pomegranate"
	_ "github.com/lib/pq"
	cli "gopkg.in/urfave/cli.v1"
)

func main() {
	app := cli.NewApp()
	app.Name = "pmg"
	app.Usage = "Create and run Pomegranate migrations"
	app.Version = "0.0.1"

	dirFlag := cli.StringFlag{
		Name:  "dir",
		Value: ".",
		Usage: "migrations directory",
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
				name := getArg(c, 0, "migration name")
				dir := c.String("dir")
				err := pomegranate.NewMigration(dir, name)
				if err != nil {
					return cli.NewExitError(err, 1)
				}
				return nil
			},
		},
		{
			Name:  "history",
			Usage: "show the migration history",
			Action: func(c *cli.Context) error {
				//dburl := getArg(c, 0, "database url")
				db, err := sql.Open("postgres", "postgres://postgres@/pmg?sslmode=disable")
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
		{
			Name:  "ingest",
			Usage: "convert .sql migrations to migrations.go file",
			Flags: []cli.Flag{dirFlag},
			Action: func(c *cli.Context) error {
				dir := c.String("dir")
				err := pomegranate.IngestMigrations(dir, "migrations.go", "migrations")
				if err != nil {
					return cli.NewExitError(err, 1)
				}
				return nil
			},
		},
	}
	app.Run(os.Args)
}

// get arg from position specified by idx. If empty, then prompt for it.
func getArg(c *cli.Context, idx int, prompt string) string {
	arg := c.Args().Get(0)
	if arg == "" {
		fmt.Printf("%s: ", prompt)
		reader := bufio.NewReader(os.Stdin)
		arg, _ = reader.ReadString('\n')
		arg = strings.Trim(arg, " \n")
	}
	return arg
}
