package main

import (
	"bufio"
	"database/sql"
	"fmt"
	"os"
	"strings"

	"github.com/btubbs/pomegranate"
	_ "github.com/lib/pq"
	cli "gopkg.in/urfave/cli.v1"
)

func main() {
	app := cli.NewApp()
	app.Name = "pmg"
	app.Usage = "Create and run Pomegranate migrations"
	app.Version = "0.0.1"
	app.Commands = []cli.Command{
		{
			Name:  "init",
			Usage: "create initial migration",
			Flags: []cli.Flag{
				cli.StringFlag{
					Name:  "dir",
					Value: ".",
					Usage: "migrations directory",
				},
			},
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
			Flags: []cli.Flag{
				cli.StringFlag{
					Name:  "dir",
					Value: ".",
					Usage: "migrations directory",
				},
			},
			Action: func(c *cli.Context) error {
				name := c.Args().Get(0)
				if name == "" {
					fmt.Print("migration name: ")
					reader := bufio.NewReader(os.Stdin)
					name, _ = reader.ReadString('\n')
					name = strings.Trim(name, " \n")
				}
				dir := c.String("dir")
				err := pomegranate.NewMigration(dir, name)
				if err != nil {
					return cli.NewExitError(err, 1)
				}
				return nil
			},
		},
		{
			Name: "foo",
			Action: func(c *cli.Context) error {
				db, err := sql.Open("postgres", "postgres://postgres@/pmg?sslmode=disable")
				if err != nil {
					return cli.NewExitError(err, 1)
				}
				migs, err := pomegranate.GetMigrationHistory(db)
				fmt.Println(migs)
				return nil
			},
		},
	}
	app.Run(os.Args)
}
