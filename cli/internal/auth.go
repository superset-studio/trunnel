package internal

import (
	"fmt"
	"os"
	"syscall"

	"github.com/spf13/cobra"
	"golang.org/x/term"
)

// authResponse mirrors the API's auth register/login response.
type authResponse struct {
	AccessToken  string   `json:"accessToken"`
	RefreshToken string   `json:"refreshToken"`
	User         authUser `json:"user"`
	Organization authOrg  `json:"organization"`
}

type authUser struct {
	ID    string `json:"id"`
	Email string `json:"email"`
	Name  string `json:"name"`
}

type authOrg struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	DisplayName string `json:"displayName"`
}

// AuthCmd returns the `auth` parent command with register/login/logout/status subcommands.
func AuthCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "auth",
		Short: "Authentication commands",
	}
	cmd.AddCommand(authRegisterCmd())
	cmd.AddCommand(authLoginCmd())
	cmd.AddCommand(authLogoutCmd())
	cmd.AddCommand(authStatusCmd())
	return cmd
}

func authRegisterCmd() *cobra.Command {
	var email, password, name, orgName string

	cmd := &cobra.Command{
		Use:   "register",
		Short: "Register a new account",
		RunE: func(cmd *cobra.Command, args []string) error {
			if password == "" {
				fmt.Print("Password: ")
				pw, err := term.ReadPassword(int(syscall.Stdin))
				fmt.Println()
				if err != nil {
					return fmt.Errorf("reading password: %w", err)
				}
				password = string(pw)
			}

			serverURL, _ := cmd.Flags().GetString("server")
			client := NewAPIClient(serverURL, "")

			body := map[string]string{
				"email":    email,
				"password": password,
				"name":     name,
				"orgName":  orgName,
			}

			var resp authResponse
			if err := client.doJSON("POST", "/api/v1/auth/register", body, &resp); err != nil {
				return err
			}

			cfg := &CLIConfig{
				ServerURL:    serverURL,
				AccessToken:  resp.AccessToken,
				RefreshToken: resp.RefreshToken,
				ActiveOrgID:  resp.Organization.ID,
				UserEmail:    resp.User.Email,
				UserName:     resp.User.Name,
			}
			if err := SaveConfig(cfg); err != nil {
				return fmt.Errorf("saving config: %w", err)
			}

			fmt.Printf("Registered as %s (org: %s)\n", resp.User.Email, resp.Organization.DisplayName)
			return nil
		},
	}

	cmd.Flags().StringVar(&email, "email", "", "email address (required)")
	cmd.Flags().StringVar(&password, "password", "", "password (prompted if omitted)")
	cmd.Flags().StringVar(&name, "name", "", "display name (required)")
	cmd.Flags().StringVar(&orgName, "org-name", "", "organization name (required)")
	_ = cmd.MarkFlagRequired("email")
	_ = cmd.MarkFlagRequired("name")
	_ = cmd.MarkFlagRequired("org-name")

	return cmd
}

func authLoginCmd() *cobra.Command {
	var email, password string

	cmd := &cobra.Command{
		Use:   "login",
		Short: "Log in to an existing account",
		RunE: func(cmd *cobra.Command, args []string) error {
			if password == "" {
				fmt.Print("Password: ")
				pw, err := term.ReadPassword(int(syscall.Stdin))
				fmt.Println()
				if err != nil {
					return fmt.Errorf("reading password: %w", err)
				}
				password = string(pw)
			}

			serverURL, _ := cmd.Flags().GetString("server")
			client := NewAPIClient(serverURL, "")

			body := map[string]string{
				"email":    email,
				"password": password,
			}

			var resp authResponse
			if err := client.doJSON("POST", "/api/v1/auth/login", body, &resp); err != nil {
				return err
			}

			cfg := &CLIConfig{
				ServerURL:    serverURL,
				AccessToken:  resp.AccessToken,
				RefreshToken: resp.RefreshToken,
				ActiveOrgID:  resp.Organization.ID,
				UserEmail:    resp.User.Email,
				UserName:     resp.User.Name,
			}
			if err := SaveConfig(cfg); err != nil {
				return fmt.Errorf("saving config: %w", err)
			}

			fmt.Printf("Logged in as %s (org: %s)\n", resp.User.Email, resp.Organization.DisplayName)
			return nil
		},
	}

	cmd.Flags().StringVar(&email, "email", "", "email address (required)")
	cmd.Flags().StringVar(&password, "password", "", "password (prompted if omitted)")
	_ = cmd.MarkFlagRequired("email")

	return cmd
}

func authLogoutCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "logout",
		Short: "Log out and clear saved credentials",
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := ClearConfig(); err != nil {
				return err
			}
			fmt.Println("Logged out")
			return nil
		},
	}
}

func authStatusCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "status",
		Short: "Show current authentication status",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := LoadConfig()
			if err != nil {
				return err
			}

			if cfg.AccessToken == "" {
				fmt.Fprintln(os.Stderr, "Not logged in. Run: kapstanctl auth login")
				return nil
			}

			fmt.Printf("Logged in as %s", cfg.UserEmail)
			if cfg.UserName != "" {
				fmt.Printf(" (%s)", cfg.UserName)
			}
			fmt.Println()
			fmt.Printf("  Org:    %s\n", cfg.ActiveOrgID)
			fmt.Printf("  Server: %s\n", cfg.ServerURL)
			return nil
		},
	}
}
