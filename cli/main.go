package main

import (
	"bufio"
	"crypto/rand"
	_ "embed"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"syscall"
	"time"

	"github.com/jakej985-rgb/m3tal-core/api"
	"github.com/jakej985-rgb/m3tal-core/pkg/client"
	"github.com/jakej985-rgb/m3tal-core/pkg/cmdutil"
	"github.com/jakej985-rgb/m3tal-core/pkg/compose"
	"github.com/jakej985-rgb/m3tal-core/pkg/config"
	"github.com/jakej985-rgb/m3tal-core/pkg/models"
	"github.com/jakej985-rgb/m3tal-core/pkg/output"
	"github.com/jakej985-rgb/m3tal-core/pkg/system"
	"github.com/spf13/cobra"
)

// Version is injected during build via ldflags.
var Version = "0.1.0"

//go:embed .env.example
var envExample string

func main() {
	exeName := filepath.Base(os.Args[0])
	switch {
	case strings.Contains(exeName, "sysmon") || (len(os.Args) > 1 && os.Args[1] == "sysmon"):
		system.SetProcessName("m3tal-sysmon")
		if len(os.Args) == 1 {
			os.Args = append(os.Args, "sysmon")
		}
	case strings.Contains(exeName, "web") || strings.Contains(exeName, "headless") || (len(os.Args) > 1 && (os.Args[1] == "web" || os.Args[1] == "headless")):
		system.SetProcessName("m3tal-web")
		if len(os.Args) == 1 {
			os.Args = append(os.Args, "headless")
		}
	case strings.Contains(exeName, "docker") || (len(os.Args) > 1 && (os.Args[1] == "docker-daemon" || os.Args[1] == "docker")):
		system.SetProcessName("m3tal-docker")
		if len(os.Args) == 1 {
			os.Args = append(os.Args, "docker-daemon")
		}
	case strings.Contains(exeName, "api") || (len(os.Args) > 1 && os.Args[1] == "api"):
		system.SetProcessName("m3tal-api")
		if len(os.Args) == 1 {
			os.Args = append(os.Args, "api")
		}
	case strings.Contains(exeName, "daemon") || (len(os.Args) > 1 && os.Args[1] == "daemon"):
		system.SetProcessName("m3tal-core")
	case strings.Contains(exeName, "gui") || strings.Contains(exeName, "ui") || (len(os.Args) > 1 && os.Args[1] == "ui") || len(os.Args) == 1:
		system.SetProcessName("m3tal-gui")
	default:
		system.SetProcessName("m3tal-core")
	}
	log.Println("[cli] M3TAL CLI invoked (Command:", os.Args, ")")
	// First-run check for Linux system installations
	if runtime.GOOS == "linux" && os.Geteuid() != 0 {
		configPath := system.GetConfigPath()
		if _, err := os.Stat(configPath); os.IsNotExist(err) {
			fmt.Printf("⚠️  System configuration not found at %s\n", configPath)
			fmt.Println("👉 Run: sudo m3tal init")
			fmt.Println("")
		}
	}

	var rootCmd = &cobra.Command{
		Use:     "m3tal",
		Short:   "M3TAL Core Desktop Application & Orchestrator",
		Version: Version,
		Run: func(cmd *cobra.Command, args []string) {
			port, _ := cmd.Flags().GetString("port")
			headless, _ := cmd.Flags().GetBool("headless")
			if !cmd.Flags().Changed("port") {
				if ep := os.Getenv("API_PORT"); ep != "" {
					port = ep
				} else if ep := os.Getenv("PORT"); ep != "" {
					port = ep
				}
			}
			launchDesktopApp(port, headless)
		},
	}

	rootCmd.Flags().Bool("headless", false, "Run in headless mode (do not launch desktop window)")
	rootCmd.Flags().String("port", "5050", "Port to serve the Web GUI on")
	rootCmd.PersistentFlags().String("api-url", "http://localhost:5050", "M3TAL API URL")
	rootCmd.PersistentFlags().String("api-token", os.Getenv("API_TOKEN"), "M3TAL API Token")
	rootCmd.PersistentFlags().Bool("local", false, "Force local execution (skip API)")
	var listCmd = &cobra.Command{
		Use:   "list",
		Short: "List all containers",
		Run: cmdutil.WithDaemon(func(c *client.Client, cmd *cobra.Command, args []string) {
			list, err := c.GetContainers()
			if err != nil {
				output.FatalError(err)
			}
			output.PrintContainersTable(list)
		}),
	}

	var psCmd = &cobra.Command{
		Use:   "ps",
		Short: "List all containers (alias for list)",
		Run:   listCmd.Run,
	}

	var startCmd = &cobra.Command{
		Use:   "start [name]",
		Short: "Start a container",
		Args:  cobra.ExactArgs(1),
		Run: cmdutil.WithDaemon(func(c *client.Client, cmd *cobra.Command, args []string) {
			err := c.ControlContainer(args[0], "start")
			if err != nil {
				output.FatalError(err)
			}
			fmt.Printf("Started %s (via API)\n", args[0])
		}),
	}

	var stopCmd = &cobra.Command{
		Use:   "stop [name]",
		Short: "Stop a container",
		Args:  cobra.ExactArgs(1),
		Run: cmdutil.WithDaemon(func(c *client.Client, cmd *cobra.Command, args []string) {
			err := c.ControlContainer(args[0], "stop")
			if err != nil {
				output.FatalError(err)
			}
			fmt.Printf("Stopped %s (via API)\n", args[0])
		}),
	}

	var statsCmd = &cobra.Command{
		Use:   "stats",
		Short: "Show comprehensive system, GPU, temperature, and storage metrics",
		Run: cmdutil.WithDaemon(func(c *client.Client, cmd *cobra.Command, args []string) {
			stats, err := c.GetTrayStats()
			if err != nil {
				output.FatalError(err)
			}
			output.PrintDetailedStats(stats)
		}),
	}

	var daemonCmd = &cobra.Command{
		Use:   "daemon",
		Short: "Manage M3TAL background API daemon and agents",
		Run: func(cmd *cobra.Command, args []string) {
			system.SetProcessName("m3tal-core")
			unlock, acquired := system.AcquireProcessLock("m3tal-core")
			if !acquired {
				log.Println("⚠️ m3tal-core daemon is already running. Exiting duplicate instance.")
				return
			}
			defer unlock()

			port := os.Getenv("API_PORT")
			if port == "" {
				port = "5050"
			}
			addr := fmt.Sprintf("127.0.0.1:%s", port)
			conn, err := net.DialTimeout("tcp", addr, 250*time.Millisecond)
			if err != nil {
				token := os.Getenv("API_TOKEN")
				if token == "" {
					token = "m3tal-secret-token"
				}
				go func() {
					log.Printf("🚀 Starting M3TAL API & Web GUI server on :%s\n", port)
					if err := api.RunServer(port, token); err != nil {
						log.Printf("⚠️  API server stopped: %v\n", err)
					}
				}()
			} else {
				conn.Close()
			}

			exe, err := os.Executable()
			if err != nil {
				exe = "m3tal"
			}

			// Spawn dedicated sysmon collector process (m3tal-sysmon)
			sysmonCmd := exec.Command(exe, "sysmon")
			sysmonCmd.Stdout = os.Stdout
			sysmonCmd.Stderr = os.Stderr
			if err := sysmonCmd.Start(); err != nil {
				log.Printf("⚠️ Failed to start m3tal-sysmon child process: %v", err)
			}

			sigChan := make(chan os.Signal, 1)
			signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
			<-sigChan

			if sysmonCmd != nil && sysmonCmd.Process != nil {
				_ = sysmonCmd.Process.Kill()
			}
		},
	}

	var daemonStartCmd = &cobra.Command{
		Use:   "start",
		Short: "Start the background M3TAL API server daemon",
		Run: cmdutil.WithClient(func(c *client.Client, cmd *cobra.Command, args []string) {
			status, err := c.GetStatus()
			if err == nil && status != nil {
				fmt.Println("🟢 M3TAL API daemon is already running.")
				return
			}

			fmt.Println("🚀 Starting M3TAL API server daemon...")
			exe, err := os.Executable()
			if err != nil {
				exe = os.Args[0]
			}

			cmdDaemon := exec.Command(exe, "api")
			cmdDaemon.Env = os.Environ()
			cmdDaemon.SysProcAttr = &syscall.SysProcAttr{
				Setsid: true,
			}
			logPath := filepath.Join(os.TempDir(), "m3tal-api.log")
			if lf, err := os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644); err == nil {
				cmdDaemon.Stdout = lf
				cmdDaemon.Stderr = lf
			}
			if err := cmdDaemon.Start(); err != nil {
				output.FatalErrorMsg("Failed to start API server daemon: %v", err)
			} else {
				fmt.Println("✅ M3TAL API server started in background (PID:", cmdDaemon.Process.Pid, ")")
				_ = cmdDaemon.Process.Release()
			}

			// Also start the M3TAL Agents daemon
			fmt.Println("🚀 Starting M3TAL Agents Daemon...")
			cmdAgent := exec.Command("sudo", "systemctl", "start", "m3tal.service")
			if err := cmdAgent.Run(); err != nil {
				cmdAgentDaemon := exec.Command(exe, "daemon")
				cmdAgentDaemon.Env = os.Environ()
				cmdAgentDaemon.SysProcAttr = &syscall.SysProcAttr{
					Setsid: true,
				}
				logPath := filepath.Join(os.TempDir(), "m3tal-daemon.log")
				if lf, err := os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644); err == nil {
					cmdAgentDaemon.Stdout = lf
					cmdAgentDaemon.Stderr = lf
				}
				if err := cmdAgentDaemon.Start(); err != nil {
					fmt.Printf("⚠️  Failed to start agents daemon: %v\n", err)
				} else {
					fmt.Println("✅ Agents daemon started in background (PID:", cmdAgentDaemon.Process.Pid, ")")
					_ = cmdAgentDaemon.Process.Release()
				}
			} else {
				fmt.Println("✅ Started m3tal.service via systemd.")
			}
		}),
	}

	var daemonStopCmd = &cobra.Command{
		Use:   "stop",
		Short: "Stop the background M3TAL API server daemon",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Println("🛑 Stopping M3TAL API Server Daemon...")
			_ = exec.Command("pkill", "-f", "m3tal api").Run()
			fmt.Println("✅ Stopped background API processes.")

			fmt.Println("🛑 Stopping M3TAL Agents Daemon...")
			cmdAgent := exec.Command("sudo", "systemctl", "stop", "m3tal.service")
			if err := cmdAgent.Run(); err != nil {
				_ = exec.Command("pkill", "-f", "m3tal daemon").Run()
				fmt.Println("✅ Stopped background agent processes.")
			} else {
				fmt.Println("✅ Stopped m3tal.service via systemd.")
			}
		},
	}

	var daemonStatusCmd = &cobra.Command{
		Use:   "status",
		Short: "Check background M3TAL API server status",
		Run: cmdutil.WithClient(func(c *client.Client, cmd *cobra.Command, args []string) {
			status, err := c.GetStatus()
			if err != nil {
				fmt.Printf("🔴 Status: Disconnected (API daemon is not running)\n")
				return
			}
			fmt.Println("🌐 M3TAL API Status:")
			fmt.Println("----------------------------------------")
			fmt.Printf("🟢 Status:     Running (Healthy)\n")
			fmt.Printf("🏥 Health:     %s\n", status.Status)
			if len(status.Components) > 0 {
				fmt.Println("🧩 Components:")
				for comp, compStat := range status.Components {
					fmt.Printf("   - %s: %s\n", comp, compStat)
				}
			}
		}),
	}

	daemonCmd.AddCommand(daemonStartCmd, daemonStopCmd, daemonStatusCmd)

	var apiCmd = &cobra.Command{
		Use:   "api",
		Short: "Run the M3TAL API server",
		Run: func(cmd *cobra.Command, args []string) {
			port, _ := cmd.Flags().GetString("port")
			token := os.Getenv("API_TOKEN")
			if token == "" {
				token = "m3tal-secret-token"
			}
			if err := api.RunServer(port, token); err != nil {
				log.Fatalf("❌ API server failed: %v", err)
			}
		},
	}
	apiCmd.Flags().String("port", "5050", "Port to listen on")

	var upCmd = &cobra.Command{
		Use:   "up [stack]",
		Short: "Initialize and start stacks",
		Run: cmdutil.WithDaemon(func(c *client.Client, cmd *cobra.Command, args []string) {
			if len(args) > 0 {
				target := args[0]
				fmt.Printf("🚀 Deploying stack %s via API...\n", target)
				_, err := c.StartStack(target)
				if err != nil {
					output.FatalError(err)
				}
				fmt.Println("✅ Stack deployed successfully.")
			} else {
				fmt.Println("🚀 Deploying all stacks via API...")
				stacks, err := c.GetStacks()
				if err != nil {
					output.FatalError(err)
				}
				if len(stacks) == 0 {
					fmt.Println("No Compose Found! Nothing To Start!")
					return
				}
				for _, s := range stacks {
					fmt.Printf("🚀 Deploying stack %s via API...\n", s.Name)
					_, _ = c.StartStack(s.Name)
				}
				fmt.Println("✅ All stacks deployed.")
			}
		}),
	}

	var downCmd = &cobra.Command{
		Use:   "down [stack]",
		Short: "Stop stacks",
		Run: cmdutil.WithDaemon(func(c *client.Client, cmd *cobra.Command, args []string) {
			if len(args) > 0 {
				target := args[0]
				fmt.Printf("🛑 Stopping stack %s via API...\n", target)
				_, err := c.StopStack(target)
				if err != nil {
					output.FatalError(err)
				}
				fmt.Println("✅ Stack stopped successfully.")
			} else {
				fmt.Println("🛑 Stopping all stacks via API...")
				stacks, err := c.GetStacks()
				if err != nil {
					output.FatalError(err)
				}
				for _, s := range stacks {
					fmt.Printf("🛑 Stopping stack %s via API...\n", s.Name)
					_, _ = c.StopStack(s.Name)
				}
				fmt.Println("✅ All stacks stopped.")
			}
		}),
	}

	var restartCmd = &cobra.Command{
		Use:   "restart [stack]",
		Short: "Restart stacks",
		Run: cmdutil.WithDaemon(func(c *client.Client, cmd *cobra.Command, args []string) {
			if len(args) > 0 {
				target := args[0]
				fmt.Printf("🔄 Restarting stack %s via API...\n", target)
				_, err := c.RestartStack(target)
				if err != nil {
					output.FatalError(err)
				}
				fmt.Println("✅ Stack restarted successfully.")
			} else {
				fmt.Println("🔄 Restarting all stacks via API...")
				stacks, err := c.GetStacks()
				if err != nil {
					output.FatalError(err)
				}
				for _, s := range stacks {
					fmt.Printf("🔄 Restarting stack %s via API...\n", s.Name)
					_, _ = c.RestartStack(s.Name)
				}
				fmt.Println("✅ All stacks restarted.")
			}
		}),
	}

	var logsCmd = &cobra.Command{
		Use:   "logs [container]",
		Short: "View logs from a container",
		Args:  cobra.ExactArgs(1),
		Run: cmdutil.WithDaemon(func(c *client.Client, cmd *cobra.Command, args []string) {
			target := args[0]
			logs, err := c.GetLogs(target)
			if err != nil {
				output.FatalError(err)
			}
			fmt.Println(logs)
		}),
	}

	var stacksCmd = &cobra.Command{
		Use:   "stacks",
		Short: "List all stacks",
		Run: cmdutil.WithDaemon(func(c *client.Client, cmd *cobra.Command, args []string) {
			stacks, err := c.GetStacks()
			if err != nil {
				output.FatalError(err)
			}
			output.PrintStacksTable(stacks)
		}),
	}

	var pullCmd = &cobra.Command{
		Use:   "pull",
		Short: "Pull latest images",
		Run: cmdutil.WithDaemon(func(c *client.Client, cmd *cobra.Command, args []string) {
			fmt.Println("📥 Pulling latest service images via API...")
			_, err := c.PullStacks("")
			if err != nil {
				output.FatalError(err)
			}
			fmt.Println("✅ Images updated.")
		}),
	}

	var dashpassCmd = &cobra.Command{
		Use:   "dashpass [username] [password]",
		Short: "Manage dashboard users",
		Run: cmdutil.WithDaemon(func(c *client.Client, cmd *cobra.Command, args []string) {
			username := ""
			if len(args) > 0 {
				username = args[0]
			} else {
				fmt.Print("Enter username: ")
				fmt.Scanln(&username)
				username = strings.TrimSpace(username)
				if username == "" {
					fmt.Println("❌ Username cannot be empty.")
					return
				}
			}

			password := ""
			if len(args) > 1 {
				password = args[1]
			} else {
				fmt.Printf("Password for %s: ", username)
				fmt.Scanln(&password)
			}

			fmt.Printf("✅ Updating user %s via API...\n", username)
			if err := c.UpdateDashpass(username, password); err != nil {
				log.Fatal(err)
			}
			fmt.Printf("✅ User %s updated successfully.\n", username)
		}),
	}

	var docCmd = &cobra.Command{
		Use:   "doctor",
		Short: "Run comprehensive pre-flight health check",
		Long: `Checks Docker daemon connectivity, .env file validity,
storage path accessibility, port availability, and system configuration.
Run this before 'm3tal up' to diagnose potential issues.`,
		Run: cmdutil.WithDaemon(func(c *client.Client, cmd *cobra.Command, args []string) {
			results, err := c.GetDoctor()
			if err != nil {
				output.FatalError(err)
			}
			output.PrintPreflightResults(results)
		}),
	}

	var initCmd = &cobra.Command{
		Use:   "init",
		Short: "Initialize environment and generate secrets",
		Run: func(cmd *cobra.Command, args []string) {
			if _, err := os.Stat(system.GetConfigPath()); err == nil {
				fmt.Println("⚠️  .env already exists. Use 'm3tal config wizard' to update.")
				return
			}
			runWizard(system.GetConfigPath(), "", false, true)

			// Post-init storage path validation
			envData, err := os.ReadFile(system.GetConfigPath())
			if err == nil {
				basePath := getEnvValue(string(envData), "BASE_STORAGE_PATH")
				if basePath != "" {
					if err := localValidateStoragePath(basePath); err != nil {
						fmt.Printf("\n⚠️  Storage path validation:\n    %v\n", err)
						fmt.Println("\n👉 Set a valid BASE_STORAGE_PATH in .env, then run:")
						fmt.Println("   mkdir -p /path/to/your/m3tal-data")
						fmt.Println("   ./m3tal config set BASE_STORAGE_PATH /path/to/your/m3tal-data")
					} else {
						fmt.Printf("✅ Storage path validated: %s\n", basePath)
					}
				}
			}

			// Initialize plugin directory structure
			if runtime.GOOS == "linux" && os.Geteuid() == 0 {
				for _, subdir := range system.PluginSubdirs {
					path := filepath.Join(system.UserPluginsDir, subdir)
					_ = os.MkdirAll(path, 0755)
				}
				fmt.Printf("✅ Plugin directories initialized (%s)\n", system.UserPluginsDir)
			}
		},
	}

	var configCmd = &cobra.Command{
		Use:   "config",
		Short: "Manage M3TAL environment variables",
		Run: func(cmd *cobra.Command, args []string) {
			if len(args) == 0 {
				runWizard(system.GetConfigPath(), "", true, true)
				return
			}
		},
	}

	var configWizardCmd = &cobra.Command{
		Use:   "wizard",
		Short: "Run the interactive configuration wizard",
		Run: func(cmd *cobra.Command, args []string) {
			target, _ := cmd.Flags().GetString("target")
			composeFile, _ := cmd.Flags().GetString("compose")
			isGlobal := false

			if target == "" {
				target = system.GetConfigPath()
				isGlobal = true
			}

			runWizard(target, composeFile, true, isGlobal)
		},
	}
	configWizardCmd.Flags().String("target", "", "Target .env file to save")
	configWizardCmd.Flags().String("compose", "", "Compose file to read required variables from")

	var configListCmd = &cobra.Command{
		Use:   "list",
		Short: "List all configuration variables",
		Run: func(cmd *cobra.Command, args []string) {
			content, err := os.ReadFile(system.GetConfigPath())
			if err != nil {
				log.Fatalf("❌ Configuration not found. Run 'init' first.")
			}
			fmt.Println(string(content))
		},
	}

	var configSetCmd = &cobra.Command{
		Use:   "set [key] [value]",
		Short: "Set a configuration variable",
		Args:  cobra.ExactArgs(2),
		Run: func(cmd *cobra.Command, args []string) {
			key := args[0]
			val := args[1]

			content, err := os.ReadFile(system.GetConfigPath())
			if err != nil {
				log.Fatalf("❌ Configuration not found. Run 'init' first.")
			}

			newContent := replaceSecret(string(content), key+"=", val)
			// If not found, append
			if !strings.Contains(newContent, key+"=") {
				newContent += fmt.Sprintf("\n%s=%s", key, val)
			}

			if err := os.WriteFile(system.GetConfigPath(), []byte(newContent), 0600); err != nil {
				log.Fatal(err)
			}
			fmt.Printf("✅ Config updated: %s=%s\n", key, val)
		},
	}

	var configGetCmd = &cobra.Command{
		Use:   "get [key]",
		Short: "Get a configuration variable",
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			key := args[0]
			content, err := os.ReadFile(system.GetConfigPath())
			if err != nil {
				log.Fatalf("❌ Configuration not found. Run 'init' first.")
			}

			lines := strings.Split(string(content), "\n")
			for _, line := range lines {
				if strings.HasPrefix(line, key+"=") {
					fmt.Println(strings.TrimPrefix(line, key+"="))
					return
				}
			}
			fmt.Printf("❌ Key '%s' not found.\n", key)
		},
	}

	var configScanCmd = &cobra.Command{
		Use:   "scan",
		Short: "Scan available Docker stacks",
		Run: func(cmd *cobra.Command, args []string) {
			stackDir := system.GetStackDir()
			matches, err := system.FindComposeFiles(stackDir)
			if err != nil {
				fmt.Println("❌ Unable to scan docker directory:", err)
				return
			}
			type stackInfo struct {
				Compose  string `json:"compose"`
				Template string `json:"template,omitempty"`
				Env      string `json:"env,omitempty"`
			}
			stacks := make(map[string]stackInfo)
			for _, match := range matches {
				name := filepath.Base(match)
				stack := strings.TrimSuffix(name, "-compose.yml")
				dir := filepath.Dir(match)
				templatePath := filepath.Join(dir, stack+".env.template")
				envPath := filepath.Join(dir, stack+".env")

				hasTemplate := false
				hasEnv := false

				if _, err := os.Stat(templatePath); err == nil {
					hasTemplate = true
				}
				if _, err := os.Stat(envPath); err == nil {
					hasEnv = true
				}

				if !hasTemplate && !hasEnv {
					continue
				}

				info := stackInfo{Compose: match}
				if hasTemplate {
					info.Template = templatePath
				}
				if hasEnv {
					info.Env = envPath
				}

				stacks[stack] = info
			}
			printJSON(stacks)
		},
	}

	var configCloudflaredCmd = &cobra.Command{
		Use:   "cloudflared",
		Short: "View or edit cloudflared configuration",
		Run: func(cmd *cobra.Command, args []string) {
			edit, _ := cmd.Flags().GetBool("edit")
			configPath := filepath.Join(system.GetStackDir(), "cloudflared-config.yml")

			if edit {
				if _, err := os.Stat(configPath); os.IsNotExist(err) {
					defaultConfig := `tunnel: YOUR_TUNNEL_ID
credentials-file: /etc/cloudflared/credentials.json

ingress:
  - hostname: dash.${DOMAIN}
    service: http://traefik:80

  - hostname: api.${DOMAIN}
    service: http://traefik:80

  - service: http_status:404
`
					_ = os.WriteFile(configPath, []byte(defaultConfig), 0644)
				}

				editor := os.Getenv("EDITOR")
				if editor == "" {
					editor = "nano"
				}

				c := exec.Command(editor, configPath)
				c.Stdin = os.Stdin
				c.Stdout = os.Stdout
				c.Stderr = os.Stderr
				if err := c.Run(); err != nil {
					log.Fatalf("❌ Editor failed: %v", err)
				}
				fmt.Println("✅ cloudflared configuration updated.")
			} else {
				content, err := os.ReadFile(configPath)
				if err != nil {
					if os.IsNotExist(err) {
						fmt.Println("⚠️  cloudflared configuration does not exist yet. Run 'm3tal config cloudflared --edit' to create it.")
						return
					}
					log.Fatalf("❌ Failed to read cloudflared config: %v", err)
				}
				fmt.Println(string(content))
			}
		},
	}
	configCloudflaredCmd.Flags().BoolP("edit", "e", false, "Edit the cloudflared configuration file in default editor")

	configCmd.AddCommand(configListCmd, configSetCmd, configGetCmd, configScanCmd, configWizardCmd, configCloudflaredCmd)

	// ─── VPN Commands ───
	var vpnCmd = &cobra.Command{
		Use:   "vpn",
		Short: "Manage Gluetun VPN connections, ports, and leak detection",
		Run: func(cmd *cobra.Command, args []string) {
			cmd.Help()
		},
	}

	var vpnStatusCmd = &cobra.Command{
		Use:   "status",
		Short: "Check VPN connection status and settings",
		Run: cmdutil.WithDaemon(func(c *client.Client, cmd *cobra.Command, args []string) {
			status, err := c.GetVPNStatus()
			if err != nil {
				output.FatalError(err)
			}
			output.PrintVPNStatus(status)
		}),
	}

	var vpnStartCmd = &cobra.Command{
		Use:   "start",
		Short: "Start the VPN connection (gluetun container)",
		Run: cmdutil.WithDaemon(func(c *client.Client, cmd *cobra.Command, args []string) {
			_, err := c.ControlVPN("start")
			if err != nil {
				output.FatalError(err)
			}
			fmt.Println("✅ VPN container start command sent successfully.")
		}),
	}

	var vpnStopCmd = &cobra.Command{
		Use:   "stop",
		Short: "Stop the VPN connection (gluetun container)",
		Run: cmdutil.WithDaemon(func(c *client.Client, cmd *cobra.Command, args []string) {
			_, err := c.ControlVPN("stop")
			if err != nil {
				output.FatalError(err)
			}
			fmt.Println("✅ VPN container stop command sent successfully.")
		}),
	}

	var vpnRestartCmd = &cobra.Command{
		Use:   "restart",
		Short: "Restart the VPN connection (gluetun container)",
		Run: cmdutil.WithDaemon(func(c *client.Client, cmd *cobra.Command, args []string) {
			_, err := c.ControlVPN("restart")
			if err != nil {
				output.FatalError(err)
			}
			fmt.Println("✅ VPN container restart command sent successfully.")
		}),
	}

	var vpnRegionCmd = &cobra.Command{
		Use:   "region [region-name]",
		Short: "Switch VPN connection region",
		Args:  cobra.ExactArgs(1),
		Run: cmdutil.WithDaemon(func(c *client.Client, cmd *cobra.Command, args []string) {
			targetRegion := args[0]
			fmt.Printf("🔄 Switching VPN region to %s...\n", targetRegion)
			_, err := c.SwitchVPNRegion(targetRegion)
			if err != nil {
				output.FatalError(err)
			}
			fmt.Println("✅ Region updated in configuration and stack restarted.")
		}),
	}

	var vpnSyncCmd = &cobra.Command{
		Use:   "sync",
		Short: "Manually sync Gluetun forwarded port to dependent containers (e.g. qBittorrent)",
		Run: cmdutil.WithDaemon(func(c *client.Client, cmd *cobra.Command, args []string) {
			fmt.Println("🔄 Querying Gluetun forwarded port and syncing...")
			port, err := c.SyncVPNPort()
			if err != nil {
				output.FatalError(err)
			}
			fmt.Printf("✅ Port %d synced to dependent services successfully.\n", port)
		}),
	}

	var vpnCheckCmd = &cobra.Command{
		Use:   "check",
		Short: "Run leak detection and verify kill switch status",
		Run: cmdutil.WithDaemon(func(c *client.Client, cmd *cobra.Command, args []string) {
			fmt.Println("🔍 Running leak check...")
			report, err := c.CheckVPNLeak()
			if err != nil {
				output.FatalError(err)
			}
			output.PrintVPNLeakReport(report)
		}),
	}

	vpnCmd.AddCommand(vpnStatusCmd, vpnStartCmd, vpnStopCmd, vpnRestartCmd, vpnRegionCmd, vpnSyncCmd, vpnCheckCmd)

	// ─── Compose Commands ───
	var composeCmd = &cobra.Command{
		Use:   "compose",
		Short: "Smart Docker Compose Editor & Linter",
		Run: func(cmd *cobra.Command, args []string) {
			cmd.Help()
		},
	}

	var composeLintCmd = &cobra.Command{
		Use:   "lint [file]",
		Short: "Lint a Docker Compose file for errors and best practices",
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			data, err := os.ReadFile(args[0])
			if err != nil {
				log.Fatalf("❌ Failed to read compose file: %v", err)
			}

			cfg, err := compose.Parse(data)
			if err != nil {
				log.Fatalf("❌ YAML Parse Error: %v", err)
			}

			issues := compose.Lint(cfg)
			if len(issues) == 0 {
				fmt.Println("✅ No issues found! Your compose file follows best practices.")
				return
			}

			fmt.Printf("📋 Found %d issue(s) in %s:\n", len(issues), args[0])
			for _, issue := range issues {
				sevEmoji := "⚠️ "
				if issue.Severity == compose.SeverityError {
					sevEmoji = "❌"
				}
				svcInfo := ""
				if issue.Service != "" {
					svcInfo = fmt.Sprintf(" (service: %s)", issue.Service)
				}
				fmt.Printf("%s [%s]%s %s\n", sevEmoji, issue.Severity, svcInfo, issue.Message)
			}
		},
	}

	var composeFixCmd = &cobra.Command{
		Use:   "fix [file]",
		Short: "Auto-fix common Docker Compose issues",
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			dryRun, _ := cmd.Flags().GetBool("dry-run")

			data, err := os.ReadFile(args[0])
			if err != nil {
				log.Fatalf("❌ Failed to read compose file: %v", err)
			}

			fixed, fixes, err := compose.AutoFix(data)
			if err != nil {
				log.Fatalf("❌ Auto-fix Error: %v", err)
			}

			if len(fixes) == 0 {
				fmt.Println("✨ No issues needed fixing.")
				return
			}

			fmt.Println("🛠️  Applied fixes:")
			for _, fix := range fixes {
				fmt.Printf(" - %s\n", fix)
			}

			if dryRun {
				fmt.Println("\n📝 Dry-run requested. Preview of fixed YAML:")
				fmt.Println(string(fixed))
			} else {
				if err := os.WriteFile(args[0], fixed, 0644); err != nil {
					log.Fatalf("❌ Failed to write fixed compose file: %v", err)
				}
				fmt.Println("\n💾 Fixes successfully saved to file.")
			}
		},
	}

	composeFixCmd.Flags().Bool("dry-run", false, "Preview fixes without modifying the file")

	var composeGenerateCmd = &cobra.Command{
		Use:   "generate [template]",
		Short: "Generate a Docker Compose file from a pre-defined template",
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			templateName := args[0]
			var tpl *compose.Template
			for _, t := range compose.Templates {
				if t.Name == templateName {
					tpl = &t
					break
				}
			}

			if tpl == nil {
				var validNames []string
				for _, t := range compose.Templates {
					validNames = append(validNames, t.Name)
				}
				log.Fatalf("❌ Template %q not found. Available templates: %s", templateName, strings.Join(validNames, ", "))
			}

			fmt.Printf("🏗️  Generating %s template...\n", tpl.Name)
			fmt.Println("Please provide values for the following parameters:")

			params := make(map[string]string)
			reader := bufio.NewReader(os.Stdin)

			for paramName, desc := range tpl.Parameters {
				fmt.Printf("👉 %s (%s): ", paramName, desc)
				val, _ := reader.ReadString('\n')
				val = strings.TrimSpace(val)
				params[paramName] = val
			}

			yamlData, err := compose.Generate(templateName, params)
			if err != nil {
				log.Fatalf("❌ Failed to generate template: %v", err)
			}

			outputPath, _ := cmd.Flags().GetString("out")
			if outputPath != "" {
				if err := os.WriteFile(outputPath, yamlData, 0644); err != nil {
					log.Fatalf("❌ Failed to write generated file: %v", err)
				}
				fmt.Printf("✅ Template generated and saved to: %s\n", outputPath)
			} else {
				fmt.Println("\n📝 Generated Compose YAML:")
				fmt.Println(string(yamlData))
			}
		},
	}

	composeGenerateCmd.Flags().StringP("out", "o", "", "Output file path to save the generated compose configuration")

	composeCmd.AddCommand(composeLintCmd, composeFixCmd, composeGenerateCmd)

	// ─── Plugin Commands (Supervised via pkg/plugins.Manager) ───
	var pluginCmd = &cobra.Command{
		Use:   "plugin",
		Short: "Manage M3TAL external plugins",
		Run: func(cmd *cobra.Command, args []string) {
			cmd.Help()
		},
	}

	var pluginListCmd = &cobra.Command{
		Use:   "list",
		Short: "List all discovered and running plugins",
		Run: cmdutil.WithDaemon(func(c *client.Client, cmd *cobra.Command, args []string) {
			list, err := c.GetManagedPlugins()
			if err != nil {
				log.Fatalf("❌ Failed to load plugins via API: %v", err)
			}

			fmt.Printf("\n🔌 Managed Plugins (%d discovered)\n\n", len(list))
			if len(list) == 0 {
				fmt.Println("   No external plugins installed in search paths.")
				fmt.Println("   Plugin search paths: ~/.config/m3tal/plugins/, /var/lib/m3tal/plugins/, ./plugins/")
				fmt.Println()
				return
			}

			for _, p := range list {
				procName := p.Name
				if !strings.HasPrefix(procName, "m3tal-") {
					procName = "m3tal-" + procName
				}
				status := fmt.Sprintf("[%s]", p.State)
				caps := ""
				if len(p.Capabilities) > 0 {
					caps = fmt.Sprintf(" (caps: %s)", strings.Join(p.Capabilities, ", "))
				}
				fmt.Printf("   %-20s v%-8s %-12s proc:%-18s%s\n", p.Name, p.Version, status, procName, caps)
				if p.Description != "" {
					fmt.Printf("      %s\n", p.Description)
				}
				if p.LastError != "" {
					fmt.Printf("      ⚠️ Error: %s\n", p.LastError)
				}
			}
			fmt.Println()
		}),
	}

	var pluginStartCmd = &cobra.Command{
		Use:   "start [name]",
		Short: "Start an external plugin subprocess",
		Args:  cobra.ExactArgs(1),
		Run: cmdutil.WithDaemon(func(c *client.Client, cmd *cobra.Command, args []string) {
			name := args[0]
			fmt.Printf("🚀 Starting plugin %q...\n", name)
			if err := c.StartPlugin(name); err != nil {
				log.Fatalf("❌ Failed to start plugin: %v", err)
			}
			fmt.Printf("✅ Plugin %q started successfully.\n", name)
		}),
	}

	var pluginStopCmd = &cobra.Command{
		Use:   "stop [name]",
		Short: "Stop a running external plugin subprocess",
		Args:  cobra.ExactArgs(1),
		Run: cmdutil.WithDaemon(func(c *client.Client, cmd *cobra.Command, args []string) {
			name := args[0]
			fmt.Printf("⏹ Stopping plugin %q...\n", name)
			if err := c.StopPlugin(name); err != nil {
				log.Fatalf("❌ Failed to stop plugin: %v", err)
			}
			fmt.Printf("✅ Plugin %q stopped.\n", name)
		}),
	}

	var exportCatalogPath string
	var pluginCatalogCmd = &cobra.Command{
		Use:   "catalog",
		Short: "Display or export the M3TAL external plugin catalog",
		Run: func(cmd *cobra.Command, args []string) {
			catalog := api.GetDefaultCatalog()
			if exportCatalogPath != "" {
				if err := os.MkdirAll(filepath.Dir(exportCatalogPath), 0755); err != nil {
					log.Fatalf("❌ Failed to create directory: %v", err)
				}
				data, err := json.MarshalIndent(catalog, "", "  ")
				if err != nil {
					log.Fatalf("❌ Failed to marshal catalog: %v", err)
				}
				if err := os.WriteFile(exportCatalogPath, data, 0644); err != nil {
					log.Fatalf("❌ Failed to export catalog to %s: %v", exportCatalogPath, err)
				}
				fmt.Printf("✅ Plugin catalog exported to %s (%d items)\n", exportCatalogPath, len(catalog))
				return
			}

			fmt.Printf("\n📦 M3TAL Plugin Catalog (%d plugins)\n\n", len(catalog))
			for _, item := range catalog {
				fmt.Printf("   %-20s v%-8s [%s] proc:%-16s\n", item.Name, item.Version, item.Status, item.ProcessName)
				fmt.Printf("      %s\n", item.Description)
				if item.AptCommand != "" {
					fmt.Printf("      👉 %s\n", item.AptCommand)
				}
			}
			fmt.Println()
		},
	}
	pluginCatalogCmd.Flags().StringVar(&exportCatalogPath, "export", "", "Export catalog to JSON file path")

	pluginCmd.AddCommand(pluginListCmd, pluginStartCmd, pluginStopCmd, pluginCatalogCmd)

	// ── Doctor subcommands ─────────────────────────────────────────────────────

	// m3tal doctor scan containers
	var doctorScanContainersCmd = &cobra.Command{
		Use:   "containers",
		Short: "Scan container health states",
		Run: cmdutil.WithDaemon(func(c *client.Client, cmd *cobra.Command, args []string) {
			results, err := c.GetDoctorContainers()
			if err != nil {
				output.FatalError(err)
			}
			fmt.Printf("\n📦 Container Health Scan (%d containers)\n", len(results))
			fmt.Println(strings.Repeat("─", 60))
			for _, r := range results {
				fmt.Printf("  %s\n", output.ContainerSummaryLine(r))
				if r.Recommendation != "" {
					fmt.Printf("     💡 %s\n", r.Recommendation)
				}
			}
			fmt.Println()
		}),
	}

	// m3tal doctor scan mounts
	var doctorScanMountsCmd = &cobra.Command{
		Use:   "mounts",
		Short: "Validate container volume and bind-mount paths",
		Run: cmdutil.WithDaemon(func(c *client.Client, cmd *cobra.Command, args []string) {
			results, err := c.GetDoctorMounts()
			if err != nil {
				output.FatalError(err)
			}
			if len(results) == 0 {
				fmt.Println("✅ No mounts found.")
				return
			}
			fmt.Printf("\n📂 Mount Validation (%d mount(s))\n", len(results))
			fmt.Println(strings.Repeat("─", 60))
			for _, r := range results {
				if r.Severity != models.SeverityPass {
					fmt.Printf("  %s\n", output.MountSummaryLine(r))
					if r.Fix != "" {
						fmt.Printf("     💡 %s\n", r.Fix)
					}
				}
			}
			ok := 0
			for _, r := range results {
				if r.Severity == models.SeverityPass {
					ok++
				}
			}
			if ok > 0 {
				fmt.Printf("  ✅ %d mount(s) OK\n", ok)
			}
			fmt.Println()
		}),
	}

	// m3tal doctor scan ports
	var doctorScanPortsCmd = &cobra.Command{
		Use:   "ports",
		Short: "Detect port conflicts on declared M3TAL ports",
		Run: cmdutil.WithDaemon(func(c *client.Client, cmd *cobra.Command, args []string) {
			results, err := c.GetDoctorPorts()
			if err != nil {
				output.FatalError(err)
			}
			fmt.Printf("\n🔌 Port Conflict Scan (%d ports checked)\n", len(results))
			fmt.Println(strings.Repeat("─", 60))
			for _, r := range results {
				fmt.Printf("  %s\n", output.PortSummaryLine(r))
			}
			fmt.Println()
		}),
	}

	// m3tal doctor scan  (runs all scanners)
	var doctorScanCmd = &cobra.Command{
		Use:   "scan",
		Short: "Run all doctor scanners (containers, mounts, ports)",
		Run: func(cmd *cobra.Command, args []string) {
			doctorScanContainersCmd.Run(cmd, args)
			doctorScanMountsCmd.Run(cmd, args)
			doctorScanPortsCmd.Run(cmd, args)
		},
	}
	doctorScanCmd.AddCommand(doctorScanContainersCmd, doctorScanMountsCmd, doctorScanPortsCmd)

	// m3tal doctor fix
	var doctorFixApply bool
	var doctorFixName string
	var doctorFixCmd = &cobra.Command{
		Use:   "fix",
		Short: "Preview (or apply) automated fixes for detected issues",
		Long: `Scans containers, mounts, and ports for issues, then proposes
fixes. By default runs in dry-run mode — pass --apply to execute the fixes.`,
		Run: cmdutil.WithDaemon(func(c *client.Client, cmd *cobra.Command, args []string) {
			res, err := c.HandleDoctorFix(doctorFixApply, doctorFixName)
			if err != nil {
				output.FatalError(err)
			}

			appliedVal, _ := res["applied"].(bool)
			if !appliedVal {
				fixesData, _ := json.Marshal(res["fixes"])
				var fixes []models.Fix
				_ = json.Unmarshal(fixesData, &fixes)
				output.PrintFixes(fixes)
			} else {
				resultsData, _ := json.Marshal(res["results"])
				var results []models.FixResult
				_ = json.Unmarshal(resultsData, &results)
				output.PrintFixResults(results)
			}
		}),
	}
	doctorFixCmd.Flags().BoolVar(&doctorFixApply, "apply", false, "Apply fixes instead of previewing")
	doctorFixCmd.Flags().StringVar(&doctorFixName, "name", "", "Restrict container fixes to this container name")

	// m3tal doctor report
	var doctorReportJSON bool
	var doctorReportOut string
	var doctorReportCmd = &cobra.Command{
		Use:   "report",
		Short: "Generate a full system health report",
		Run: cmdutil.WithDaemon(func(c *client.Client, cmd *cobra.Command, args []string) {
			report, err := c.GetDoctorReport()
			if err != nil {
				output.FatalError(err)
			}

			if doctorReportJSON {
				data, err := json.MarshalIndent(report, "", "  ")
				if err != nil {
					fmt.Fprintf(os.Stderr, "error marshalling report: %v\n", err)
					return
				}
				fmt.Println(string(data))
				return
			}
			if doctorReportOut != "" {
				data, err := json.MarshalIndent(report, "", "  ")
				if err != nil {
					fmt.Fprintf(os.Stderr, "error marshalling report: %v\n", err)
					return
				}
				if err := os.WriteFile(doctorReportOut, data, 0644); err != nil {
					fmt.Fprintf(os.Stderr, "❌ Failed to write report: %v\n", err)
					os.Exit(1)
				}
				fmt.Printf("✅ Report written to %s\n", doctorReportOut)
				return
			}
			output.PrintReport(report)
		}),
	}
	doctorReportCmd.Flags().BoolVar(&doctorReportJSON, "json", false, "Output report as JSON")
	doctorReportCmd.Flags().StringVar(&doctorReportOut, "out", "", "Write JSON report to file path")

	docCmd.AddCommand(doctorScanCmd, doctorFixCmd, doctorReportCmd)

	var uiCmd = &cobra.Command{
		Use:   "ui",
		Short: "Launch M3TAL Desktop Application",
		Run: func(cmd *cobra.Command, args []string) {
			port, _ := cmd.Flags().GetString("port")
			headless, _ := cmd.Flags().GetBool("headless")
			launchDesktopApp(port, headless)
		},
	}
	uiCmd.Flags().String("port", "5050", "Port to serve on")
	uiCmd.Flags().Bool("headless", false, "Run in headless mode (do not launch desktop window)")

	var headlessCmd = &cobra.Command{
		Use:     "headless [port]",
		Aliases: []string{"server", "web"},
		Short:   "Start M3TAL in headless mode (ideal for home servers)",
		Run: func(cmd *cobra.Command, args []string) {
			port, _ := cmd.Flags().GetString("port")
			if len(args) > 0 && args[0] != "" {
				port = args[0]
			}
			if !cmd.Flags().Changed("port") && len(args) == 0 {
				if ep := os.Getenv("API_PORT"); ep != "" {
					port = ep
				} else if ep := os.Getenv("PORT"); ep != "" {
					port = ep
				}
			}
			launchDesktopApp(port, true)
		},
	}
	var versionCmd = &cobra.Command{
		Use:   "version",
		Short: "Print the M3TAL version",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Printf("M3TAL Core v%s\n", Version)
		},
	}

	var sysmonCmd = &cobra.Command{
		Use:     "sysmon",
		Aliases: []string{"monitor", "metrics-collector"},
		Short:   "Start M3TAL System Monitor telemetry collector daemon",
		Run: cmdutil.WithDaemon(func(c *client.Client, cmd *cobra.Command, args []string) {
			system.SetProcessName("m3tal-sysmon")
			unlock, acquired := system.AcquireProcessLock("m3tal-sysmon")
			if !acquired {
				return
			}
			defer unlock()

			fmt.Println("🚀 M3TAL System Monitor telemetry collector running (m3tal-sysmon)...")
			ticker := time.NewTicker(3 * time.Second)
			defer ticker.Stop()
			for range ticker.C {
				_, _ = c.GetTrayStats()
			}
		}),
	}

	var dockerDaemonCmd = &cobra.Command{
		Use:     "docker-daemon",
		Aliases: []string{"docker-mon", "docker-worker"},
		Short:   "Start M3TAL Docker monitoring and reconciliation daemon",
		Run: func(cmd *cobra.Command, args []string) {
			system.SetProcessName("m3tal-docker")
			unlock, acquired := system.AcquireProcessLock("m3tal-docker")
			if !acquired {
				return
			}
			defer unlock()

			fmt.Println("🐳 M3TAL Docker Worker running (m3tal-docker)...")
			api.RunAgentsDaemon()
		},
	}

	rootCmd.AddCommand(versionCmd, listCmd, psCmd, startCmd, stopCmd, statsCmd, daemonCmd, apiCmd, upCmd, downCmd, restartCmd, logsCmd, pullCmd, dashpassCmd, initCmd, docCmd, configCmd, pluginCmd, composeCmd, vpnCmd, initProxyCmds(), initDashCmd(), trayCmd, stacksCmd, uiCmd, headlessCmd, sysmonCmd, dockerDaemonCmd, initStackCmd(), initFixCmd(), initDeployCmd(), initReconcileCmd(), initAgentCmd())

	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func getLocalLANIP() string {
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		return "127.0.0.1"
	}
	for _, addr := range addrs {
		if ipnet, ok := addr.(*net.IPNet); ok && !ipnet.IP.IsLoopback() {
			if ip4 := ipnet.IP.To4(); ip4 != nil {
				ipStr := ip4.String()
				if strings.HasPrefix(ipStr, "192.168.") || strings.HasPrefix(ipStr, "10.") {
					return ipStr
				}
			}
		}
	}
	for _, addr := range addrs {
		if ipnet, ok := addr.(*net.IPNet); ok && !ipnet.IP.IsLoopback() {
			if ip4 := ipnet.IP.To4(); ip4 != nil {
				return ip4.String()
			}
		}
	}
	return "127.0.0.1"
}

// openDesktopApp launches M3TAL in a standalone application window (no browser address bar, no tabs).
// It attempts to use Chrome/Chromium --app mode for a borderless native-looking window with --class=m3tal,
// falling back to openBrowser if no standalone runtime is detected.
func openDesktopApp(url string) *exec.Cmd {
	if runtime.GOOS == "linux" {
		for _, browser := range []string{"google-chrome", "chromium", "chromium-browser", "brave-browser"} {
			if _, err := exec.LookPath(browser); err == nil {
				configDir := filepath.Join(os.Getenv("HOME"), ".config", "m3tal", "browser-profile")
				_ = os.MkdirAll(configDir, 0755)
				cmd := exec.Command(browser,
					"--app="+url,
					"--window-size=1280,850",
					"--class=m3tal",
					"--user-data-dir="+configDir,
					"--no-first-run",
					"--no-default-browser-check",
				)
				if err := cmd.Start(); err == nil {
					return cmd
				}
			}
		}
	}
	openBrowser(url)
	return nil
}

func launchDesktopApp(port string, forceHeadless bool) {
	if !forceHeadless && runtime.GOOS == "linux" && (os.Getenv("DISPLAY") != "" || os.Getenv("WAYLAND_DISPLAY") != "") {
		system.SetProcessName("m3tal-gui")
		unlock, acquired := system.AcquireProcessLock("m3tal-gui")
		if !acquired {
			fmt.Println("🟢 M3TAL Desktop App is already active.")
			return
		}
		defer unlock()
	}

	if port == "" {
		if ep := os.Getenv("API_PORT"); ep != "" {
			port = ep
		} else if ep := os.Getenv("PORT"); ep != "" {
			port = ep
		} else {
			port = "5050"
		}
	}
	addr := fmt.Sprintf("127.0.0.1:%s", port)

	serverStartedByUs := false
	conn, err := net.DialTimeout("tcp", addr, 250*time.Millisecond)
	if err == nil {
		conn.Close()
		fmt.Printf("🟢 M3TAL Desktop backend is already active on port %s.\n", port)
	} else {
		fmt.Printf("🚀 Starting M3TAL Desktop backend on port %s...\n", port)

		token := os.Getenv("API_TOKEN")
		if token == "" {
			token = "m3tal-secret-token"
		}

		go func() {
			if err := api.RunServer(port, token); err != nil {
				log.Fatalf("❌ Backend server failed: %v", err)
			}
		}()
		serverStartedByUs = true

		// Wait for server to bind
		for i := 0; i < 15; i++ {
			time.Sleep(150 * time.Millisecond)
			if c, err := net.DialTimeout("tcp", addr, 100*time.Millisecond); err == nil {
				c.Close()
				break
			}
		}
	}

	lanIP := getLocalLANIP()
	localURL := fmt.Sprintf("http://localhost:%s/gui/", port)
	networkURL := fmt.Sprintf("http://%s:%s/gui/", lanIP, port)

	fmt.Println()
	fmt.Println("╔════════════════════════════════════════════════════════════╗")
	fmt.Println("║               M3TAL Desktop Application Active             ║")
	fmt.Println("║                                                            ║")
	fmt.Printf("║  Local:   %-48s ║\n", localURL)
	if lanIP != "127.0.0.1" {
		fmt.Printf("║  Network: %-48s ║\n", networkURL)
	}
	fmt.Println("║                                                            ║")
	fmt.Println("╚════════════════════════════════════════════════════════════╝")
	fmt.Println()

	token := config.GetAPIToken()
	if token == "" {
		token = os.Getenv("API_TOKEN")
		if token == "" {
			token = "change_me_api_token"
		}
	}
	appURL := fmt.Sprintf("http://localhost:%s/gui/?token=%s", port, token)

	isHeadless := forceHeadless || (runtime.GOOS == "linux" && os.Getenv("DISPLAY") == "" && os.Getenv("WAYLAND_DISPLAY") == "")

	var browserCmd *exec.Cmd
	if !isHeadless {
		fmt.Printf("🖥️  Opening M3TAL Desktop App Window: %s\n", localURL)
		browserCmd = openDesktopApp(appURL)
	} else {
		fmt.Println("🌐 Headless mode active. Access the Web GUI remotely using the Network URL above.")
	}

	// Always spawn dedicated background telemetry process (m3tal-sysmon)
	exe, err := os.Executable()
	if err != nil {
		exe = "m3tal"
	}
	sysmonCmd := exec.Command(exe, "sysmon")
	_ = sysmonCmd.Start()

	cleanup := func() {
		if sysmonCmd != nil && sysmonCmd.Process != nil {
			_ = sysmonCmd.Process.Kill()
		}
	}

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	if browserCmd != nil {
		fmt.Println("⚡ M3TAL Desktop is running (m3tal-gui). Close the window or press Ctrl+C to exit...")
		done := make(chan error, 1)
		go func() {
			done <- browserCmd.Wait()
		}()
		select {
		case <-sigChan:
			fmt.Println("\nStopping M3TAL Desktop...")
			if browserCmd.Process != nil {
				_ = browserCmd.Process.Kill()
			}
		case <-done:
			fmt.Println("\nM3TAL Desktop window closed.")
		}
		cleanup()
		return
	}

	if serverStartedByUs || isHeadless {
		fmt.Println("⚡ Press Ctrl+C to exit M3TAL...")
		<-sigChan
		fmt.Println("\nStopping M3TAL Desktop backend...")
		cleanup()
	}
}

func parseComposeVariables(composeFile string) map[string]string {
	data, err := os.ReadFile(composeFile)
	if err != nil {
		return nil
	}

	// Regex matches ${VAR} or ${VAR:-default} or ${VAR-default}
	re := regexp.MustCompile(`\$\{([A-Z0-9_]+)(?::?-([^}]+))?\}`)
	matches := re.FindAllStringSubmatch(string(data), -1)

	vars := make(map[string]string)
	for _, match := range matches {
		if len(match) > 1 {
			key := match[1]
			val := ""
			if len(match) > 2 {
				val = match[2]
			}
			if existing, ok := vars[key]; !ok || (existing == "" && val != "") {
				vars[key] = val
			}
		}
	}
	return vars
}

func runWizard(targetFile string, composeFile string, update bool, isGlobal bool) {
	fmt.Printf("🛠️  M3TAL Configuration Wizard (%s)\n", filepath.Base(targetFile))

	if isGlobal && runtime.GOOS == "linux" && os.Geteuid() == 0 {
		configDir := filepath.Dir(system.ConfigPath)
		_ = os.MkdirAll(configDir, 0755)
		_ = os.MkdirAll(system.DataPath, 0755)
		fmt.Printf("✅ System directories initialized (%s, %s)\n", configDir, system.DataPath)
	}

	_ = os.MkdirAll("./data", 0755)

	var existingData []byte
	var realTargetFile = targetFile
	isSymlink := false

	if _, err := os.Lstat(targetFile); err == nil {
		info, _ := os.Lstat(targetFile)
		if info.Mode()&os.ModeSymlink != 0 {
			isSymlink = true
			linkTarget, err := os.Readlink(targetFile)
			if err == nil {
				if !filepath.IsAbs(linkTarget) {
					linkTarget = filepath.Join(filepath.Dir(targetFile), linkTarget)
				}
				realTargetFile = linkTarget
				fmt.Printf("👉 Target %s is a symlink pointing to %s. Keeping symlink and updating the shared configuration.\n", filepath.Base(targetFile), filepath.Base(linkTarget))
			}
		}
	}

	if update {
		if _, err := os.Stat(realTargetFile); err == nil {
			existingData, _ = os.ReadFile(realTargetFile)
		}
	}

	existingVars := make(map[string]string)
	if existingData != nil {
		lines := strings.Split(string(existingData), "\n")
		for _, line := range lines {
			if strings.Contains(line, "=") && !strings.HasPrefix(line, "#") {
				parts := strings.SplitN(line, "=", 2)
				existingVars[parts[0]] = parts[1]
			}
		}
	}

	requiredVars := make(map[string]string)
	var orderedKeys []string

	if isGlobal {
		lines := strings.Split(string(envExample), "\n")
		for _, line := range lines {
			if strings.Contains(line, "=") && !strings.HasPrefix(line, "#") {
				parts := strings.SplitN(line, "=", 2)
				requiredVars[parts[0]] = parts[1]
				orderedKeys = append(orderedKeys, parts[0])
			}
		}
	} else if composeFile != "" {
		parsedVars := parseComposeVariables(composeFile)
		for k, v := range parsedVars {
			requiredVars[k] = v
			orderedKeys = append(orderedKeys, k)
		}
	}

	if len(orderedKeys) == 0 {
		log.Fatalf("❌ Missing configuration source or no variables found.")
	}

	var finalLines []string
	for _, key := range orderedKeys {
		defaultVal := requiredVars[key]

		val := defaultVal
		if existing, ok := existingVars[key]; ok {
			val = existing
		}

		if isGlobal && (key == "DASHBOARD_SECRET" || key == "API_TOKEN") && !update && val == "" {
			newSecret := generateSecret()
			fmt.Printf("[Auto] %s generated: %s\n", key, newSecret)
			finalLines = append(finalLines, key+"="+newSecret)
			continue
		}

		colorReset := "\033[0m"
		colorRed := "\033[31m"

		promptStr := fmt.Sprintf("%s [%s]: ", key, val)
		if val == "" {
			fmt.Printf("%s%s%s", colorRed, promptStr, colorReset)
		} else {
			fmt.Printf("%s", promptStr)
		}

		var input string
		fmt.Scanln(&input)
		if input != "" {
			finalLines = append(finalLines, key+"="+input)
		} else {
			finalLines = append(finalLines, key+"="+val)
		}
	}

	if isSymlink {
		sharedContentBytes, err := os.ReadFile(realTargetFile)
		var sharedContent string
		if err == nil {
			sharedContent = string(sharedContentBytes)
		}

		for _, line := range finalLines {
			if strings.Contains(line, "=") {
				parts := strings.SplitN(line, "=", 2)
				key := parts[0]
				val := parts[1]

				if strings.Contains(sharedContent, key+"=") {
					sharedContent = replaceSecret(sharedContent, key+"=", val)
				} else {
					sharedContent = strings.TrimSuffix(sharedContent, "\n") + fmt.Sprintf("\n%s=%s\n", key, val)
				}
			}
		}

		if err := os.WriteFile(realTargetFile, []byte(sharedContent), 0600); err != nil {
			log.Fatal(err)
		}
		fmt.Printf("\n✅ Shared configuration updated inline: %s\n", realTargetFile)
	} else {
		content := strings.Join(finalLines, "\n")
		if err := os.WriteFile(targetFile, []byte(content), 0600); err != nil {
			log.Fatal(err)
		}
		fmt.Printf("\n✅ Configuration saved to %s\n", targetFile)
	}

	// Synchronize ADMIN_PASSWORD to users.json if set in the global wizard
	if isGlobal {
		adminPass := ""
		for _, line := range finalLines {
			if strings.HasPrefix(line, "ADMIN_PASSWORD=") {
				adminPass = strings.TrimPrefix(line, "ADMIN_PASSWORD=")
				break
			}
		}
		if adminPass != "" {
			// Best-effort: try to update via API if daemon is running
			c := client.NewClient(config.GetAPIURL(), config.GetAPIToken())
			_ = c.UpdateDashpass("admin", adminPass)
		}
	}

	if isGlobal {
		fmt.Println("👉 You may also want to symlink this to .env for local stack commands.")
	}
}

func printJSON(v interface{}) {
	data, _ := json.MarshalIndent(v, "", "  ")
	fmt.Println(string(data))
}

func getEnvValue(content, key string) string {
	lines := strings.Split(content, "\n")
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, key+"=") && !strings.HasPrefix(trimmed, "#") {
			parts := strings.SplitN(trimmed, "=", 2)
			if len(parts) == 2 {
				return parts[1]
			}
		}
	}
	return ""
}

func generateSecret() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

func replaceSecret(content, key, secret string) string {
	lines := strings.Split(content, "\n")
	for i, line := range lines {
		if strings.HasPrefix(line, key) {
			lines[i] = key + secret
		}
	}
	return strings.Join(lines, "\n")
}

func localValidateStoragePath(basePath string) error {
	if basePath == "" {
		return fmt.Errorf("BASE_STORAGE_PATH is empty. Set it in your .env file")
	}

	absPath, err := filepath.Abs(basePath)
	if err != nil {
		return fmt.Errorf("cannot resolve path '%s': %w", basePath, err)
	}

	info, err := os.Stat(absPath)
	if os.IsNotExist(err) {
		return fmt.Errorf("directory does not exist: %s\nCreate it with: mkdir -p %s", absPath, absPath)
	}
	if err != nil {
		return fmt.Errorf("cannot access '%s': %w", absPath, err)
	}
	if !info.IsDir() {
		return fmt.Errorf("path exists but is not a directory: %s", absPath)
	}

	// Test writability
	testFile := filepath.Join(absPath, ".m3tal-write-test")
	f, err := os.Create(testFile)
	if err != nil {
		return fmt.Errorf("path '%s' is NOT writable: %w", absPath, err)
	}
	f.Close()
	os.Remove(testFile)

	return nil
}
