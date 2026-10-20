package cmdutil

import (
	"time"

	"github.com/jakej985-rgb/m3tal-core/pkg/client"
	"github.com/jakej985-rgb/m3tal-core/pkg/config"
	"github.com/jakej985-rgb/m3tal-core/pkg/output"
	"github.com/jakej985-rgb/m3tal-core/pkg/system"
	"github.com/spf13/cobra"
)

// WithClient initializes the client from config and flags, then executes the command.
func WithClient(f func(c *client.Client, cmd *cobra.Command, args []string)) func(cmd *cobra.Command, args []string) {
	return func(cmd *cobra.Command, args []string) {
		c := getClient(cmd)
		f(c, cmd, args)
	}
}

// WithDaemon initializes the client, ensures the API daemon is running, then executes the command.
func WithDaemon(f func(c *client.Client, cmd *cobra.Command, args []string)) func(cmd *cobra.Command, args []string) {
	return func(cmd *cobra.Command, args []string) {
		c := getClient(cmd)
		// Ensure daemon is running
		err := system.EnsureAPIRunning(c.BaseURL, c.Token, 5*time.Second)
		if err != nil {
			output.FatalError(err)
		}
		f(c, cmd, args)
	}
}

func getClient(cmd *cobra.Command) *client.Client {
	url := config.GetAPIURL()
	token := config.GetAPIToken()

	// Apply CLI flag overrides if present
	if cmd != nil {
		if u, err := cmd.Flags().GetString("api-url"); err == nil && u != "" {
			url = u
		} else if u, err := cmd.PersistentFlags().GetString("api-url"); err == nil && u != "" {
			url = u
		}
		if t, err := cmd.Flags().GetString("api-token"); err == nil && t != "" {
			token = t
		} else if t, err := cmd.PersistentFlags().GetString("api-token"); err == nil && t != "" {
			token = t
		}
	}

	return client.NewClient(url, token)
}
