package internal

import (
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/spf13/cobra"
)

type memberResponse struct {
	ID    string `json:"id"`
	Role  string `json:"role"`
	Email string `json:"email"`
	Name  string `json:"name"`
}

// MembersCmd returns the `members` parent command with list/invite/update-role/remove subcommands.
func MembersCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "members",
		Short: "Organization member commands",
	}
	cmd.PersistentFlags().String("org", "", "organization ID (defaults to active org)")
	cmd.AddCommand(membersListCmd())
	cmd.AddCommand(membersInviteCmd())
	cmd.AddCommand(membersUpdateRoleCmd())
	cmd.AddCommand(membersRemoveCmd())
	return cmd
}

func membersListCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List organization members",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, client, err := clientFromConfig(cmd)
			if err != nil {
				return err
			}
			orgID, err := resolveOrgID(cmd, cfg)
			if err != nil {
				return err
			}

			var members []memberResponse
			path := fmt.Sprintf("/api/v1/organizations/%s/members", orgID)
			if err := client.doJSON("GET", path, nil, &members); err != nil {
				return err
			}

			w := tabwriter.NewWriter(os.Stdout, 0, 4, 2, ' ', 0)
			fmt.Fprintln(w, "ID\tNAME\tEMAIL\tROLE")
			for _, m := range members {
				fmt.Fprintf(w, "%s\t%s\t%s\t%s\n", m.ID, m.Name, m.Email, m.Role)
			}
			return w.Flush()
		},
	}
}

func membersInviteCmd() *cobra.Command {
	var email, role string

	cmd := &cobra.Command{
		Use:   "invite",
		Short: "Invite a member to the organization",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, client, err := clientFromConfig(cmd)
			if err != nil {
				return err
			}
			orgID, err := resolveOrgID(cmd, cfg)
			if err != nil {
				return err
			}

			body := map[string]string{"email": email, "role": role}
			path := fmt.Sprintf("/api/v1/organizations/%s/members/invite", orgID)

			var resp memberResponse
			if err := client.doJSON("POST", path, body, &resp); err != nil {
				return err
			}

			fmt.Printf("Invited %s as %s\n", email, role)
			return nil
		},
	}

	cmd.Flags().StringVar(&email, "email", "", "email to invite (required)")
	cmd.Flags().StringVar(&role, "role", "", "role: owner, admin, member, or viewer (required)")
	_ = cmd.MarkFlagRequired("email")
	_ = cmd.MarkFlagRequired("role")

	return cmd
}

func membersUpdateRoleCmd() *cobra.Command {
	var role string

	cmd := &cobra.Command{
		Use:   "update-role <member-id>",
		Short: "Update a member's role",
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

			body := map[string]string{"role": role}
			path := fmt.Sprintf("/api/v1/organizations/%s/members/%s", orgID, args[0])
			if err := client.doNoContent("PUT", path, body); err != nil {
				return err
			}

			fmt.Printf("Updated member %s to role %s\n", args[0], role)
			return nil
		},
	}

	cmd.Flags().StringVar(&role, "role", "", "new role: owner, admin, member, or viewer (required)")
	_ = cmd.MarkFlagRequired("role")

	return cmd
}

func membersRemoveCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "remove <member-id>",
		Short: "Remove a member from the organization",
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

			path := fmt.Sprintf("/api/v1/organizations/%s/members/%s", orgID, args[0])
			if err := client.doNoContent("DELETE", path, nil); err != nil {
				return err
			}

			fmt.Printf("Removed member %s\n", args[0])
			return nil
		},
	}
}
