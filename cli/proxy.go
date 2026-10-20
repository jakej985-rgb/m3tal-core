package main

import (
	"fmt"

	"github.com/jakej985-rgb/m3tal-core/pkg/client"
	"github.com/jakej985-rgb/m3tal-core/pkg/cmdutil"
	"github.com/jakej985-rgb/m3tal-core/pkg/models"
	"github.com/jakej985-rgb/m3tal-core/pkg/output"
	"github.com/spf13/cobra"
)

var (
	exposeDomain      string
	exposePort        int
	exposeSSL         bool
	exposeMiddlewares []string
)

func initProxyCmds() *cobra.Command {
	var proxyCmd = &cobra.Command{
		Use:   "proxy",
		Short: "Manage reverse proxy routes, auto-discovery, and SSL",
		Run: func(cmd *cobra.Command, args []string) {
			cmd.Help()
		},
	}

	var proxyStatusCmd = &cobra.Command{
		Use:   "status",
		Short: "List all active reverse proxy routes",
		Run: cmdutil.WithDaemon(func(c *client.Client, cmd *cobra.Command, args []string) {
			var list []models.RouteRecord
			err := c.Request("GET", "/api/v2/routes", nil, &list)
			if err != nil {
				output.FatalError(err)
			}
			output.PrintRoutesTable(list)
		}),
	}

	var proxyDiscoverCmd = &cobra.Command{
		Use:   "discover",
		Short: "Auto-discover proxy candidate containers running on the host",
		Run: cmdutil.WithDaemon(func(c *client.Client, cmd *cobra.Command, args []string) {
			var list []models.DiscoverableService
			err := c.Request("GET", "/api/v2/proxy/discover", nil, &list)
			if err != nil {
				output.FatalError(err)
			}
			output.PrintDiscoverTable(list)
		}),
	}

	var proxyExposeCmd = &cobra.Command{
		Use:   "expose [container-name]",
		Short: "Expose a container service via Traefik routing rule",
		Args:  cobra.ExactArgs(1),
		Run: cmdutil.WithDaemon(func(c *client.Client, cmd *cobra.Command, args []string) {
			service := args[0]

			if exposeDomain == "" {
				output.FatalErrorMsg("Domain must be specified via --domain")
			}
			if exposePort == 0 {
				output.FatalErrorMsg("Internal port must be specified via --port")
			}

			body := map[string]interface{}{
				"service":     service,
				"domain":      exposeDomain,
				"port":        exposePort,
				"ssl":         exposeSSL,
				"middlewares": exposeMiddlewares,
			}
			if err := c.Request("POST", "/api/v2/proxy/expose", body, nil); err != nil {
				output.FatalError(err)
			}

			scheme := "http"
			if exposeSSL {
				scheme = "https"
			}
			fmt.Printf("✅ Exposed %s on %s://%s (port %d, via API)\n", service, scheme, exposeDomain, exposePort)
		}),
	}

	proxyExposeCmd.Flags().StringVar(&exposeDomain, "domain", "", "Public/internal domain name (e.g. app.local)")
	proxyExposeCmd.Flags().IntVar(&exposePort, "port", 0, "Internal container port to forward to")
	proxyExposeCmd.Flags().BoolVar(&exposeSSL, "ssl", false, "Enable Let's Encrypt SSL/TLS automation for this route")
	proxyExposeCmd.Flags().StringSliceVar(&exposeMiddlewares, "middlewares", []string{}, "Comma-separated list of middlewares to attach")

	var proxyUnexposeCmd = &cobra.Command{
		Use:   "unexpose [domain-name]",
		Short: "Remove a routing rule and close the proxy door",
		Args:  cobra.ExactArgs(1),
		Run: cmdutil.WithDaemon(func(c *client.Client, cmd *cobra.Command, args []string) {
			domain := args[0]
			body := map[string]interface{}{
				"domain": domain,
			}
			if err := c.Request("POST", "/api/v2/proxy/unexpose", body, nil); err != nil {
				output.FatalError(err)
			}
			fmt.Printf("✅ Unexposed domain %s (via API)\n", domain)
		}),
	}

	var proxySecureCmd = &cobra.Command{
		Use:   "secure [email-address]",
		Short: "Configure automated Let's Encrypt TLS generation & expose ports 80/443 on the host",
		Args:  cobra.ExactArgs(1),
		Run: cmdutil.WithDaemon(func(c *client.Client, cmd *cobra.Command, args []string) {
			email := args[0]
			body := map[string]interface{}{
				"email": email,
			}
			if err := c.Request("POST", "/api/v2/proxy/secure", body, nil); err != nil {
				output.FatalError(err)
			}
			fmt.Println("✅ Let's Encrypt SSL configured and routing stack redeployed (via API)")
		}),
	}

	proxyCmd.AddCommand(proxyStatusCmd, proxyDiscoverCmd, proxyExposeCmd, proxyUnexposeCmd, proxySecureCmd)
	return proxyCmd
}
