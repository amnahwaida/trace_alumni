package handlers

import (
	"database/sql"
	"fmt"
	"log"
	"math"
	"net/http"
	"strconv"
	"strings"

	"trace-alumni/internal/database"
	"trace-alumni/internal/middleware"
	"trace-alumni/internal/models"
)

// PapanList renders the announcement list in dashboard
func PapanList(w http.ResponseWriter, r *http.Request) {
	user := middleware.GetUser(r)
	search := strings.TrimSpace(r.URL.Query().Get("search"))

	// Get global toggle setting
	var val string
	_ = database.DB.QueryRow("SELECT value FROM settings WHERE key = 'papan_enabled'").Scan(&val)
	if val == "" {
		val = "1"
	}
	papanEnabled := val == "1"

	papanPerPage := 5
	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	if page < 1 {
		page = 1
	}
	offset := (page - 1) * papanPerPage

	var totalCount int
	var rows *sql.Rows
	var err error

	if search != "" {
		pat := "%" + search + "%"
		database.DB.QueryRow(
			"SELECT COUNT(*) FROM info_papan WHERE judul LIKE ? OR deskripsi LIKE ? OR kategori LIKE ?",
			pat, pat, pat).Scan(&totalCount)
		rows, err = database.DB.Query(`
			SELECT p.id, p.judul, p.deskripsi, p.link_eksternal, p.kategori, p.dibuat_oleh, p.dibuat_pada, p.is_active, p.aktif_sampai, u.username
			FROM info_papan p
			LEFT JOIN users u ON p.dibuat_oleh = u.id
			WHERE p.judul LIKE ? OR p.deskripsi LIKE ? OR p.kategori LIKE ?
			ORDER BY p.id DESC LIMIT ? OFFSET ?
		`, pat, pat, pat, papanPerPage, offset)
	} else {
		database.DB.QueryRow("SELECT COUNT(*) FROM info_papan").Scan(&totalCount)
		rows, err = database.DB.Query(`
			SELECT p.id, p.judul, p.deskripsi, p.link_eksternal, p.kategori, p.dibuat_oleh, p.dibuat_pada, p.is_active, p.aktif_sampai, u.username
			FROM info_papan p
			LEFT JOIN users u ON p.dibuat_oleh = u.id
			ORDER BY p.id DESC LIMIT ? OFFSET ?
		`, papanPerPage, offset)
	}

	totalPages := int(math.Ceil(float64(totalCount) / float64(papanPerPage)))

	if err != nil {
		log.Printf("Papan query error: %v", err)
		http.Error(w, "Database error", 500)
		return
	}
	defer rows.Close()

	var list []models.InfoPapan
	for rows.Next() {
		var p models.InfoPapan
		var username sql.NullString
		err := rows.Scan(&p.ID, &p.Judul, &p.Deskripsi, &p.LinkEksternal, &p.Kategori, &p.DibuatOleh, &p.DibuatPada, &p.IsActive, &p.AktifSampai, &username)
		if err != nil {
			log.Printf("Papan scan error: %v", err)
			continue
		}
		if username.Valid {
			p.DibuatOlehNama = username.String
		} else {
			p.DibuatOlehNama = "Sistem"
		}
		list = append(list, p)
	}

	data := map[string]interface{}{
		"User":         user,
		"Papan":        list,
		"PapanEnabled": papanEnabled,
		"PageTitle":    "Papan Pengumuman",
		"ActiveMenu":   "papan",
		"Page":         page,
		"TotalPages":   totalPages,
		"TotalCount":   totalCount,
		"Search":       search,
		"Success":      r.URL.Query().Get("success"),
		"Error":        r.URL.Query().Get("error"),
	}

	if r.Header.Get("HX-Request") == "true" {
		renderTemplate(w, "papan_table.html", data)
		return
	}
	renderTemplate(w, "papan_list.html", data)
}

// PapanCreate renders the form to create announcement
func PapanCreate(w http.ResponseWriter, r *http.Request) {
	user := middleware.GetUser(r)
	data := map[string]interface{}{
		"User":       user,
		"PageTitle":  "Tambah Pengumuman",
		"ActiveMenu": "papan",
		"IsEdit":     false,
		"Papan":      models.InfoPapan{IsActive: true}, // Default to active on create
		"Error":      r.URL.Query().Get("error"),
	}
	renderTemplate(w, "papan_form.html", data)
}

// PapanStore handles creation form submission
func PapanStore(w http.ResponseWriter, r *http.Request) {
	user := middleware.GetUser(r)
	r.ParseForm()

	judul := strings.TrimSpace(r.FormValue("judul"))
	deskripsi := strings.TrimSpace(r.FormValue("deskripsi"))
	link := strings.TrimSpace(r.FormValue("link_eksternal"))
	kategori := r.FormValue("kategori")
	
	isActive := 0
	if r.FormValue("is_active") == "1" {
		isActive = 1
	}

	aktifSampai := strings.TrimSpace(r.FormValue("aktif_sampai"))
	var aktifSampaiPtr *string
	if aktifSampai != "" {
		aktifSampaiPtr = &aktifSampai
	}

	if judul == "" || deskripsi == "" || kategori == "" {
		http.Redirect(w, r, "/dashboard/papan/create?error=Judul,+Deskripsi,+dan+Kategori+wajib+diisi", 303)
		return
	}

	// Validate link (Must start with https://)
	var linkPtr *string
	if link != "" {
		if !strings.HasPrefix(link, "https://") {
			http.Redirect(w, r, "/dashboard/papan/create?error=Link+Eksternal+wajib+diawali+dengan+https://", 303)
			return
		}
		linkPtr = &link
	}

	_, err := database.DB.Exec(`
		INSERT INTO info_papan (judul, deskripsi, link_eksternal, kategori, dibuat_oleh, is_active, aktif_sampai)
		VALUES (?, ?, ?, ?, ?, ?, ?)
	`, judul, deskripsi, linkPtr, kategori, user.ID, isActive, aktifSampaiPtr)

	if err != nil {
		log.Printf("Insert papan error: %v", err)
		http.Redirect(w, r, "/dashboard/papan/create?error=Gagal+menyimpan+pengumuman", 303)
		return
	}

	http.Redirect(w, r, "/dashboard/papan?success=Pengumuman+berhasil+ditambahkan", 303)
}

// PapanEdit renders edit form
func PapanEdit(w http.ResponseWriter, r *http.Request) {
	user := middleware.GetUser(r)
	id, _ := strconv.Atoi(r.URL.Query().Get("id"))
	if id == 0 {
		http.Redirect(w, r, "/dashboard/papan", 303)
		return
	}

	var p models.InfoPapan
	err := database.DB.QueryRow(`
		SELECT id, judul, deskripsi, link_eksternal, kategori, dibuat_oleh, dibuat_pada, is_active, aktif_sampai
		FROM info_papan WHERE id = ?
	`, id).Scan(&p.ID, &p.Judul, &p.Deskripsi, &p.LinkEksternal, &p.Kategori, &p.DibuatOleh, &p.DibuatPada, &p.IsActive, &p.AktifSampai)

	if err != nil {
		http.Redirect(w, r, "/dashboard/papan?error=Pengumuman+tidak+ditemukan", 303)
		return
	}

	data := map[string]interface{}{
		"User":       user,
		"PageTitle":  "Edit Pengumuman",
		"ActiveMenu": "papan",
		"IsEdit":     true,
		"Papan":      p,
		"Error":      r.URL.Query().Get("error"),
	}
	renderTemplate(w, "papan_form.html", data)
}

// PapanUpdate handles edit form submission
func PapanUpdate(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	id, _ := strconv.Atoi(r.FormValue("id"))
	judul := strings.TrimSpace(r.FormValue("judul"))
	deskripsi := strings.TrimSpace(r.FormValue("deskripsi"))
	link := strings.TrimSpace(r.FormValue("link_eksternal"))
	kategori := r.FormValue("kategori")
	
	isActive := 0
	if r.FormValue("is_active") == "1" {
		isActive = 1
	}

	aktifSampai := strings.TrimSpace(r.FormValue("aktif_sampai"))
	var aktifSampaiPtr *string
	if aktifSampai != "" {
		aktifSampaiPtr = &aktifSampai
	}

	if id == 0 || judul == "" || deskripsi == "" || kategori == "" {
		http.Redirect(w, r, fmt.Sprintf("/dashboard/papan/edit?id=%d&error=Semua+field+wajib+diisi", id), 303)
		return
	}

	var linkPtr *string
	if link != "" {
		if !strings.HasPrefix(link, "https://") {
			http.Redirect(w, r, fmt.Sprintf("/dashboard/papan/edit?id=%d&error=Link+Eksternal+wajib+diawali+dengan+https://", id), 303)
			return
		}
		linkPtr = &link
	}

	_, err := database.DB.Exec(`
		UPDATE info_papan SET judul = ?, deskripsi = ?, link_eksternal = ?, kategori = ?, is_active = ?, aktif_sampai = ?
		WHERE id = ?
	`, judul, deskripsi, linkPtr, kategori, isActive, aktifSampaiPtr, id)

	if err != nil {
		log.Printf("Update papan error: %v", err)
		http.Redirect(w, r, fmt.Sprintf("/dashboard/papan/edit?id=%d&error=Gagal+mengupdate+pengumuman", id), 303)
		return
	}

	http.Redirect(w, r, "/dashboard/papan?success=Pengumuman+berhasil+diperbarui", 303)
}

// PapanDelete handles announcement deletion
func PapanDelete(w http.ResponseWriter, r *http.Request) {
	id, _ := strconv.Atoi(r.URL.Query().Get("id"))
	if id == 0 {
		http.Redirect(w, r, "/dashboard/papan", 303)
		return
	}

	_, err := database.DB.Exec("DELETE FROM info_papan WHERE id = ?", id)
	if err != nil {
		log.Printf("Delete papan error: %v", err)
		http.Redirect(w, r, "/dashboard/papan?error=Gagal+menghapus+pengumuman", 303)
		return
	}

	http.Redirect(w, r, "/dashboard/papan?success=Pengumuman+berhasil+dihapus", 303)
}

// PapanToggleGlobal toggles global active status of announcement board
func PapanToggleGlobal(w http.ResponseWriter, r *http.Request) {
	var val string
	_ = database.DB.QueryRow("SELECT value FROM settings WHERE key = 'papan_enabled'").Scan(&val)
	newVal := "1"
	if val == "1" {
		newVal = "0"
	}
	_, err := database.DB.Exec("UPDATE settings SET value = ? WHERE key = 'papan_enabled'", newVal)
	if err != nil {
		log.Printf("Toggle global settings error: %v", err)
		http.Redirect(w, r, "/dashboard/papan?error=Gagal+mengubah+status+papan+pengumuman", 303)
		return
	}
	msg := "Papan pengumuman dinonaktifkan secara global"
	if newVal == "1" {
		msg = "Papan pengumuman diaktifkan secara global"
	}
	http.Redirect(w, r, "/dashboard/papan?success="+msg, 303)
}

// PublicPapan renders announcements for landing page
func PublicPapan(w http.ResponseWriter, r *http.Request) {
	// Check if enabled globally
	var val string
	_ = database.DB.QueryRow("SELECT value FROM settings WHERE key = 'papan_enabled'").Scan(&val)
	if val == "" {
		val = "1"
	}
	if val == "0" {
		data := map[string]interface{}{
			"PapanEnabled": false,
			"PageTitle":    "Papan Pengumuman - SMAS Muhammadiyah 1 Ngawi",
		}
		renderTemplate(w, "public_papan.html", data)
		return
	}

	rows, err := database.DB.Query(`
		SELECT id, judul, deskripsi, link_eksternal, kategori, dibuat_pada, aktif_sampai
		FROM info_papan
		WHERE is_active = 1 AND (aktif_sampai IS NULL OR aktif_sampai = '' OR date(aktif_sampai) >= date('now', 'localtime'))
		ORDER BY id DESC
	`)
	if err != nil {
		log.Printf("Public papan query error: %v", err)
		http.Error(w, "Database error", 500)
		return
	}
	defer rows.Close()

	var list []models.InfoPapan
	for rows.Next() {
		var p models.InfoPapan
		err := rows.Scan(&p.ID, &p.Judul, &p.Deskripsi, &p.LinkEksternal, &p.Kategori, &p.DibuatPada, &p.AktifSampai)
		if err != nil {
			log.Printf("Public papan scan error: %v", err)
			continue
		}
		list = append(list, p)
	}

	data := map[string]interface{}{
		"Papan":        list,
		"PapanEnabled": true,
		"PageTitle":    "Papan Pengumuman - SMAS Muhammadiyah 1 Ngawi",
	}
	renderTemplate(w, "public_papan.html", data)
}

// PapanToggleActive toggles active status of a specific announcement
func PapanToggleActive(w http.ResponseWriter, r *http.Request) {
	id, _ := strconv.Atoi(r.URL.Query().Get("id"))
	if id == 0 {
		http.Redirect(w, r, "/dashboard/papan", 303)
		return
	}

	var isActive bool
	err := database.DB.QueryRow("SELECT is_active FROM info_papan WHERE id = ?", id).Scan(&isActive)
	if err != nil {
		log.Printf("Query active state error: %v", err)
		http.Redirect(w, r, "/dashboard/papan?error=Pengumuman+tidak+ditemukan", 303)
		return
	}

	newActive := 1
	if isActive {
		newActive = 0
	}

	_, err = database.DB.Exec("UPDATE info_papan SET is_active = ? WHERE id = ?", newActive, id)
	if err != nil {
		log.Printf("Update active state error: %v", err)
		http.Redirect(w, r, "/dashboard/papan?error=Gagal+mengubah+status+pengumuman", 303)
		return
	}

	msg := "Status pengumuman berhasil diubah"
	http.Redirect(w, r, "/dashboard/papan?success="+msg, 303)
}
