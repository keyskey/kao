package main

import (
	"fmt"
	"os"

	"github.com/keyskey/kao/internal/collect"
	"github.com/keyskey/kao/internal/evaluate"
	"github.com/keyskey/kao/internal/run"
	"github.com/spf13/cobra"
)

func main() {
	root := &cobra.Command{
		Use:   "kao",
		Short: "KAO (花押) — change management control evaluation CLI",
	}

	root.AddCommand(collect.NewCommand())
	root.AddCommand(evaluate.NewCommand())
	root.AddCommand(run.NewCommand())

	if err := root.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
