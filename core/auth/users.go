package auth

import (
	"encoding/json"
	"os"
	"path/filepath"

	"golang.org/x/crypto/bcrypt"
)

// User represents a dashboard user
type User struct {
	Username string `json:"username"`
	Hash     string `json:"token_hash"`
	Role     string `json:"role"`
}

// UpdateUser updates or creates a user in the specified JSON file
func UpdateUser(filePath, username, password string) error {
	// Self-healing: if users.json exists as a directory (Docker bind-mount error), remove it
	if info, err := os.Stat(filePath); err == nil && info.IsDir() {
		_ = os.RemoveAll(filePath)
	}

	// Self-healing: if users.json is missing, initialize it from users.example.json if available
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		dir := filepath.Dir(filePath)
		examplePath := filepath.Join(dir, "users.example.json")
		if _, err := os.Stat(examplePath); err == nil {
			if data, err := os.ReadFile(examplePath); err == nil {
				_ = os.WriteFile(filePath, data, 0644)
			}
		}
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return err
	}

	var users []User
	data, err := os.ReadFile(filePath)
	if err == nil {
		json.Unmarshal(data, &users)
	}

	found := false
	for i, u := range users {
		if u.Username == username {
			users[i].Hash = string(hash)
			found = true
			break
		}
	}
	if !found {
		users = append(users, User{Username: username, Hash: string(hash), Role: "admin"})
	}

	newData, err := json.MarshalIndent(users, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(filePath, newData, 0644)
}
