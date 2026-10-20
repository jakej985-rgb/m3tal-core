package main

import (
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/jakej985-rgb/m3tal-core/pkg/client"
	"github.com/jakej985-rgb/m3tal-core/pkg/cmdutil"
	"github.com/jakej985-rgb/m3tal-core/pkg/output"
	"github.com/jakej985-rgb/m3tal-core/pkg/registry"
	"github.com/jakej985-rgb/m3tal-core/pkg/system"
	"github.com/spf13/cobra"
)

func initStackCmd() *cobra.Command {
	var stackCmd = &cobra.Command{
		Use:   "stack",
		Short: "Manage prebuilt M3TAL compose stacks",
		Long:  `Search, install, and update prebuilt compose stacks from the official M3TAL stacks registry.`,
	}

	var searchCmd = &cobra.Command{
		Use:   "search [term]",
		Short: "Search available stacks in the registry",
		Run: cmdutil.WithClient(func(c *client.Client, cmd *cobra.Command, args []string) {
			regURL := registry.GetRegistryURL()
			idx, err := registry.FetchIndex(regURL)
			if err != nil {
				output.FatalError(err)
			}

			term := ""
			if len(args) > 0 {
				term = strings.ToLower(args[0])
			}

			var filtered []registry.StackMetadata
			for _, s := range idx.Stacks {
				if term == "" ||
					strings.Contains(strings.ToLower(s.Name), term) ||
					strings.Contains(strings.ToLower(s.Category), term) ||
					strings.Contains(strings.ToLower(s.Description), term) {
					filtered = append(filtered, s)
				}
			}

			if len(filtered) == 0 {
				fmt.Println("No matching stacks found in the registry.")
				return
			}

			output.PrintRegistrySearchTable(filtered)
		}),
	}

	var installCmd = &cobra.Command{
		Use:   "install [name]",
		Short: "Install a prebuilt stack into M3TAL",
		Args:  cobra.ExactArgs(1),
		Run: cmdutil.WithDaemon(func(c *client.Client, cmd *cobra.Command, args []string) {
			name := args[0]
			regURL := registry.GetRegistryURL()
			destDir := system.GetStackDir()

			// 1. Fetch index to locate stack & metadata
			idx, err := registry.FetchIndex(regURL)
			if err != nil {
				output.FatalError(err)
			}

			var target *registry.StackMetadata
			for i := range idx.Stacks {
				if strings.EqualFold(idx.Stacks[i].Name, name) {
					target = &idx.Stacks[i]
					break
				}
			}

			if target == nil {
				output.FatalErrorMsg("Stack %q not found in the registry.", name)
			}

			// 2. Validate pre-flight checks
			warnings := registry.ValidateRequirements(target, destDir)
			if len(warnings) > 0 {
				fmt.Println("⚠️  Pre-flight checks yielded the following warnings:")
				for _, w := range warnings {
					fmt.Printf("   - %s\n", w)
				}
				fmt.Print("👉 Would you like to proceed with installation anyway? (y/n): ")
				var confirm string
				fmt.Scanln(&confirm)
				if !strings.EqualFold(confirm, "y") && !strings.EqualFold(confirm, "yes") {
					fmt.Println("❌ Installation cancelled.")
					return
				}
			}

			// 3. Download the stack compose and metadata files
			fmt.Printf("📥 Downloading stack %q from registry...\n", target.Name)
			if err := registry.DownloadStack(target.Name, regURL, destDir); err != nil {
				output.FatalError(err)
			}

			// 4. Activate the stack by calling API
			fmt.Printf("🚀 Deploying stack %s via API...\n", target.Name)
			_, err = c.StartStack(strings.ToLower(target.Name))
			if err != nil {
				fmt.Printf("⚠️  Failed to automatically deploy stack %s: %v\n", target.Name, err)
				fmt.Printf("👉 You can deploy it manually using: m3tal up %s\n", target.Name)
			} else {
				fmt.Printf("✅ Stack %s successfully installed and started!\n", target.Name)
			}
		}),
	}

	var updateCmd = &cobra.Command{
		Use:   "update [name]",
		Short: "Update installed prebuilt stacks to the latest version",
		Run: cmdutil.WithDaemon(func(c *client.Client, cmd *cobra.Command, args []string) {
			regURL := registry.GetRegistryURL()
			destDir := system.GetStackDir()

			idx, err := registry.FetchIndex(regURL)
			if err != nil {
				output.FatalError(err)
			}

			type updateJob struct {
				meta   registry.StackMetadata
				oldVer string
			}
			var jobs []updateJob

			if len(args) > 0 {
				// Update single stack specifically
				name := args[0]
				var target *registry.StackMetadata
				for i := range idx.Stacks {
					if strings.EqualFold(idx.Stacks[i].Name, name) {
						target = &idx.Stacks[i]
						break
					}
				}
				if target == nil {
					output.FatalErrorMsg("Stack %q not found in registry.", name)
				}

				oldVer, err := readLocalVersion(destDir, target.Name)
				if err != nil {
					// Assume not installed or no metadata, proceed to install
					oldVer = "none"
				}

				if oldVer == "none" || oldVer != target.Version {
					jobs = append(jobs, updateJob{meta: *target, oldVer: oldVer})
				} else {
					fmt.Printf("✅ Stack %s is already up to date (version %s).\n", target.Name, target.Version)
				}
			} else {
				// Scan and update all installed stacks
				filepath.WalkDir(destDir, func(path string, d fs.DirEntry, err error) error {
					if err != nil || d.IsDir() {
						return nil
					}
					if strings.HasSuffix(d.Name(), ".stack.json") {
						stackName := strings.TrimSuffix(d.Name(), ".stack.json")
						// Find matching stack in registry index
						for _, s := range idx.Stacks {
							if strings.EqualFold(s.Name, stackName) {
								oldVer, err := readLocalVersion(destDir, s.Name)
								if err == nil && oldVer != s.Version {
									jobs = append(jobs, updateJob{meta: s, oldVer: oldVer})
								}
								break
							}
						}
					}
					return nil
				})
			}

			if len(jobs) == 0 {
				fmt.Println("All installed stacks are up to date.")
				return
			}

			for _, job := range jobs {
				fmt.Printf("🔄 Updating stack %s from version %s to %s...\n", job.meta.Name, job.oldVer, job.meta.Version)
				if err := registry.DownloadStack(job.meta.Name, regURL, destDir); err != nil {
					fmt.Printf("❌ Failed to update stack %s: %v\n", job.meta.Name, err)
					continue
				}

				fmt.Printf("🚀 Redeploying updated stack %s via API...\n", job.meta.Name)
				_, err = c.StartStack(strings.ToLower(job.meta.Name))
				if err != nil {
					fmt.Printf("⚠️  Redeployment failed for stack %s: %v\n", job.meta.Name, err)
				} else {
					fmt.Printf("✅ Stack %s updated successfully!\n", job.meta.Name)
				}
			}
		}),
	}

	stackCmd.AddCommand(searchCmd, installCmd, updateCmd)
	return stackCmd
}

type localStackMetadata struct {
	Name    string `json:"name"`
	Version string `json:"version"`
}

func readLocalVersion(stackDir, name string) (string, error) {
	metaPath := filepath.Join(stackDir, fmt.Sprintf("%s.stack.json", strings.ToLower(name)))
	data, err := os.ReadFile(metaPath)
	if err != nil {
		return "", err
	}
	var meta localStackMetadata
	if err := json.Unmarshal(data, &meta); err != nil {
		return "", err
	}
	return meta.Version, nil
}
