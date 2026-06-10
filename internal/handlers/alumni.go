package handlers

import (
	"database/sql"
	"fmt"
	"image"
	"image/jpeg"
	_ "image/png" // Support PNG uploads
	_ "image/gif" // Support GIF uploads
	"log"
	"math"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"trace-alumni/internal/database"
	"trace-alumni/internal/middleware"
	"trace-alumni/internal/models"

	"github.com/disintegration/imaging"
	"github.com/google/uuid"
)

const alumniPerPage = 5

func AlumniList(w http.ResponseWriter, r *http.Request) {
	user := middleware.GetUser(r)
	search := strings.TrimSpace(r.URL.Query().Get("search"))
	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	if page < 1 {
		page = 1
	}
	offset := (page - 1) * alumniPerPage

	var totalCount int
	var rows *sql.Rows
	var err error

	if search != "" {
		pat := "%" + search + "%"
		database.DB.QueryRow(`
			SELECT COUNT(*) FROM alumni 
			WHERE nama_lengkap LIKE ? 
			   OR CAST(tahun_lulus AS TEXT) LIKE ? 
			   OR domisili_sekarang LIKE ? 
			   OR pekerjaan LIKE ? 
			   OR no_hp LIKE ?`,
			pat, pat, pat, pat, pat).Scan(&totalCount)
		rows, err = database.DB.Query(`
			SELECT id,nama_lengkap,alamat_asli,domisili_sekarang,no_hp,email,
			tahun_lulus,tanggal_lahir,pekerjaan,instansi,url_linkedin,foto_profil,
			created_at,updated_at FROM alumni
			WHERE nama_lengkap LIKE ? 
			   OR CAST(tahun_lulus AS TEXT) LIKE ? 
			   OR domisili_sekarang LIKE ? 
			   OR pekerjaan LIKE ? 
			   OR no_hp LIKE ?
			ORDER BY id DESC LIMIT ? OFFSET ?`, pat, pat, pat, pat, pat, alumniPerPage, offset)
	} else {
		database.DB.QueryRow("SELECT COUNT(*) FROM alumni").Scan(&totalCount)
		rows, err = database.DB.Query(`
			SELECT id,nama_lengkap,alamat_asli,domisili_sekarang,no_hp,email,
			tahun_lulus,tanggal_lahir,pekerjaan,instansi,url_linkedin,foto_profil,
			created_at,updated_at FROM alumni ORDER BY id DESC LIMIT ? OFFSET ?`,
			alumniPerPage, offset)
	}
	if err != nil {
		log.Printf("Query error: %v", err)
		http.Error(w, "Database error", 500)
		return
	}
	defer rows.Close()

	var list []models.Alumni
	for rows.Next() {
		var a models.Alumni
		if err := rows.Scan(&a.ID, &a.NamaLengkap, &a.AlamatAsli, &a.DomisiliSekarang,
			&a.NoHP, &a.Email, &a.TahunLulus, &a.TanggalLahir,
			&a.Pekerjaan, &a.Instansi, &a.URLLinkedIn, &a.FotoProfil,
			&a.CreatedAt, &a.UpdatedAt); err != nil {
			log.Printf("Scan error: %v", err)
			continue
		}
		list = append(list, a)
	}

	totalPages := int(math.Ceil(float64(totalCount) / float64(alumniPerPage)))
	data := map[string]interface{}{
		"User": user, "Alumni": list, "Search": search,
		"Page": page, "TotalPages": totalPages, "TotalCount": totalCount,
		"PageTitle": "Data Alumni", "ActiveMenu": "alumni",
		"Success": r.URL.Query().Get("success"),
		"Error":   r.URL.Query().Get("error"),
	}
	if r.Header.Get("HX-Request") == "true" {
		renderTemplate(w, "alumni_table.html", data)
		return
	}
	renderTemplate(w, "alumni_list.html", data)
}

func AlumniCreate(w http.ResponseWriter, r *http.Request) {
	user := middleware.GetUser(r)
	data := map[string]interface{}{
		"User": user, "PageTitle": "Tambah Alumni", "ActiveMenu": "alumni",
		"IsEdit": false, "Alumni": models.Alumni{},
		"Success": r.URL.Query().Get("success"),
		"Error":   r.URL.Query().Get("error"),
	}
	renderTemplate(w, "alumni_form.html", data)
}

func processPhoto(r *http.Request, fieldName string) (string, error) {
	file, header, err := r.FormFile(fieldName)
	if err != nil {
		if err == http.ErrMissingFile {
			return "", nil
		}
		return "", err
	}
	defer file.Close()

	// Limit 2 MB
	if header.Size > 2*1024*1024 {
		return "", fmt.Errorf("ukuran file foto melebihi 2MB")
	}

	img, _, err := image.Decode(file)
	if err != nil {
		return "", fmt.Errorf("gagal memproses file foto: %w", err)
	}

	// Fit to max 200x200px
	resizedImg := imaging.Fit(img, 200, 200, imaging.Lanczos)

	dataDir := os.Getenv("DATA_DIR")
	if dataDir == "" {
		dataDir = "./data"
	}
	uploadsDir := filepath.Join(dataDir, "uploads")

	fileName := uuid.New().String() + ".jpg"
	filePath := filepath.Join(uploadsDir, fileName)

	out, err := os.Create(filePath)
	if err != nil {
		return "", fmt.Errorf("gagal menyimpan foto: %w", err)
	}
	defer out.Close()

	// Compress to JPEG with Quality 65%
	err = jpeg.Encode(out, resizedImg, &jpeg.Options{Quality: 65})
	if err != nil {
		os.Remove(filePath)
		return "", fmt.Errorf("gagal kompresi foto: %w", err)
	}

	return fileName, nil
}

func AlumniStore(w http.ResponseWriter, r *http.Request) {
	err := r.ParseMultipartForm(5 << 20)
	if err != nil {
		http.Redirect(w, r, "/dashboard/alumni/create?error=Gagal+mengunggah+file", 303)
		return
	}

	nama := strings.TrimSpace(r.FormValue("nama_lengkap"))
	tls := r.FormValue("tahun_lulus")
	if nama == "" || tls == "" {
		http.Redirect(w, r, "/dashboard/alumni/create?error=Nama+dan+Tahun+Lulus+wajib+diisi", 303)
		return
	}
	tl, err := strconv.Atoi(tls)
	if err != nil || tl < 1900 || tl > time.Now().Year()+1 {
		http.Redirect(w, r, "/dashboard/alumni/create?error=Tahun+Lulus+tidak+valid", 303)
		return
	}

	fotoFileName, err := processPhoto(r, "foto_profil")
	if err != nil {
		http.Redirect(w, r, fmt.Sprintf("/dashboard/alumni/create?error=%s", url.QueryEscape(err.Error())), 303)
		return
	}

	var fotoPtr *string
	if fotoFileName != "" {
		fotoPtr = &fotoFileName
	}

	_, err = database.DB.Exec(`INSERT INTO alumni (nama_lengkap,alamat_asli,domisili_sekarang,no_hp,email,tahun_lulus,tanggal_lahir,pekerjaan,instansi,url_linkedin,foto_profil) VALUES (?,?,?,?,?,?,?,?,?,?,?)`,
		nama, ns(r.FormValue("alamat_asli")), ns(r.FormValue("domisili_sekarang")),
		ns(r.FormValue("no_hp")), ns(r.FormValue("email")), tl,
		ns(r.FormValue("tanggal_lahir")), ns(r.FormValue("pekerjaan")),
		ns(r.FormValue("instansi")), ns(r.FormValue("url_linkedin")), fotoPtr)

	if err != nil {
		log.Printf("Insert error: %v", err)
		if fotoFileName != "" {
			dataDir := os.Getenv("DATA_DIR")
			if dataDir == "" {
				dataDir = "./data"
			}
			os.Remove(filepath.Join(dataDir, "uploads", fotoFileName))
		}
		http.Redirect(w, r, "/dashboard/alumni/create?error=Gagal+menyimpan", 303)
		return
	}
	http.Redirect(w, r, "/dashboard/alumni?success=Data+alumni+berhasil+ditambahkan", 303)
}

func AlumniEdit(w http.ResponseWriter, r *http.Request) {
	user := middleware.GetUser(r)
	id, _ := strconv.Atoi(r.URL.Query().Get("id"))
	if id == 0 {
		http.Redirect(w, r, "/dashboard/alumni", 303)
		return
	}
	var a models.Alumni
	err := database.DB.QueryRow(`SELECT id,nama_lengkap,alamat_asli,domisili_sekarang,no_hp,email,tahun_lulus,tanggal_lahir,pekerjaan,instansi,url_linkedin,foto_profil,created_at,updated_at FROM alumni WHERE id=?`, id).Scan(
		&a.ID, &a.NamaLengkap, &a.AlamatAsli, &a.DomisiliSekarang, &a.NoHP, &a.Email,
		&a.TahunLulus, &a.TanggalLahir, &a.Pekerjaan, &a.Instansi, &a.URLLinkedIn,
		&a.FotoProfil, &a.CreatedAt, &a.UpdatedAt)
	if err != nil {
		http.Redirect(w, r, "/dashboard/alumni?error=Data+tidak+ditemukan", 303)
		return
	}
	data := map[string]interface{}{
		"User": user, "PageTitle": "Edit Alumni", "ActiveMenu": "alumni",
		"IsEdit": true, "Alumni": a, "Error": r.URL.Query().Get("error"),
	}
	renderTemplate(w, "alumni_form.html", data)
}

func AlumniUpdate(w http.ResponseWriter, r *http.Request) {
	err := r.ParseMultipartForm(5 << 20)
	if err != nil {
		http.Error(w, "Gagal mengunggah file", http.StatusBadRequest)
		return
	}

	id, _ := strconv.Atoi(r.FormValue("id"))
	nama := strings.TrimSpace(r.FormValue("nama_lengkap"))
	tls := r.FormValue("tahun_lulus")
	if nama == "" || tls == "" {
		http.Redirect(w, r, fmt.Sprintf("/dashboard/alumni/edit?id=%d&error=Wajib+diisi", id), 303)
		return
	}
	tl, err := strconv.Atoi(tls)
	if err != nil || tl < 1900 || tl > time.Now().Year()+1 {
		http.Redirect(w, r, fmt.Sprintf("/dashboard/alumni/edit?id=%d&error=Tahun+tidak+valid", id), 303)
		return
	}

	var existingFoto *string
	database.DB.QueryRow("SELECT foto_profil FROM alumni WHERE id = ?", id).Scan(&existingFoto)

	newFotoFileName, err := processPhoto(r, "foto_profil")
	if err != nil {
		http.Redirect(w, r, fmt.Sprintf("/dashboard/alumni/edit?id=%d&error=%s", id, url.QueryEscape(err.Error())), 303)
		return
	}

	var updateQuery string
	var args []interface{}
	if newFotoFileName != "" {
		updateQuery = `UPDATE alumni SET nama_lengkap=?,alamat_asli=?,domisili_sekarang=?,no_hp=?,email=?,tahun_lulus=?,tanggal_lahir=?,pekerjaan=?,instansi=?,url_linkedin=?,foto_profil=?,updated_at=CURRENT_TIMESTAMP WHERE id=?`
		args = []interface{}{
			nama, ns(r.FormValue("alamat_asli")), ns(r.FormValue("domisili_sekarang")),
			ns(r.FormValue("no_hp")), ns(r.FormValue("email")), tl,
			ns(r.FormValue("tanggal_lahir")), ns(r.FormValue("pekerjaan")),
			ns(r.FormValue("instansi")), ns(r.FormValue("url_linkedin")), newFotoFileName, id,
		}
	} else {
		updateQuery = `UPDATE alumni SET nama_lengkap=?,alamat_asli=?,domisili_sekarang=?,no_hp=?,email=?,tahun_lulus=?,tanggal_lahir=?,pekerjaan=?,instansi=?,url_linkedin=?,updated_at=CURRENT_TIMESTAMP WHERE id=?`
		args = []interface{}{
			nama, ns(r.FormValue("alamat_asli")), ns(r.FormValue("domisili_sekarang")),
			ns(r.FormValue("no_hp")), ns(r.FormValue("email")), tl,
			ns(r.FormValue("tanggal_lahir")), ns(r.FormValue("pekerjaan")),
			ns(r.FormValue("instansi")), ns(r.FormValue("url_linkedin")), id,
		}
	}

	_, err = database.DB.Exec(updateQuery, args...)
	if err != nil {
		log.Printf("Update error: %v", err)
		if newFotoFileName != "" {
			dataDir := os.Getenv("DATA_DIR")
			if dataDir == "" {
				dataDir = "./data"
			}
			os.Remove(filepath.Join(dataDir, "uploads", newFotoFileName))
		}
		http.Redirect(w, r, fmt.Sprintf("/dashboard/alumni/edit?id=%d&error=Gagal+update", id), 303)
		return
	}

	if newFotoFileName != "" && existingFoto != nil && *existingFoto != "" {
		dataDir := os.Getenv("DATA_DIR")
		if dataDir == "" {
			dataDir = "./data"
		}
		os.Remove(filepath.Join(dataDir, "uploads", *existingFoto))
	}

	http.Redirect(w, r, "/dashboard/alumni?success=Data+alumni+berhasil+diperbarui", 303)
}

func AlumniDelete(w http.ResponseWriter, r *http.Request) {
	id, _ := strconv.Atoi(r.URL.Query().Get("id"))
	
	var existingFoto *string
	database.DB.QueryRow("SELECT foto_profil FROM alumni WHERE id = ?", id).Scan(&existingFoto)

	_, err := database.DB.Exec("DELETE FROM alumni WHERE id=?", id)
	if err == nil && existingFoto != nil && *existingFoto != "" {
		dataDir := os.Getenv("DATA_DIR")
		if dataDir == "" {
			dataDir = "./data"
		}
		os.Remove(filepath.Join(dataDir, "uploads", *existingFoto))
	}

	if r.Header.Get("HX-Request") == "true" {
		w.Header().Set("HX-Redirect", "/dashboard/alumni?success=Data+dihapus")
		w.WriteHeader(200)
		return
	}
	http.Redirect(w, r, "/dashboard/alumni?success=Data+dihapus", 303)
}

func ns(s string) *string {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil
	}
	return &s
}
