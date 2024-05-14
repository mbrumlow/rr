package main

import (
	"fmt"
	"os"
	"rr/pkg/run"
	"rr/pkg/tui"

	"github.com/urfave/cli/v2"
)

func main() {
	app := &cli.App{
		Name:        "rr",
		Usage:       "A Swiss army knife for remote commands",
		Description: "A repeatable remote command runner",
		Commands: []*cli.Command{
			{
				Name:    "run",
				Aliases: []string{"r"},
				Flags: []cli.Flag{
					&cli.BoolFlag{
						Name: "direct",
					},
				},
				ArgsUsage: " file",
				Action: func(c *cli.Context) error {
					count := c.Args().Len()
					if count == 0 {
						return nil
					} else if count > 1 {
						return fmt.Errorf("only one run file can be specified at a time")
					}

					follower := &tui.Simple{}
					return run.File(c.Args().First(), follower)

				},
			},
			{
				Name:    "server",
				Aliases: []string{"s"},
				Action: func(c *cli.Context) error {
					return run.Serve()
				},
			},
		},
	}
	if err := app.Run(os.Args); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(-1)
	}
}
