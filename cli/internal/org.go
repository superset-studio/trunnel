package internal

import (
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/spf13/cobra"
)

type orgResponse struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	DisplayName string `json:"displayName"`
}

// OrgCmd returns the `org` parent command with list/create/switch subcommands.
func OrgCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "org",
		Short: "Organization commands",
	}
	cmd.AddCommand(orgListCmd())
	cmd.AddCommand(orgCreateCmd())
	cmd.AddCommand(orgSwitchCmd())
	return cmd
}

func orgListCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List organizations you belong to",
		RunE: func(cmd *cobra.Command, args []string) error {
			_, client, err := clientFromConfig(cmd)
			if err != nil {
				return err
			}

			var orgs []orgResponse
			if err := client.doJSON("GET", "/api/v1/organizations", nil, &orgs); err != nil {
				return err
			}

			w := tabwriter.NewWriter(os.Stdout, 0, 4, 2, ' ', 0)
			fmt.Fprintln(w, "ID\tNAME\tDISPLAY NAME")
			for _, o := range orgs {
				fmt.Fprintf(w, "%s\t%s\t%s\n", o.ID, o.Name, o.DisplayName)
			}
			return w.Flush()
		},
	}
}

func orgCreateCmd() *cobra.Command {
	var displayName string

	cmd := &cobra.Command{
		Use:   "create",
		Short: "Create a new organization",
		RunE: func(cmd *cobra.Command, args []string) error {
			_, client, err := clientFromConfig(cmd)
			if err != nil {
				return err
			}

			body := map[string]string{"displayName": displayName}
			var org orgResponse
			if err := client.doJSON("POST", "/api/v1/organizations", body, &org); err != nil {
				return err
			}

			fmt.Printf("Created organization %s (%s)\n", org.DisplayName, org.ID)
			return nil
		},
	}

	cmd.Flags().StringVar(&displayName, "name", "", "display name for the organization (required)")
	_ = cmd.MarkFlagRequired("name")

	return cmd
}

func orgSwitchCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "switch <org-id>",
		Short: "Set the active organization",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := LoadConfig()
			if err != nil {
				return err
			}
			cfg.ActiveOrgID = args[0]
			if err := SaveConfig(cfg); err != nil {
				return err
			}
			fmt.Printf("Switched to organization %s\n", args[0])
			return nil
		},
	}
}

// clientFromConfig loads the saved config and returns an authenticated API client.
func clientFromConfig(cmd *cobra.Command) (*CLIConfig, *APIClient, error) {
	cfg, err := LoadConfig()
	if err != nil {
		return nil, nil, err
	}
	if cfg.AccessToken == "" {
		return nil, nil, fmt.Errorf("not logged in — run: kapstanctl auth login")
	}

	serverURL, _ := cmd.Flags().GetString("server")
	if serverURL != "" {
		cfg.ServerURL = serverURL
	}
	if cfg.ServerURL == "" {
		cfg.ServerURL = "http://localhost:8080"
	}

	return cfg, NewAPIClient(cfg.ServerURL, cfg.AccessToken), nil
}

// resolveOrgID returns the active org ID from the --org flag or saved config.
func resolveOrgID(cmd *cobra.Command, cfg *CLIConfig) (string, error) {
	orgID, _ := cmd.Flags().GetString("org")
	if orgID != "" {
		return orgID, nil
	}
	if cfg.ActiveOrgID != "" {
		return cfg.ActiveOrgID, nil
	}
	return "", fmt.Errorf("no active organization — use --org or run: kapstanctl org switch <org-id>")
}
