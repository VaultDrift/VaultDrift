package main

import (
	"context"
	"fmt"
	"log"

	"github.com/vaultdrift/vaultdrift/internal/db"
)

func main() {
	// Open the database
	database, err := db.Open(db.Config{Path: "./data/vaultdrift.db"})
	if err != nil {
		log.Fatal("Failed to open database:", err)
	}
	defer database.Close()

	ctx := context.Background()

	// Check roles
	var roleCount int
	err = database.QueryRow(ctx, "SELECT COUNT(*) FROM roles").Scan(&roleCount)
	if err != nil {
		log.Println("Error counting roles:", err)
	} else {
		fmt.Printf("Roles count: %d\n", roleCount)
	}

	// Check users
	var userCount int
	err = database.QueryRow(ctx, "SELECT COUNT(*) FROM users").Scan(&userCount)
	if err != nil {
		log.Println("Error counting users:", err)
	} else {
		fmt.Printf("Users count: %d\n", userCount)
	}

	// Try to get admin user
	user, err := database.GetUserByUsername(ctx, "admin")
	if err != nil {
		log.Println("Error getting admin user:", err)
	} else {
		fmt.Printf("Admin user found:\n")
		fmt.Printf("  ID: %s\n", user.ID)
		fmt.Printf("  Username: %s\n", user.Username)
		fmt.Printf("  Email: %s\n", user.Email)
		fmt.Printf("  Role: %s\n", user.Role)
		fmt.Printf("  Status: %s\n", user.Status)
		fmt.Printf("  PasswordHash: %s...\n", user.PasswordHash[:50])
	}
}
