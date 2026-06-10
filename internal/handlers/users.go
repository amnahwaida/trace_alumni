package handlers

import (
	"database/sql"
	"log"
	"math"
	"net/http"
	"strconv"
	"strings"

	"trace-alumni/internal/database"
	"trace-alumni/internal/middleware"
	"trace-alumni/internal/models"

	"golang.org/x/crypto/bcrypt"
)

func UserList(w http.ResponseWriter, r *http.Request) {
	user := middleware.GetUser(r)
	search := strings.TrimSpace(r.URL.Query().Get("search"))

	usersPerPage := 5
	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	if page < 1 {
		page = 1
	}
	offset := (page - 1) * usersPerPage

	var totalCount int
	var rows *sql.Rows
	var err error

	if search != "" {
		pat := "%" + search + "%"
		database.DB.QueryRow(
			"SELECT COUNT(*) FROM users WHERE username LIKE ? OR role LIKE ?",
			pat, pat).Scan(&totalCount)
		rows, err = database.DB.Query(
			"SELECT id,username,role,created_at FROM users WHERE username LIKE ? OR role LIKE ? ORDER BY id ASC LIMIT ? OFFSET ?",
			pat, pat, usersPerPage, offset)
	} else {
		database.DB.QueryRow("SELECT COUNT(*) FROM users").Scan(&totalCount)
		rows, err = database.DB.Query("SELECT id,username,role,created_at FROM users ORDER BY id ASC LIMIT ? OFFSET ?", usersPerPage, offset)
	}

	totalPages := int(math.Ceil(float64(totalCount) / float64(usersPerPage)))

	if err != nil {
		log.Printf("User list error: %v", err)
		http.Error(w, "Database error", 500)
		return
	}
	defer rows.Close()

	var users []models.User
	for rows.Next() {
		var u models.User
		rows.Scan(&u.ID, &u.Username, &u.Role, &u.CreatedAt)
		users = append(users, u)
	}

	data := map[string]interface{}{
		"User":       user,
		"Users":      users,
		"PageTitle":  "Manajemen User",
		"ActiveMenu": "users",
		"Page":       page,
		"TotalPages": totalPages,
		"TotalCount": totalCount,
		"Search":     search,
		"Success":    r.URL.Query().Get("success"),
		"Error":      r.URL.Query().Get("error"),
	}

	if r.Header.Get("HX-Request") == "true" {
		renderTemplate(w, "user_table.html", data)
		return
	}
	renderTemplate(w, "user_list.html", data)
}

func UserCreate(w http.ResponseWriter, r *http.Request) {
	user := middleware.GetUser(r)
	data := map[string]interface{}{
		"User": user, "PageTitle": "Tambah User", "ActiveMenu": "users",
		"Error": r.URL.Query().Get("error"),
	}
	renderTemplate(w, "user_form.html", data)
}

func UserStore(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	username := strings.TrimSpace(r.FormValue("username"))
	password := r.FormValue("password")
	role := r.FormValue("role")

	if username == "" || password == "" || role == "" {
		http.Redirect(w, r, "/dashboard/users/create?error=Semua+field+wajib+diisi", 303)
		return
	}
	if role != "admin" && role != "staff" && role != "super_admin" {
		http.Redirect(w, r, "/dashboard/users/create?error=Role+tidak+valid", 303)
		return
	}
	if len(password) < 6 {
		http.Redirect(w, r, "/dashboard/users/create?error=Password+minimal+6+karakter", 303)
		return
	}

	hash, _ := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	_, err := database.DB.Exec("INSERT INTO users (username,password_hash,role) VALUES (?,?,?)",
		username, string(hash), role)
	if err != nil {
		if strings.Contains(err.Error(), "UNIQUE") {
			http.Redirect(w, r, "/dashboard/users/create?error=Username+sudah+digunakan", 303)
			return
		}
		log.Printf("User insert error: %v", err)
		http.Redirect(w, r, "/dashboard/users/create?error=Gagal+menyimpan", 303)
		return
	}
	http.Redirect(w, r, "/dashboard/users?success=User+berhasil+ditambahkan", 303)
}

func UserDelete(w http.ResponseWriter, r *http.Request) {
	user := middleware.GetUser(r)
	id, _ := strconv.Atoi(r.URL.Query().Get("id"))
	if id == user.ID {
		http.Redirect(w, r, "/dashboard/users?error=Tidak+bisa+menghapus+akun+sendiri", 303)
		return
	}
	database.DB.Exec("DELETE FROM sessions WHERE user_id=?", id)
	database.DB.Exec("DELETE FROM users WHERE id=?", id)
	http.Redirect(w, r, "/dashboard/users?success=User+berhasil+dihapus", 303)
}
