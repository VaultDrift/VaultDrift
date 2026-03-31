package db

import (
	"context"
	"testing"
)

// TestUserOperations tests core user database operations
func TestUserOperations(t *testing.T) {
	// Create in-memory database for testing
	mgr, err := Open(Config{Path: ":memory:"})
	if err != nil {
		t.Fatalf("Failed to create database: %v", err)
	}
	defer mgr.Close()

	ctx := context.Background()

	t.Run("CreateAndGetUser", func(t *testing.T) {
		user := &User{
			Username:     "testuser",
			Email:        "test@example.com",
			PasswordHash: "hashedpassword123",
			Role:         "user",
			Status:       "active",
		}

		// Create user
		err := mgr.CreateUser(ctx, user)
		if err != nil {
			t.Fatalf("Failed to create user: %v", err)
		}

		if user.ID == "" {
			t.Error("User ID should be set after creation")
		}

		// Get user by ID
		retrieved, err := mgr.GetUserByID(ctx, user.ID)
		if err != nil {
			t.Fatalf("Failed to get user: %v", err)
		}

		if retrieved.Username != user.Username {
			t.Errorf("Username mismatch: got %s, want %s", retrieved.Username, user.Username)
		}

		// Get user by email
		byEmail, err := mgr.GetUserByEmail(ctx, user.Email)
		if err != nil {
			t.Fatalf("Failed to get user by email: %v", err)
		}

		if byEmail.ID != user.ID {
			t.Error("User retrieved by email should have same ID")
		}

		// Get user by username
		byUsername, err := mgr.GetUserByUsername(ctx, user.Username)
		if err != nil {
			t.Fatalf("Failed to get user by username: %v", err)
		}

		if byUsername.ID != user.ID {
			t.Error("User retrieved by username should have same ID")
		}

		t.Logf("✅ User CRUD operations working")
	})

	t.Run("UserNotFound", func(t *testing.T) {
		_, err := mgr.GetUserByID(ctx, "non-existent-id")
		if err == nil {
			t.Error("Should return error for non-existent user")
		}
	})

	t.Run("UpdateUser", func(t *testing.T) {
		user := &User{
			Username:     "updateuser",
			Email:        "update@example.com",
			PasswordHash: "oldhash",
			Role:         "user",
			Status:       "active",
		}
		mgr.CreateUser(ctx, user)

		// Update user
		updates := map[string]interface{}{
			"role": "admin",
		}
		err := mgr.UpdateUser(ctx, user.ID, updates)
		if err != nil {
			t.Fatalf("Failed to update user: %v", err)
		}

		// Verify update
		updated, _ := mgr.GetUserByID(ctx, user.ID)
		if updated.Role != "admin" {
			t.Errorf("Role should be updated to admin, got %s", updated.Role)
		}

		t.Logf("✅ User update working")
	})

	t.Run("ListUsers", func(t *testing.T) {
		// Create multiple users
		for i := 0; i < 3; i++ {
			user := &User{
				Username:     "listuser" + string(rune('0'+i)),
				Email:        "list" + string(rune('0'+i)) + "@example.com",
				PasswordHash: "hash",
				Role:         "user",
				Status:       "active",
			}
			mgr.CreateUser(ctx, user)
		}

		users, total, err := mgr.ListUsers(ctx, 0, 10, UserFilter{})
		if err != nil {
			t.Fatalf("Failed to list users: %v", err)
		}

		if len(users) < 3 {
			t.Errorf("Expected at least 3 users, got %d", len(users))
		}

		if total < 3 {
			t.Errorf("Expected total >= 3, got %d", total)
		}

		t.Logf("✅ List users working (%d users, %d total)", len(users), total)
	})

	t.Run("ListUsersWithFilter", func(t *testing.T) {
		// Test with role filter
		_, total, err := mgr.ListUsers(ctx, 0, 10, UserFilter{Role: "admin"})
		if err != nil {
			t.Fatalf("Failed to list users with filter: %v", err)
		}

		// Should have at least the admin we created
		if total < 1 {
			t.Errorf("Expected at least 1 admin, got %d", total)
		}

		t.Logf("✅ List users with filter working (%d admins)", total)
	})

	t.Run("DeleteUser", func(t *testing.T) {
		user := &User{
			Username:     "deleteuser",
			Email:        "delete@example.com",
			PasswordHash: "hash",
			Role:         "user",
			Status:       "active",
		}
		mgr.CreateUser(ctx, user)

		err := mgr.DeleteUser(ctx, user.ID)
		if err != nil {
			t.Fatalf("Failed to delete user: %v", err)
		}

		// Verify deletion
		_, err = mgr.GetUserByID(ctx, user.ID)
		if err == nil {
			t.Error("User should not exist after deletion")
		}

		t.Logf("✅ User delete working")
	})

	t.Run("UpdateUsedBytes", func(t *testing.T) {
		user := &User{
			Username:     "quotauser",
			Email:        "quota@example.com",
			PasswordHash: "hash",
			Role:         "user",
			Status:       "active",
		}
		mgr.CreateUser(ctx, user)

		// Update used bytes
		err := mgr.UpdateUsedBytes(ctx, user.ID, 1024)
		if err != nil {
			t.Fatalf("Failed to update used bytes: %v", err)
		}

		// Verify
		updated, _ := mgr.GetUserByID(ctx, user.ID)
		if updated.UsedBytes != 1024 {
			t.Errorf("UsedBytes should be 1024, got %d", updated.UsedBytes)
		}

		// Add more
		err = mgr.UpdateUsedBytes(ctx, user.ID, 512)
		if err != nil {
			t.Fatalf("Failed to update used bytes: %v", err)
		}

		updated, _ = mgr.GetUserByID(ctx, user.ID)
		if updated.UsedBytes != 1536 {
			t.Errorf("UsedBytes should be 1536, got %d", updated.UsedBytes)
		}

		t.Logf("✅ Update used bytes working")
	})

	t.Run("GetUserCount", func(t *testing.T) {
		count, err := mgr.GetUserCount(ctx)
		if err != nil {
			t.Fatalf("Failed to get user count: %v", err)
		}

		if count < 1 {
			t.Errorf("Expected at least 1 user, got %d", count)
		}

		t.Logf("✅ User count: %d", count)
	})

	t.Run("DuplicateUser", func(t *testing.T) {
		user := &User{
			Username:     "duplicateuser",
			Email:        "dup@example.com",
			PasswordHash: "hash",
			Role:         "user",
			Status:       "active",
		}
		mgr.CreateUser(ctx, user)

		// Try to create same user again
		err := mgr.CreateUser(ctx, user)
		if err == nil {
			t.Error("Should fail for duplicate user")
		}

		t.Logf("✅ Duplicate user detection working")
	})
}

