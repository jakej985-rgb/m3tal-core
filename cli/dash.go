package main

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/jakej985-rgb/m3tal-core/pkg/client"
	"github.com/jakej985-rgb/m3tal-core/pkg/cmdutil"
	"github.com/jakej985-rgb/m3tal-core/pkg/output"
	"github.com/jakej985-rgb/m3tal-core/pkg/system"
	"github.com/spf13/cobra"
)

func initDashCmd() *cobra.Command {
	var dashCmd = &cobra.Command{
		Use:   "dash",
		Short: "Manage the M3TAL Dashboard",
		Long:  `Start, stop, and monitor the M3TAL dashboard container via the API.`,
	}

	var startCmd = &cobra.Command{
		Use:   "start",
		Short: "Start the dashboard service",
		Run: cmdutil.WithDaemon(func(c *client.Client, cmd *cobra.Command, args []string) {
			_, err := c.StartStack("m3tal")
			if err != nil {
				output.FatalError(err)
			}
			fmt.Println("✅ Dashboard started successfully.")
		}),
	}

	var stopCmd = &cobra.Command{
		Use:   "stop",
		Short: "Stop the dashboard service",
		Run: cmdutil.WithDaemon(func(c *client.Client, cmd *cobra.Command, args []string) {
			_, err := c.StopStack("m3tal")
			if err != nil {
				output.FatalError(err)
			}
			fmt.Println("✅ Dashboard stopped successfully.")
		}),
	}

	var restartCmd = &cobra.Command{
		Use:   "restart",
		Short: "Restart the dashboard service",
		Run: cmdutil.WithDaemon(func(c *client.Client, cmd *cobra.Command, args []string) {
			_, err := c.RestartStack("m3tal")
			if err != nil {
				output.FatalError(err)
			}
			fmt.Println("✅ Dashboard restarted successfully.")
		}),
	}

	var logsCmd = &cobra.Command{
		Use:   "logs",
		Short: "View dashboard logs",
		Run: cmdutil.WithDaemon(func(c *client.Client, cmd *cobra.Command, args []string) {
			containers, err := c.GetContainers()
			if err != nil {
				output.FatalError(err)
			}
			found := false
			for _, container := range containers {
				for _, name := range container.Names {
					if strings.Contains(name, "dashboard") || strings.Contains(name, "m3tal-dashboard") {
						logs, err := c.GetLogs(container.ID)
						if err != nil {
							output.FatalError(err)
						}
						fmt.Println(logs)
						found = true
						break
					}
				}
				if found {
					break
				}
			}
			if !found {
				fmt.Println("⚠️  No active dashboard container found.")
			}
		}),
	}

	var statusCmd = &cobra.Command{
		Use:   "status",
		Short: "Check dashboard status",
		Run: cmdutil.WithDaemon(func(c *client.Client, cmd *cobra.Command, args []string) {
			stacks, err := c.GetStacks()
			if err != nil {
				output.FatalError(err)
			}
			for _, s := range stacks {
				if s.Name == "m3tal" {
					fmt.Printf("Dashboard Stack Status: %s\n", s.Status)
					return
				}
			}
			fmt.Println("Dashboard Stack Status: not found")
		}),
	}

	var upCmd = &cobra.Command{
		Use:   "up",
		Short: "Pull config, start API, and start dashboard",
		Run: cmdutil.WithDaemon(func(c *client.Client, cmd *cobra.Command, args []string) {
			pullConfig()

			fmt.Println("🚀 Deploying Dashboard via API...")
			_, err := c.StartStack("m3tal")
			if err != nil {
				output.FatalError(err)
			}
			fmt.Println("✅ Dashboard successfully started!")
		}),
	}

	var pullCmd = &cobra.Command{
		Use:   "pull",
		Short: "Pull dashboard image and config",
		Run: cmdutil.WithClient(func(c *client.Client, cmd *cobra.Command, args []string) {
			pullConfig()
			fmt.Println("✅ Dashboard configuration files downloaded. Images will be pulled automatically when starting.")
		}),
	}

	var pullConfigCmd = &cobra.Command{
		Use:   "pull-config",
		Short: "Download the latest dashboard compose file from GitHub",
		Run: cmdutil.WithClient(func(c *client.Client, cmd *cobra.Command, args []string) {
			pullConfig()
		}),
	}

	dashCmd.AddCommand(upCmd, pullCmd, pullConfigCmd, startCmd, stopCmd, restartCmd, logsCmd, statusCmd)
	return dashCmd
}

func downloadFile(url, targetPath string) error {
	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("bad HTTP status: %s", resp.Status)
	}
	out, err := os.Create(targetPath)
	if err != nil {
		return err
	}
	defer out.Close()
	_, err = io.Copy(out, resp.Body)
	return err
}

func pullConfig() {
	stackDir := system.GetStackDir()

	urls := map[string]string{
		"m3tal-compose.yml":         "https://raw.githubusercontent.com/jakej985-rgb/m3tal-core/main/deploy/stack/m3tal-compose.yml",
		"m3tal-compose.local.yml":   "https://raw.githubusercontent.com/jakej985-rgb/m3tal-core/main/deploy/stack/m3tal-compose.local.yml",
		"m3tal-compose.traefik.yml": "https://raw.githubusercontent.com/jakej985-rgb/m3tal-core/main/deploy/stack/m3tal-compose.traefik.yml",
	}

	fmt.Printf("📥 Pulling latest dashboard config from GitHub...\n")

	// Pre-flight check: Can we write to this directory?
	if info, err := os.Stat(stackDir); err == nil {
		if info.Mode().Perm()&0200 == 0 {
			fmt.Printf("⚠️  Insufficient permissions to write to %s\n", stackDir)
			fmt.Println("👉 Try running: sudo m3tal dash up")
			return
		}
	}

	// Ensure directory exists
	if err := os.MkdirAll(stackDir, 0755); err != nil {
		fmt.Printf("❌ Failed to create stack directory: %v\n", err)
		fmt.Println("👉 Try running with sudo")
		os.Exit(1)
	}

	for filename, url := range urls {
		targetFile := filepath.Join(stackDir, filename)
		// Only download overrides if they don't exist yet, but always update the base m3tal-compose.yml
		if filename != "m3tal-compose.yml" {
			if _, err := os.Stat(targetFile); err == nil {
				fmt.Printf("ℹ️  Preserving local customized %s\n", filename)
				continue
			}
		}
		if err := downloadFile(url, targetFile); err != nil {
			fmt.Printf("⚠️  Failed to download %s: %v\n", filename, err)
			if _, err := os.Stat(targetFile); err == nil {
				output.FatalErrorMsg("No local %s found and download failed.", filename)
			}
		} else {
			fmt.Printf("✅ Saved to %s\n", targetFile)
		}
	}

	ensureUsersFile(stackDir)
}

func ensureUsersFile(stackDir string) {
	usersPath := filepath.Join(stackDir, "users.json")
	if info, err := os.Stat(usersPath); err == nil && info.IsDir() {
		fmt.Printf("⚠️  Found directory named users.json. Removing to recreate as file...\n")
		_ = os.RemoveAll(usersPath)
	}
	if _, err := os.Stat(usersPath); os.IsNotExist(err) {
		defaultUsers := "[\n  {\n    \"username\": \"admin\",\n    \"token_hash\": \"$2b$12$XAGHaPhf67CK3AQF.w26E.fQ5/iS4E0FNHobqhMMYIEdQ2v/1z4l2\",\n    \"role\": \"admin\"\n  }\n]\n"
		if err := os.WriteFile(usersPath, []byte(defaultUsers), 0664); err != nil {
			fmt.Printf("⚠️  Failed to initialize users.json: %v\n", err)
		} else {
			fmt.Println("✅ Initialized default users.json configuration file.")
		}
	}
}
