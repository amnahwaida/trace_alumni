package handlers

import (
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"trace-alumni/internal/database"
	"trace-alumni/internal/middleware"

	"golang.org/x/crypto/bcrypt"
)

var templates *template.Template

func InitTemplates(templateDir string) error {
	funcMap := template.FuncMap{
		"add": func(a, b int) int { return a + b },
		"sub": func(a, b int) int { return a - b },
		"seq": func(start, end int) []int {
			s := make([]int, 0, end-start+1)
			for i := start; i <= end; i++ {
				s = append(s, i)
			}
			return s
		},
		"deref": func(s *string) string {
			if s == nil {
				return ""
			}
			return *s
		},
		"safeHTML": func(s string) template.HTML {
			return template.HTML(s)
		},
		"truncate": func(s string, n int) string {
			if len(s) <= n {
				return s
			}
			return s[:n] + "..."
		},
		"categoryBadge": func(cat string) string {
			switch cat {
			case "loker":
				return "bg-blue-500"
			case "beasiswa":
				return "bg-green-500"
			case "reuni":
				return "bg-purple-500"
			default:
				return "bg-gray-500"
			}
		},
		"categoryLabel": func(cat string) string {
			switch cat {
			case "loker":
				return "Lowongan Kerja"
			case "beasiswa":
				return "Beasiswa"
			case "reuni":
				return "Reuni"
			default:
				return "Umum"
			}
		},
		"formatDate": func(dateStr string) string {
			if dateStr == "" {
				return "-"
			}
			// Let's try parsing as RFC3339 (default SQLite datetime/ISO)
			t, err := time.Parse(time.RFC3339, dateStr)
			if err != nil {
				// Try parsing other common formats like YYYY-MM-DD HH:MM:SS or YYYY-MM-DD
				t, err = time.Parse("2006-01-02 15:04:05", dateStr)
				if err != nil {
					t, err = time.Parse("2006-01-02", dateStr)
					if err != nil {
						return dateStr // Fallback to raw string
					}
				}
			}

			months := []string{
				"Januari", "Februari", "Maret", "April", "Mei", "Juni",
				"Juli", "Agustus", "September", "Oktober", "November", "Desember",
			}
			day := t.Day()
			month := months[t.Month()-1]
			year := t.Year()

			return fmt.Sprintf("%d %s %d", day, month, year)
		},
		"pageNumbers": func(current, total int) []int {
			start := current - 2
			if start < 1 {
				start = 1
			}
			end := start + 4
			if end > total {
				end = total
				start = end - 4
				if start < 1 {
					start = 1
				}
			}
			s := make([]int, 0, end-start+1)
			for i := start; i <= end; i++ {
				s = append(s, i)
			}
			return s
		},
	}

	tmpl := template.New("").Funcs(funcMap)
	err := filepath.Walk(templateDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() && strings.HasSuffix(info.Name(), ".html") {
			_, err = tmpl.ParseFiles(path)
			if err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		return err
	}

	templates = tmpl
	log.Println("✅ Templates loaded")
	return nil
}

func getGlobalSettings() map[string]string {
	settings := map[string]string{
		"school_name":    "SMAS Muhammadiyah 1 Ngawi",
		"project_credit": "© 2026 SMAS Muhammadiyah 1 Ngawi. Sistem Rekam Jejak Alumni.",
		"favicon_path":   "",
		"logo_path":      "",
	}

	rows, err := database.DB.Query("SELECT key, value FROM settings")
	if err != nil {
		return settings
	}
	defer rows.Close()

	for rows.Next() {
		var key, value string
		if err := rows.Scan(&key, &value); err == nil {
			settings[key] = value
		}
	}
	return settings
}

func renderTemplate(w http.ResponseWriter, name string, data interface{}) {
	if m, ok := data.(map[string]interface{}); ok {
		m["Settings"] = getGlobalSettings()
	}
	err := templates.ExecuteTemplate(w, name, data)
	if err != nil {
		log.Printf("Template error (%s): %v", name, err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}
}

// generateSessionID creates a cryptographically random session ID
func generateSessionID() (string, error) {
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes), nil
}

// LoginPage renders the login form
func LoginPage(w http.ResponseWriter, r *http.Request) {
	// If already logged in, redirect to dashboard
	cookie, err := r.Cookie("session_id")
	if err == nil {
		var count int
		database.DB.QueryRow("SELECT COUNT(*) FROM sessions WHERE id = ? AND expires_at > ?",
			cookie.Value, time.Now().UTC()).Scan(&count)
		if count > 0 {
			http.Redirect(w, r, "/dashboard", http.StatusSeeOther)
			return
		}
	}

	data := map[string]interface{}{
		"Error": r.URL.Query().Get("error"),
	}
	renderTemplate(w, "login.html", data)
}

// LoginPost handles login form submission
func LoginPost(w http.ResponseWriter, r *http.Request) {
	username := r.FormValue("username")
	password := r.FormValue("password")

	if username == "" || password == "" {
		http.Redirect(w, r, "/login?error=Username+dan+password+wajib+diisi", http.StatusSeeOther)
		return
	}

	var user struct {
		ID           int
		PasswordHash string
		Role         string
	}

	err := database.DB.QueryRow(
		"SELECT id, password_hash, role FROM users WHERE username = ?", username,
	).Scan(&user.ID, &user.PasswordHash, &user.Role)

	if err == sql.ErrNoRows {
		http.Redirect(w, r, "/login?error=Username+atau+password+salah", http.StatusSeeOther)
		return
	}
	if err != nil {
		log.Printf("Login query error: %v", err)
		http.Redirect(w, r, "/login?error=Terjadi+kesalahan+server", http.StatusSeeOther)
		return
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password)); err != nil {
		http.Redirect(w, r, "/login?error=Username+atau+password+salah", http.StatusSeeOther)
		return
	}

	// Create session
	sessionID, err := generateSessionID()
	if err != nil {
		log.Printf("Session generation error: %v", err)
		http.Redirect(w, r, "/login?error=Terjadi+kesalahan+server", http.StatusSeeOther)
		return
	}

	expiresAt := time.Now().UTC().Add(24 * time.Hour)
	_, err = database.DB.Exec(
		"INSERT INTO sessions (id, user_id, expires_at) VALUES (?, ?, ?)",
		sessionID, user.ID, expiresAt,
	)
	if err != nil {
		log.Printf("Session insert error: %v", err)
		http.Redirect(w, r, "/login?error=Terjadi+kesalahan+server", http.StatusSeeOther)
		return
	}

	// Clean up expired sessions
	database.DB.Exec("DELETE FROM sessions WHERE expires_at < ?", time.Now().UTC())

	http.SetCookie(w, &http.Cookie{
		Name:     "session_id",
		Value:    sessionID,
		Path:     "/",
		MaxAge:   86400,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	})

	http.Redirect(w, r, "/dashboard", http.StatusSeeOther)
}

// Logout destroys the session
func Logout(w http.ResponseWriter, r *http.Request) {
	cookie, err := r.Cookie("session_id")
	if err == nil {
		database.DB.Exec("DELETE FROM sessions WHERE id = ?", cookie.Value)
	}

	http.SetCookie(w, &http.Cookie{
		Name:     "session_id",
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
	})

	http.Redirect(w, r, "/login", http.StatusSeeOther)
}

// Dashboard renders the main dashboard page
func Dashboard(w http.ResponseWriter, r *http.Request) {
	user := middleware.GetUser(r)

	// Get stats
	var totalAlumni, totalPengumuman, totalUsers int
	database.DB.QueryRow("SELECT COUNT(*) FROM alumni").Scan(&totalAlumni)
	database.DB.QueryRow("SELECT COUNT(*) FROM info_papan").Scan(&totalPengumuman)
	database.DB.QueryRow("SELECT COUNT(*) FROM users").Scan(&totalUsers)

	data := map[string]interface{}{
		"User":            user,
		"TotalAlumni":     totalAlumni,
		"TotalPengumuman": totalPengumuman,
		"TotalUsers":      totalUsers,
		"PageTitle":       "Dashboard",
		"ActiveMenu":      "dashboard",
	}
	renderTemplate(w, "dashboard.html", data)
}
