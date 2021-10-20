package main

import (
	"fmt"
	"os"

	"github.com/alecthomas/kong"
	"github.com/goresed/goresed/gsed"
	"golang.org/x/tools/imports"
)

func main() {
	cmd := kong.Parse(&CLI)

	switch cmd.Command() {
	case "regenerate":
		err := sed(CLI.Regenerate.File, CLI.Regenerate.References)
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
	Regenerate struct {
		File       string   `default:"goresed.yaml" help:"Specify an alternate YAML file."`
		References []string `help:"Specify YAML references."`
	} `cmd:"" help:"Replace generated code."`
}

func sed(pth string, refs []string) error {
	yml, err := os.Open(pth)
	if err != nil {
		return err
	}

	var opts []gsed.Option

	for _, pth := range refs {
		yml, err := os.Open(pth)
		if err != nil {
			return err
		}

		opts = append(opts, gsed.WithReferences(yml))
	}

	dir, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("os: get current/working directory: %w", err)
	}

	opts = append(opts, gsed.WithDirectory(dir))

	gofmt := imports.Options{
		Fragment:  true,
		Comments:  true,
		TabIndent: true,
		TabWidth:  8,
	}

	opts = append(opts, gsed.WithGofmt(&gofmt))

	err = gsed.New(yml, opts...)
	if err != nil {
		fmt.Fprint(os.Stderr, err)
		os.Exit(1)
	}

	return nil
}
