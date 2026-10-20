package config

import (
	"bufio"
	"os"
	"strings"
)

// MergeEnv merges a global environment file and a stack-specific env file into a new file.
// The function preserves the order of variables from the stack file by default, appending any
// variables from the global file that are not already defined in the stack file. If a variable
// exists in both files, the stack file's value is kept.
//
// The function writes the merged content to destPath. It creates destPath if it does not
// exist, overwriting if it does. The merge preserves comments and blank lines from both
// source files.
//
// Return an error if any input file cannot be opened or read.
//
// Example:
//
//	err := MergeEnv("/etc/m3tal/config.yaml", "/run/m3tal/media.env", "/tmp/media.env")
func MergeEnv(globalPath, stackPath, destPath string) error {
	// Read global file
	globalVars := make(map[string]struct{})
	gFile, err := os.Open(globalPath)
	if err != nil {
		return err
	}
	defer gFile.Close()
	scanner := bufio.NewScanner(gFile)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") || !strings.Contains(line, "=") {
			continue
		}
		key := strings.SplitN(line, "=", 2)[0]
		globalVars[key] = struct{}{}
	}
	if err := scanner.Err(); err != nil {
		return err
	}

	// Open stack file for reading and dest file for writing
	stackFile, err := os.Open(stackPath)
	if err != nil {
		return err
	}
	defer stackFile.Close()
	out, err := os.Create(destPath)
	if err != nil {
		return err
	}
	defer out.Close()
	outWriter := bufio.NewWriter(out)

	scanner = bufio.NewScanner(stackFile)
	stackVarsSeen := make(map[string]struct{})
	for scanner.Scan() {
		line := scanner.Text()
		trimmed := strings.TrimSpace(line)
		if trimmed != "" && !strings.HasPrefix(trimmed, "#") && strings.Contains(trimmed, "=") {
			key := strings.SplitN(trimmed, "=", 2)[0]
			stackVarsSeen[key] = struct{}{}
		}
		outWriter.WriteString(line + "\n")
	}
	if err := scanner.Err(); err != nil {
		return err
	}

	// Append missing global variables
	for key := range globalVars {
		if _, ok := stackVarsSeen[key]; !ok {
			// Lookup value from global file
			value, err := getEnvValueFromFile(globalPath, key)
			if err != nil {
				continue
			}
			outWriter.WriteString(key + "=" + value + "\n")
		}
	}
	outWriter.Flush()
	return nil
}

// getEnvValueFromFile reads a single variable value from a file. It assumes the file
// contains key=value lines. Leading/trailing whitespace around the key and value are
// trimmed. Returns an empty string if the key is not found.
func getEnvValueFromFile(filePath, key string) (string, error) {
	f, err := os.Open(filePath)
	if err != nil {
		return "", err
	}
	defer f.Close()
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") || !strings.Contains(line, "=") {
			continue
		}
		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			continue
		}
		k := strings.TrimSpace(parts[0])
		v := strings.TrimSpace(parts[1])
		if k == key {
			return v, nil
		}
	}
	return "", nil
}

// The following helper is used to near match the design of the current command logic
// which calls config.MergeEnv from orchestrator or cmd.  The actual integration into
// the main m3tal command is handled elsewhere; this file simply provides the core merge
// functionality and export.
