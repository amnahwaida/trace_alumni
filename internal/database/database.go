package database

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"path/filepath"

	_ "modernc.org/sqlite"
)

var DB *sql.DB

func InitDB(dataDir string) error {
	// Ensure data directory exists
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		return fmt.Errorf("failed to create data directory: %w", err)
	}

	// Ensure uploads directory exists
	uploadsDir := filepath.Join(dataDir, "uploads")
	if err := os.MkdirAll(uploadsDir, 0755); err != nil {
		return fmt.Errorf("failed to create uploads directory: %w", err)
	}

	dbPath := filepath.Join(dataDir, "alumni.db")
	var err error
	DB, err = sql.Open("sqlite", dbPath+"?_journal_mode=WAL&_busy_timeout=5000")
	if err != nil {
		return fmt.Errorf("failed to open database: %w", err)
	}

	// Enable WAL mode and foreign keys
	if _, err := DB.Exec("PRAGMA journal_mode=WAL"); err != nil {
		return fmt.Errorf("failed to set WAL mode: %w", err)
	}
	if _, err := DB.Exec("PRAGMA foreign_keys=ON"); err != nil {
		return fmt.Errorf("failed to enable foreign keys: %w", err)
	}

	// Set connection pool for SQLite (single writer)
	DB.SetMaxOpenConns(1)

	log.Println("✅ Database connected:", dbPath)
	return nil
}

func RunMigrations() error {
	migrations := []string{
		`CREATE TABLE IF NOT EXISTS users (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			username TEXT UNIQUE NOT NULL,
			password_hash TEXT NOT NULL,
			role TEXT CHECK(role IN ('super_admin', 'admin', 'staff')) NOT NULL,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE TABLE IF NOT EXISTS alumni (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			nama_lengkap TEXT NOT NULL,
			alamat_asli TEXT,
			domisili_sekarang TEXT,
			no_hp TEXT,
			email TEXT,
			tahun_lulus INTEGER NOT NULL,
			tanggal_lahir DATE,
			pekerjaan TEXT,
			instansi TEXT,
			url_linkedin TEXT,
			foto_profil TEXT,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE INDEX IF NOT EXISTS idx_alumni_search ON alumni(nama_lengkap, tahun_lulus)`,
		`CREATE TABLE IF NOT EXISTS info_papan (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			judul TEXT NOT NULL,
			deskripsi TEXT NOT NULL,
			link_eksternal TEXT,
			kategori TEXT DEFAULT 'umum' CHECK(kategori IN ('loker', 'beasiswa', 'reuni', 'umum')),
			dibuat_oleh INTEGER REFERENCES users(id),
			dibuat_pada DATETIME DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE TABLE IF NOT EXISTS sessions (
			id TEXT PRIMARY KEY,
			user_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
			expires_at DATETIME NOT NULL,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)`,
	}

	for _, m := range migrations {
		if _, err := DB.Exec(m); err != nil {
			return fmt.Errorf("migration failed: %w\nSQL: %s", err, m)
		}
	}

	// Dynamic schema updates for settings and info_papan status
	_, _ = DB.Exec(`CREATE TABLE IF NOT EXISTS settings (
		key TEXT PRIMARY KEY,
		value TEXT NOT NULL
	)`)
	_, _ = DB.Exec(`INSERT OR IGNORE INTO settings (key, value) VALUES ('papan_enabled', '1')`)
	_, _ = DB.Exec(`INSERT OR IGNORE INTO settings (key, value) VALUES ('school_name', 'SMAS Muhammadiyah 1 Ngawi')`)
	_, _ = DB.Exec(`INSERT OR IGNORE INTO settings (key, value) VALUES ('project_credit', '© 2026 SMAS Muhammadiyah 1 Ngawi. Sistem Rekam Jejak Alumni.')`)
	_, _ = DB.Exec(`INSERT OR IGNORE INTO settings (key, value) VALUES ('papan_description', 'Temukan info lowongan kerja, beasiswa, agenda reuni, dan informasi penting lainnya.')`)
	_, _ = DB.Exec(`INSERT OR IGNORE INTO settings (key, value) VALUES ('favicon_path', '')`)
	_, _ = DB.Exec(`INSERT OR IGNORE INTO settings (key, value) VALUES ('logo_path', '')`)
	_, _ = DB.Exec(`INSERT OR IGNORE INTO settings (key, value) VALUES ('pwa_app_name', 'Alumni Tracker')`)
	_, _ = DB.Exec(`INSERT OR IGNORE INTO settings (key, value) VALUES ('pwa_icon_path', '')`)

	var hasIsActive bool
	rows, err := DB.Query("PRAGMA table_info(info_papan)")
	if err == nil {
		for rows.Next() {
			var cid int
			var name, ctype string
			var notnull, pk int
			var dfltVal interface{}
			if err := rows.Scan(&cid, &name, &ctype, &notnull, &dfltVal, &pk); err == nil {
				if name == "is_active" {
					hasIsActive = true
				}
			}
		}
		rows.Close()
	}
	if !hasIsActive {
		if _, err := DB.Exec("ALTER TABLE info_papan ADD COLUMN is_active INTEGER DEFAULT 1"); err != nil {
			log.Printf("Failed to add is_active column: %v", err)
		}
	}

	var hasAktifSampai bool
	rows, err = DB.Query("PRAGMA table_info(info_papan)")
	if err == nil {
		for rows.Next() {
			var cid int
			var name, ctype string
			var notnull, pk int
			var dfltVal interface{}
			if err := rows.Scan(&cid, &name, &ctype, &notnull, &dfltVal, &pk); err == nil {
				if name == "aktif_sampai" {
					hasAktifSampai = true
				}
			}
		}
		rows.Close()
	}
	if !hasAktifSampai {
		if _, err := DB.Exec("ALTER TABLE info_papan ADD COLUMN aktif_sampai DATETIME"); err != nil {
			log.Printf("Failed to add aktif_sampai column: %v", err)
		}
	}

	// Add status column to alumni table for pending/active verification
	var hasStatus bool
	rows, err = DB.Query("PRAGMA table_info(alumni)")
	if err == nil {
		for rows.Next() {
			var cid int
			var name, ctype string
			var notnull, pk int
			var dfltVal interface{}
			if err := rows.Scan(&cid, &name, &ctype, &notnull, &dfltVal, &pk); err == nil {
				if name == "status" {
					hasStatus = true
				}
			}
		}
		rows.Close()
	}
	if !hasStatus {
		if _, err := DB.Exec("ALTER TABLE alumni ADD COLUMN status TEXT DEFAULT 'active'"); err != nil {
			log.Printf("Failed to add status column: %v", err)
		}
		// Ensure all existing records are marked as active
		DB.Exec("UPDATE alumni SET status = 'active' WHERE status IS NULL")
		log.Println("✅ Added 'status' column to alumni table")
	}

	// Create klaim_profil table
	_, err = DB.Exec(`CREATE TABLE IF NOT EXISTS klaim_profil (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		alumni_id INTEGER NOT NULL,
		no_wa_verifikasi TEXT NOT NULL,
		catatan_pengklaim TEXT,
		update_no_hp TEXT,
		update_email TEXT,
		update_domisili TEXT,
		update_pekerjaan TEXT,
		update_instansi TEXT,
		update_linkedin TEXT,
		update_foto_filename TEXT,
		status TEXT DEFAULT 'pending' CHECK(status IN ('pending', 'approved', 'rejected')),
		dibuat_pada DATETIME DEFAULT CURRENT_TIMESTAMP,
		FOREIGN KEY(alumni_id) REFERENCES alumni(id) ON DELETE CASCADE
	)`)
	if err != nil {
		log.Printf("Failed to create table klaim_profil: %v", err)
	} else {
		log.Println("✅ Table 'klaim_profil' created or verified")
	}

	log.Println("✅ Database migrations completed")
	return nil
}

func Close() {
	if DB != nil {
		DB.Close()
	}
}
