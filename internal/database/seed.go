package database

import (
	"log"

	"golang.org/x/crypto/bcrypt"
)

func SeedSuperAdmin() error {
	var count int
	err := DB.QueryRow("SELECT COUNT(*) FROM users WHERE role = 'super_admin'").Scan(&count)
	if err != nil {
		return err
	}

	if count > 0 {
		log.Println("ℹ️  Super Admin already exists, skipping seed")
		return nil
	}

	// Default credentials: admin / admin123 (CHANGE IN PRODUCTION!)
	hash, err := bcrypt.GenerateFromPassword([]byte("admin123"), bcrypt.DefaultCost)
	if err != nil {
		return err
	}

	_, err = DB.Exec(
		"INSERT INTO users (username, password_hash, role) VALUES (?, ?, ?)",
		"admin", string(hash), "super_admin",
	)
	if err != nil {
		return err
	}

	log.Println("✅ Super Admin seeded (username: admin, password: admin123)")
	return nil
}
