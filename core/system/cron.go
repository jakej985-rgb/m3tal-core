package system

import (
	"bufio"
	"bytes"
	"os"
	"os/exec"
	"strings"

	"github.com/jakej985-rgb/m3tal-core/pkg/models"
)

// ListCronJobs retrieves all scheduled tasks from user/system cron files and systemd timers.
func ListCronJobs() ([]models.CronJob, error) {
	var jobs []models.CronJob

	// 1. Fetch current user's crontab
	cmd := exec.Command("crontab", "-l")
	var out bytes.Buffer
	cmd.Stdout = &out
	if err := cmd.Run(); err == nil {
		scanner := bufio.NewScanner(&out)
		for scanner.Scan() {
			line := strings.TrimSpace(scanner.Text())
			if line == "" || strings.HasPrefix(line, "#") {
				continue
			}
			// User crontabs: minute hour day-of-month month day-of-week command
			fields := strings.Fields(line)
			if len(fields) >= 6 {
				sched := strings.Join(fields[0:5], " ")
				cmdStr := strings.Join(fields[5:], " ")
				jobs = append(jobs, models.CronJob{
					Schedule: sched,
					Command:  cmdStr,
					User:     "current",
					Source:   "cron",
				})
			}
		}
	}

	// 2. Fetch /etc/crontab (system crontab)
	if file, err := os.Open("/etc/crontab"); err == nil {
		defer file.Close()
		scanner := bufio.NewScanner(file)
		for scanner.Scan() {
			line := strings.TrimSpace(scanner.Text())
			if line == "" || strings.HasPrefix(line, "#") || strings.Contains(line, "=") {
				continue
			}
			// System crontab: minute hour day-of-month month day-of-week user command
			fields := strings.Fields(line)
			if len(fields) >= 7 {
				sched := strings.Join(fields[0:5], " ")
				user := fields[5]
				cmdStr := strings.Join(fields[6:], " ")
				jobs = append(jobs, models.CronJob{
					Schedule: sched,
					Command:  cmdStr,
					User:     user,
					Source:   "cron",
				})
			}
		}
	}

	// 3. Fetch systemd timers
	cmdTimers := exec.Command("systemctl", "list-timers", "--no-legend", "--no-pager")
	var outTimers bytes.Buffer
	cmdTimers.Stdout = &outTimers
	if err := cmdTimers.Run(); err == nil {
		scanner := bufio.NewScanner(&outTimers)
		for scanner.Scan() {
			line := strings.TrimSpace(scanner.Text())
			if line == "" {
				continue
			}

			// Typical fields format (variable columns depending on MDT/timezone strings):
			// We can split columns, usually the last two fields are the TIMER name and the SERVICE name.
			fields := strings.Fields(line)
			if len(fields) >= 2 {
				timerUnit := fields[len(fields)-2]
				serviceUnit := fields[len(fields)-1]

				// Re-parse the timer configuration/details
				// Next execution can be estimated from the first columns (date & time & timezone)
				nextExec := "Systemd scheduled"
				if len(fields) >= 5 {
					// Join first few columns as scheduled timer info
					nextExec = strings.Join(fields[0:3], " ")
				}

				jobs = append(jobs, models.CronJob{
					Schedule: nextExec,
					Command:  "Trigger service: " + serviceUnit,
					User:     "root",
					Source:   timerUnit,
				})
			}
		}
	}

	return jobs, nil
}
