package main

import (
	"fmt"
	"github.com/vaultdrift/vaultdrift/internal/auth"
)

func main() {
	// The stored hash from seed.go
	storedHash := "$argon2id$v=19$m=65536,t=3,p=1$6IOdsT/ZB3Yc39wdhST32A$0psObWN9M9SCF1zLWMsE9lef9JFfK2pqsY1o2M3x2zg"
	password := "admin"

	valid, err := auth.VerifyPassword(password, storedHash)
	if err != nil {
		fmt.Println("Verify error:", err)
		return
	}

	if valid {
		fmt.Println("✓ Password matches!")
	} else {
		fmt.Println("✗ Password does not match!")
	}
}
