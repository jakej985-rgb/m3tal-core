package main

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/jakej985-rgb/m3tal-core/pkg/client"
	"github.com/jakej985-rgb/m3tal-core/pkg/cmdutil"
	"github.com/jakej985-rgb/m3tal-core/pkg/models"
	"github.com/jakej985-rgb/m3tal-core/pkg/output"
	"github.com/jakej985-rgb/m3tal-core/pkg/system"
	"github.com/spf13/cobra"
)

func initFixCmd() *cobra.Command {
	var fixCmd = &cobra.Command{
		Use:   "fix",
		Short: "Apply automated fixes for detected issues",
	}

	var brokenCmd = &cobra.Command{
		Use:   "broken",
		Short: "Manage broken components",
	}

	var stackCmd = &cobra.Command{
		Use:   "stack",
		Short: "Detect and repair issues in stacks",
		Run: cmdutil.WithDaemon(func(c *client.Client, cmd *cobra.Command, args []string) {
			fmt.Println("🏥 Running system diagnostics and applying fixes...")
			res, err := c.HandleDoctorFix(true, "")
			if err != nil {
				output.FatalError(err)
			}

			resultsData, _ := json.Marshal(res["results"])
			var results []models.FixResult
			_ = json.Unmarshal(resultsData, &results)

			if len(results) == 0 {
				fmt.Println("✨ No issues found, stack is healthy.")
			} else {
				output.PrintFixResults(results)
				fmt.Println("🎉 Automated self-healing completed!")
			}
		}),
	}

	brokenCmd.AddCommand(stackCmd)
	fixCmd.AddCommand(brokenCmd)
	return fixCmd
}

func initDeployCmd() *cobra.Command {
	var deployCmd = &cobra.Command{
		Use:   "deploy",
		Short: "Deploy services and configurations",
	}

	var newCmd = &cobra.Command{
		Use:   "new",
		Short: "Deploy new resources",
	}

	var serviceCmd = &cobra.Command{
		Use:   "service [compose-file-path] [optional-stack-name]",
		Short: "Deploy a new compose stack/service",
		Args:  cobra.MinimumNArgs(1),
		Run: cmdutil.WithDaemon(func(c *client.Client, cmd *cobra.Command, args []string) {
			filePath := args[0]
			if _, err := os.Stat(filePath); err != nil {
				output.FatalError(fmt.Errorf("compose file not found: %s", filePath))
			}

			base := filepath.Base(filePath)
			stackName := ""
			if strings.HasSuffix(base, "-compose.yml") {
				stackName = strings.TrimSuffix(base, "-compose.yml")
			} else if strings.HasSuffix(base, ".yml") || strings.HasSuffix(base, ".yaml") {
				stackName = strings.TrimSuffix(strings.TrimSuffix(base, ".yml"), ".yaml")
			} else {
				stackName = "custom-service"
			}
			if len(args) > 1 {
				stackName = args[1]
			}

			stackName = strings.ToLower(stackName)
			fmt.Printf("📦 Deploying new service stack %q from %s...\n", stackName, filePath)

			stackDir := system.GetStackDir()
			if err := os.MkdirAll(stackDir, 0755); err != nil {
				output.FatalError(fmt.Errorf("failed to create stacks directory: %w", err))
			}

			destPath := filepath.Join(stackDir, stackName+"-compose.yml")

			src, err := os.Open(filePath)
			if err != nil {
				output.FatalError(fmt.Errorf("failed to open source compose file: %w", err))
			}
			defer src.Close()

			dst, err := os.OpenFile(destPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
			if err != nil {
				output.FatalError(fmt.Errorf("failed to create destination compose file: %w", err))
			}
			defer dst.Close()

			if _, err = io.Copy(dst, src); err != nil {
				output.FatalError(fmt.Errorf("failed to copy compose file: %w", err))
			}

			fmt.Printf("✅ Copied compose file to %s\n", destPath)

			fmt.Println("🚀 Triggering deployment on M3TAL API...")
			_, err = c.StartStack(stackName)
			if err != nil {
				output.FatalError(fmt.Errorf("deployment failed: %w", err))
			}

			fmt.Printf("🎉 Service stack %q deployed successfully!\n", stackName)
		}),
	}

	newCmd.AddCommand(serviceCmd)
	deployCmd.AddCommand(newCmd)
	return deployCmd
}

func initReconcileCmd() *cobra.Command {
	var reconcileCmd = &cobra.Command{
		Use:   "reconcile",
		Short: "Synchronize desired and actual state",
	}

	var systemCmd = &cobra.Command{
		Use:   "system",
		Short: "Trigger state reconciliation loop",
		Run: cmdutil.WithDaemon(func(c *client.Client, cmd *cobra.Command, args []string) {
			fmt.Println("🔄 Forcing system state reconciliation...")
			res, err := c.ReconcileSystem()
			if err != nil {
				output.FatalError(err)
			}

			actionsData, _ := json.Marshal(res["actions"])
			var actions []string
			_ = json.Unmarshal(actionsData, &actions)

			if len(actions) == 0 {
				fmt.Println("🟢 Desired state and actual container states are fully in sync. No actions needed.")
			} else {
				fmt.Printf("🛠️  Reconciliation actions executed:\n")
				for _, act := range actions {
					fmt.Printf("   - %s\n", act)
				}
				fmt.Println("🎉 State reconciliation completed!")
			}
		}),
	}

	reconcileCmd.AddCommand(systemCmd)
	return reconcileCmd
}

func initAgentCmd() *cobra.Command {
	var agentCmd = &cobra.Command{
		Use:   "agent",
		Short: "Manage M3TAL background agents and commands",
	}

	var fixBrokenStackCmd = &cobra.Command{
		Use:   "fix-broken-stack",
		Short: "Detect and repair issues in stacks",
		Run: cmdutil.WithDaemon(func(c *client.Client, cmd *cobra.Command, args []string) {
			fmt.Println("🏥 Running system diagnostics and applying fixes...")
			res, err := c.HandleDoctorFix(true, "")
			if err != nil {
				output.FatalError(err)
			}
			resultsData, _ := json.Marshal(res["results"])
			var results []models.FixResult
			_ = json.Unmarshal(resultsData, &results)
			if len(results) == 0 {
				fmt.Println("✨ No issues found, stack is healthy.")
			} else {
				output.PrintFixResults(results)
				fmt.Println("🎉 Automated self-healing completed!")
			}
		}),
	}

	var deployNewServiceCmd = &cobra.Command{
		Use:   "deploy-new-service [compose-file-path] [optional-stack-name]",
		Short: "Deploy a new compose stack/service",
		Args:  cobra.MinimumNArgs(1),
		Run: cmdutil.WithDaemon(func(c *client.Client, cmd *cobra.Command, args []string) {
			filePath := args[0]
			if _, err := os.Stat(filePath); err != nil {
				output.FatalError(fmt.Errorf("compose file not found: %s", filePath))
			}
			base := filepath.Base(filePath)
			stackName := ""
			if strings.HasSuffix(base, "-compose.yml") {
				stackName = strings.TrimSuffix(base, "-compose.yml")
			} else if strings.HasSuffix(base, ".yml") || strings.HasSuffix(base, ".yaml") {
				stackName = strings.TrimSuffix(strings.TrimSuffix(base, ".yml"), ".yaml")
			} else {
				stackName = "custom-service"
			}
			if len(args) > 1 {
				stackName = args[1]
			}
			stackName = strings.ToLower(stackName)
			fmt.Printf("📦 Deploying new service stack %q from %s...\n", stackName, filePath)
			stackDir := system.GetStackDir()
			_ = os.MkdirAll(stackDir, 0755)
			destPath := filepath.Join(stackDir, stackName+"-compose.yml")
			src, err := os.Open(filePath)
			if err != nil {
				output.FatalError(fmt.Errorf("failed to open source compose file: %w", err))
			}
			defer src.Close()
			dst, err := os.OpenFile(destPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
			if err != nil {
				output.FatalError(fmt.Errorf("failed to create destination compose file: %w", err))
			}
			defer dst.Close()
			if _, err = io.Copy(dst, src); err != nil {
				output.FatalError(fmt.Errorf("failed to copy compose file: %w", err))
			}
			fmt.Printf("✅ Copied compose file to %s\n", destPath)
			fmt.Println("🚀 Triggering deployment on M3TAL API...")
			_, err = c.StartStack(stackName)
			if err != nil {
				output.FatalError(fmt.Errorf("deployment failed: %w", err))
			}
			fmt.Printf("🎉 Service stack %q deployed successfully!\n", stackName)
		}),
	}

	var reconcileSystemCmd = &cobra.Command{
		Use:   "reconcile-system",
		Short: "Trigger state reconciliation loop",
		Run: cmdutil.WithDaemon(func(c *client.Client, cmd *cobra.Command, args []string) {
			fmt.Println("🔄 Forcing system state reconciliation...")
			res, err := c.ReconcileSystem()
			if err != nil {
				output.FatalError(err)
			}
			actionsData, _ := json.Marshal(res["actions"])
			var actions []string
			_ = json.Unmarshal(actionsData, &actions)
			if len(actions) == 0 {
				fmt.Println("🟢 Desired state and actual container states are fully in sync. No actions needed.")
			} else {
				fmt.Printf("🛠️  Reconciliation actions executed:\n")
				for _, act := range actions {
					fmt.Printf("   - %s\n", act)
				}
				fmt.Println("🎉 State reconciliation completed!")
			}
		}),
	}

	agentCmd.AddCommand(fixBrokenStackCmd, deployNewServiceCmd, reconcileSystemCmd)
	return agentCmd
}
