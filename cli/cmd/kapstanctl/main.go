package main

import (
	"os"

	"github.com/spf13/cobra"

	"github.com/superset-studio/kapstan/cli/internal"
)

var version = "dev"

func main() {
	root := &cobra.Command{
		Use:     "kapstanctl",
		Short:   "Kapstan CLI — manage your Kapstan platform",
		Version: version,
	}

	root.PersistentFlags().String("server", "http://localhost:9650", "Kapstan API server URL")

	root.AddCommand(internal.AuthCmd())
	root.AddCommand(internal.OrgCmd())
	root.AddCommand(internal.MembersCmd())
	root.AddCommand(internal.APIKeysCmd())

	if err := root.Execute(); err != nil {
		os.Exit(1)
	}
}
