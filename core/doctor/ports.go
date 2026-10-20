package doctor

import (
	"bufio"
	"context"
	"encoding/hex"
	"fmt"
	"github.com/jakej985-rgb/m3tal-core/pkg/models"
	"net"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"
)

type PortResult = models.PortResult
type Severity = models.Severity

// DefaultDeclaredPorts is the set of ports M3TAL reserves by default.
var DefaultDeclaredPorts = []int{80, 443, 8080, 8443, 3000, 9000}

// ScanPortConflicts checks the declared ports for conflicts by reading
// /proc/net/tcp and /proc/net/tcp6 (Linux) and falling back to `ss -tlnp`.
func ScanPortConflicts(declaredPorts []int) ([]PortResult, error) {
	if len(declaredPorts) == 0 {
		declaredPorts = DefaultDeclaredPorts
	}

	listening, err := getListeningPorts()
	if err != nil {
		// Fallback: try ss
		listening, err = getListeningPortsSS()
		if err != nil {
			return nil, fmt.Errorf("cannot determine listening ports: %w", err)
		}
	}

	var results []PortResult
	for _, port := range declaredPorts {
		entry, inUse := listening[port]
		r := PortResult{
			Port:  port,
			InUse: inUse,
		}
		if inUse {
			r.Conflict = true
			r.OwnedBy = entry.processName
			r.PID = entry.pid
			r.Severity = SeverityFail
			r.Suggestion = nextFreePort(port+1, listening)
			r.Note = fmt.Sprintf("Port %d is occupied by %q (PID %d). Consider using port %d instead.",
				port, entry.processName, entry.pid, r.Suggestion)
		} else {
			r.Severity = SeverityPass
		}
		results = append(results, r)
	}
	return results, nil
}

// procEntry holds info parsed from /proc/net/tcp.
type procEntry struct {
	port        int
	pid         int
	processName string
}

// getListeningPorts reads /proc/net/tcp and /proc/net/tcp6 to build a map of
// port → procEntry for all LISTEN sockets.
func getListeningPorts() (map[int]procEntry, error) {
	result := make(map[int]procEntry)

	for _, path := range []string{"/proc/net/tcp", "/proc/net/tcp6"} {
		if err := parseProcNetTCP(path, result); err != nil {
			// One may not exist (e.g. no IPv6)
			continue
		}
	}

	if len(result) == 0 {
		return nil, fmt.Errorf("no data from /proc/net/tcp")
	}

	// Enrich with process names via /proc/<pid>/comm
	for port, e := range result {
		if e.pid > 0 {
			commPath := fmt.Sprintf("/proc/%d/comm", e.pid)
			if data, err := os.ReadFile(commPath); err == nil {
				e.processName = strings.TrimSpace(string(data))
				result[port] = e
			}
		}
	}

	return result, nil
}

// parseProcNetTCP reads a single /proc/net/tcp* file.
// Format: sl  local_address rem_address st tx_queue rx_queue ...
// st=0A means LISTEN.
func parseProcNetTCP(path string, out map[int]procEntry) error {
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	scanner.Scan() // skip header
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		fields := strings.Fields(line)
		if len(fields) < 10 {
			continue
		}
		state := fields[3]
		if state != "0A" { // 0A = TCP_LISTEN
			continue
		}
		localAddr := fields[1]
		port, err := parseHexPort(localAddr)
		if err != nil {
			continue
		}
		inode := fields[9]
		pid := findPIDForInode(inode)
		out[port] = procEntry{port: port, pid: pid}
	}
	return scanner.Err()
}

// parseHexPort extracts the port from an address like "00000000:1F90".
func parseHexPort(addr string) (int, error) {
	parts := strings.Split(addr, ":")
	if len(parts) != 2 {
		return 0, fmt.Errorf("bad addr: %s", addr)
	}
	portHex, err := hex.DecodeString(parts[1])
	if err != nil || len(portHex) != 2 {
		return 0, fmt.Errorf("bad port hex: %s", parts[1])
	}
	return int(portHex[0])<<8 | int(portHex[1]), nil
}

// findPIDForInode scans /proc/*/fd/* to match a socket inode to a PID.
// This is best-effort; returns 0 on failure.
func findPIDForInode(inode string) int {
	target := fmt.Sprintf("socket:[%s]", inode)

	entries, err := os.ReadDir("/proc")
	if err != nil {
		return 0
	}
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		pid, err := strconv.Atoi(e.Name())
		if err != nil {
			continue
		}
		fdDir := fmt.Sprintf("/proc/%d/fd", pid)
		fds, err := os.ReadDir(fdDir)
		if err != nil {
			continue
		}
		for _, fd := range fds {
			link, err := os.Readlink(fmt.Sprintf("%s/%s", fdDir, fd.Name()))
			if err != nil {
				continue
			}
			if link == target {
				return pid
			}
		}
	}
	return 0
}

// getListeningPortsSS is a fallback using `ss -tlnp`.
func getListeningPortsSS() (map[int]procEntry, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	out, err := exec.CommandContext(ctx, "ss", "-tlnp").Output()
	if err != nil {
		return nil, fmt.Errorf("ss failed: %w", err)
	}

	result := make(map[int]procEntry)
	scanner := bufio.NewScanner(strings.NewReader(string(out)))
	scanner.Scan() // skip header
	for scanner.Scan() {
		line := scanner.Text()
		fields := strings.Fields(line)
		if len(fields) < 5 {
			continue
		}
		// Local address is field index 3: *:80 or 0.0.0.0:80
		local := fields[3]
		colonIdx := strings.LastIndex(local, ":")
		if colonIdx < 0 {
			continue
		}
		port, err := strconv.Atoi(local[colonIdx+1:])
		if err != nil {
			continue
		}
		entry := procEntry{port: port}
		// Try to extract process name from the last field: users:(("nginx",pid=123,fd=6))
		if len(fields) >= 6 {
			procField := fields[len(fields)-1]
			if strings.HasPrefix(procField, "users:") {
				entry.processName = extractSSProcessName(procField)
				entry.pid = extractSSPID(procField)
			}
		}
		result[port] = entry
	}
	return result, nil
}

func extractSSProcessName(s string) string {
	// users:(("nginx",pid=123,fd=6))
	start := strings.Index(s, "((\"")
	if start < 0 {
		return ""
	}
	s = s[start+3:]
	end := strings.Index(s, "\"")
	if end < 0 {
		return s
	}
	return s[:end]
}

func extractSSPID(s string) int {
	// pid=123
	idx := strings.Index(s, "pid=")
	if idx < 0 {
		return 0
	}
	s = s[idx+4:]
	end := strings.IndexAny(s, ",)")
	if end > 0 {
		s = s[:end]
	}
	pid, _ := strconv.Atoi(s)
	return pid
}

// nextFreePort returns the next port >= start that is not in the listening map.
func nextFreePort(start int, listening map[int]procEntry) int {
	for p := start; p < 65535; p++ {
		if _, used := listening[p]; used {
			continue
		}
		// Double-check with a quick TCP dial
		addr := fmt.Sprintf("localhost:%d", p)
		conn, err := net.DialTimeout("tcp", addr, 100*time.Millisecond)
		if err != nil {
			return p
		}
		conn.Close()
	}
	return 0
}
