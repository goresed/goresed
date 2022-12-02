package main

import (
	"errors"
	"fmt"
	"os"

	"github.com/alecthomas/kong"
	"github.com/goresed/goresed/gsed"
	"golang.org/x/tools/imports"
)

func main() {
	cmd := kong.Parse(&CLI)

	switch cmd.Command() {
	case "replace":
		if err := sed(CLI.Replace.Configurations); err != nil {
			fmt.Fprintf(os.Stderr, "%s\n", err)
			os.Exit(1)
		}

	case "go":
		err := sed(CLI.Go.Configurations,
			gsed.WithGofmt(&imports.Options{
				Fragment:  true,
				Comments:  true,
				TabIndent: true,
				TabWidth:  8,
			}),
		)
		if err != nil {
			fmt.Fprintf(os.Stderr, "%s\n", err)
			os.Exit(1)
		}

	default:
		fmt.Fprint(os.Stderr, cmd.Command())
		os.Exit(1)
	}
}

var CLI struct {
	Replace SearchAndReplace `cmd:"" help:"Search and replace."`
	Go      SearchAndReplace `cmd:"" help:"Search and replace in Go source code."`
}

type SearchAndReplace struct {
	Configurations []string `name:"config" help:"Path to configuration."`
}

func sed(configs []string, opts ...gsed.Option) error {
	if len(configs) == 0 {
		return errors.New("missing config")
	}

	if configs[0] == "" {
		return errors.New("empty config")
	}

	yml, err := os.Open(configs[0])
	if err != nil {
		return err
	}

	if len(configs) > 1 {
		for _, pth := range configs[1:] {
			yml, err := os.Open(pth)
			if err != nil {
				return err
			}

			opts = append(opts, gsed.WithYAMLReferences(yml))
		}
	}

	dir, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("get current/working os directory: %w", err)
	}

	opts = append(opts, gsed.WithDirectory(dir))

	err = gsed.New(yml, opts...)
	if err != nil {
		fmt.Fprint(os.Stderr, err)
		os.Exit(1)
	}

	return nil
}
