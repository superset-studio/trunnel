package internal

import (
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/spf13/cobra"
)

type apiKeyResponse struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	KeyPrefix   string `json:"keyPrefix"`
	AccessLevel string `json:"accessLevel"`
	CreatedAt   string `json:"createdAt"`
	LastUsedAt  string `json:"lastUsedAt"`
}

type createAPIKeyResponse struct {
	APIKey apiKeyResponse `json:"apiKey"`
	Key    string         `json:"key"`
}

// APIKeysCmd returns the `apikeys` parent command with create/list/revoke subcommands.
func APIKeysCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "apikeys",
		Short: "API key commands",
	}
	cmd.PersistentFlags().String("org", "", "organization ID (defaults to active org)")
	cmd.AddCommand(apikeysCreateCmd())
	cmd.AddCommand(apikeysListCmd())
	cmd.AddCommand(apikeysRevokeCmd())
	return cmd
}

func apikeysCreateCmd() *cobra.Command {
	var name, accessLevel string

	cmd := &cobra.Command{
		Use:   "create",
		Short: "Create a new API key",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, client, err := clientFromConfig(cmd)
			if err != nil {
				return err
			}
			orgID, err := resolveOrgID(cmd, cfg)
			if err != nil {
				return err
			}

			body := map[string]string{"name": name, "accessLevel": accessLevel}
			path := fmt.Sprintf("/api/v1/organizations/%s/api-keys", orgID)

			var resp createAPIKeyResponse
			if err := client.doJSON("POST", path, body, &resp); err != nil {
				return err
			}

			fmt.Printf("API Key: %s\n", resp.Key)
			fmt.Println("Save this key — it won't be shown again.")
			return nil
		},
	}

	cmd.Flags().StringVar(&name, "name", "", "key name (required)")
	cmd.Flags().StringVar(&accessLevel, "access-level", "", "access level (required)")
	_ = cmd.MarkFlagRequired("name")
	_ = cmd.MarkFlagRequired("access-level")

	return cmd
}

func apikeysListCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List API keys",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, client, err := clientFromConfig(cmd)
			if err != nil {
				return err
			}
			orgID, err := resolveOrgID(cmd, cfg)
			if err != nil {
				return err
			}

			var keys []apiKeyResponse
			path := fmt.Sprintf("/api/v1/organizations/%s/api-keys", orgID)
			if err := client.doJSON("GET", path, nil, &keys); err != nil {
				return err
			}

			w := tabwriter.NewWriter(os.Stdout, 0, 4, 2, ' ', 0)
			fmt.Fprintln(w, "ID\tNAME\tPREFIX\tACCESS LEVEL\tCREATED")
			for _, k := range keys {
				fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\n", k.ID, k.Name, k.KeyPrefix, k.AccessLevel, k.CreatedAt)
			}
			return w.Flush()
		},
	}
}

func apikeysRevokeCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "revoke <key-id>",
		Short: "Revoke an API key",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, client, err := clientFromConfig(cmd)
			if err != nil {
				return err
			}
			orgID, err := resolveOrgID(cmd, cfg)
			if err != nil {
				return err
			}

			path := fmt.Sprintf("/api/v1/organizations/%s/api-keys/%s", orgID, args[0])
			if err := client.doNoContent("DELETE", path, nil); err != nil {
				return err
			}

			fmt.Printf("Revoked API key %s\n", args[0])
			return nil
		},
	}
}
