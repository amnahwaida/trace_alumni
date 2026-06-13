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
			WHERE status = 'active' AND (nama_lengkap LIKE ? 
			   OR CAST(tahun_lulus AS TEXT) LIKE ? 
			   OR domisili_sekarang LIKE ? 
			   OR pekerjaan LIKE ? 
			   OR no_hp LIKE ?)`,
			pat, pat, pat, pat, pat).Scan(&totalCount)
		rows, err = database.DB.Query(`
			SELECT id,nama_lengkap,alamat_asli,domisili_sekarang,no_hp,email,
			tahun_lulus,tanggal_lahir,pekerjaan,instansi,url_linkedin,foto_profil,
			created_at,updated_at FROM alumni
			WHERE status = 'active' AND (nama_lengkap LIKE ? 
			   OR CAST(tahun_lulus AS TEXT) LIKE ? 
			   OR domisili_sekarang LIKE ? 
			   OR pekerjaan LIKE ? 
			   OR no_hp LIKE ?)
			ORDER BY id DESC LIMIT ? OFFSET ?`, pat, pat, pat, pat, pat, alumniPerPage, offset)
	} else {
		database.DB.QueryRow("SELECT COUNT(*) FROM alumni WHERE status = 'active'").Scan(&totalCount)
		rows, err = database.DB.Query(`
			SELECT id,nama_lengkap,alamat_asli,domisili_sekarang,no_hp,email,
			tahun_lulus,tanggal_lahir,pekerjaan,instansi,url_linkedin,foto_profil,
			created_at,updated_at FROM alumni WHERE status = 'active' ORDER BY id DESC LIMIT ? OFFSET ?`,
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

func ProcessPhoto(r *http.Request, fieldName string) (string, error) {
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

	fotoFileName, err := ProcessPhoto(r, "foto_profil")
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

	newFotoFileName, err := ProcessPhoto(r, "foto_profil")
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

func AlumniVerifyList(w http.ResponseWriter, r *http.Request) {
	user := middleware.GetUser(r)
	
	rows, err := database.DB.Query(`
		SELECT id,nama_lengkap,alamat_asli,domisili_sekarang,no_hp,email,
		tahun_lulus,tanggal_lahir,pekerjaan,instansi,url_linkedin,foto_profil,
		created_at,updated_at FROM alumni WHERE status = 'pending' ORDER BY id DESC`)
	if err != nil {
		log.Printf("Query verify error: %v", err)
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
			log.Printf("Scan verify error: %v", err)
			continue
		}
		list = append(list, a)
	}

	data := map[string]interface{}{
		"User": user, "Alumni": list,
		"PageTitle": "Verifikasi Alumni Baru", "ActiveMenu": "verify_alumni",
		"Success": r.URL.Query().Get("success"),
		"Error":   r.URL.Query().Get("error"),
	}
	renderTemplate(w, "alumni_verify.html", data)
}

func AlumniVerifyApprove(w http.ResponseWriter, r *http.Request) {
	id, _ := strconv.Atoi(r.URL.Query().Get("id"))
	if id == 0 {
		id, _ = strconv.Atoi(r.FormValue("id"))
	}
	
	_, err := database.DB.Exec("UPDATE alumni SET status = 'active', updated_at = CURRENT_TIMESTAMP WHERE id = ?", id)
	if err != nil {
		log.Printf("Approve error: %v", err)
		http.Redirect(w, r, "/dashboard/alumni/verify?error=Gagal+menyetujui+data", 303)
		return
	}

	if r.Header.Get("HX-Request") == "true" {
		w.Header().Set("HX-Redirect", "/dashboard/alumni/verify?success=Data+berhasil+disetujui")
		w.WriteHeader(200)
		return
	}
	http.Redirect(w, r, "/dashboard/alumni/verify?success=Data+berhasil+disetujui", 303)
}

func AlumniVerifyReject(w http.ResponseWriter, r *http.Request) {
	id, _ := strconv.Atoi(r.URL.Query().Get("id"))
	if id == 0 {
		id, _ = strconv.Atoi(r.FormValue("id"))
	}

	var existingFoto *string
	database.DB.QueryRow("SELECT foto_profil FROM alumni WHERE id = ?", id).Scan(&existingFoto)

	_, err := database.DB.Exec("DELETE FROM alumni WHERE id = ?", id)
	if err != nil {
		log.Printf("Reject error: %v", err)
		http.Redirect(w, r, "/dashboard/alumni/verify?error=Gagal+menolak+data", 303)
		return
	}

	if existingFoto != nil && *existingFoto != "" {
		dataDir := os.Getenv("DATA_DIR")
		if dataDir == "" {
			dataDir = "./data"
		}
		os.Remove(filepath.Join(dataDir, "uploads", *existingFoto))
	}

	if r.Header.Get("HX-Request") == "true" {
		w.Header().Set("HX-Redirect", "/dashboard/alumni/verify?success=Data+berhasil+ditolak")
		w.WriteHeader(200)
		return
	}
	http.Redirect(w, r, "/dashboard/alumni/verify?success=Data+berhasil+ditolak", 303)
}

// ==================== Claims / Update Profile Handlers ====================

func AlumniClaimList(w http.ResponseWriter, r *http.Request) {
	user := middleware.GetUser(r)

	rows, err := database.DB.Query(`
		SELECT k.id, k.alumni_id, k.no_wa_verifikasi, k.catatan_pengklaim,
			k.update_no_hp, k.update_email, k.update_domisili,
			k.update_pekerjaan, k.update_instansi, k.update_linkedin,
			k.update_foto_filename, k.status, k.dibuat_pada,
			a.nama_lengkap, a.tahun_lulus,
			a.no_hp, a.email, a.domisili_sekarang, a.pekerjaan,
			a.instansi, a.url_linkedin, a.foto_profil
		FROM klaim_profil k
		JOIN alumni a ON k.alumni_id = a.id
		WHERE k.status = 'pending'
		ORDER BY k.dibuat_pada DESC`)
	if err != nil {
		log.Printf("Query claims error: %v", err)
		http.Error(w, "Database error", 500)
		return
	}
	defer rows.Close()

	var list []models.KlaimProfil
	for rows.Next() {
		var k models.KlaimProfil
		if err := rows.Scan(
			&k.ID, &k.AlumniID, &k.NoWAVerifikasi, &k.CatatanPengklaim,
			&k.UpdateNoHP, &k.UpdateEmail, &k.UpdateDomisili,
			&k.UpdatePekerjaan, &k.UpdateInstansi, &k.UpdateLinkedIn,
			&k.UpdateFotoFilename, &k.Status, &k.DibuatPada,
			&k.AlumniNamaLengkap, &k.AlumniTahunLulus,
			&k.OrigNoHP, &k.OrigEmail, &k.OrigDomisili, &k.OrigPekerjaan,
			&k.OrigInstansi, &k.OrigLinkedIn, &k.OrigFotoProfil,
		); err != nil {
			log.Printf("Scan claim error: %v", err)
			continue
		}
		list = append(list, k)
	}

	data := map[string]interface{}{
		"User":       user,
		"Claims":     list,
		"PageTitle":  "Permintaan Update / Klaim Data",
		"ActiveMenu": "claims",
		"Success":    r.URL.Query().Get("success"),
		"Error":      r.URL.Query().Get("error"),
	}
	renderTemplate(w, "alumni_claims.html", data)
}

func AlumniClaimApprove(w http.ResponseWriter, r *http.Request) {
	claimIDStr := r.FormValue("id")
	claimID, _ := strconv.Atoi(claimIDStr)
	if claimID == 0 {
		http.Redirect(w, r, "/dashboard/alumni/claims?error=ID+klaim+tidak+valid", 303)
		return
	}

	// Fetch claim data
	var k models.KlaimProfil
	err := database.DB.QueryRow(`
		SELECT id, alumni_id, update_no_hp, update_email, update_domisili,
			update_pekerjaan, update_instansi, update_linkedin, update_foto_filename
		FROM klaim_profil WHERE id = ? AND status = 'pending'`, claimID).Scan(
		&k.ID, &k.AlumniID, &k.UpdateNoHP, &k.UpdateEmail, &k.UpdateDomisili,
		&k.UpdatePekerjaan, &k.UpdateInstansi, &k.UpdateLinkedIn, &k.UpdateFotoFilename)
	if err != nil {
		log.Printf("Claim approve fetch error: %v", err)
		http.Redirect(w, r, "/dashboard/alumni/claims?error=Klaim+tidak+ditemukan", 303)
		return
	}

	// Use COALESCE(NULLIF(?, ''), column) to only update non-empty fields
	queryUpdate := `
		UPDATE alumni SET 
			no_hp = COALESCE(NULLIF(?, ''), no_hp),
			email = COALESCE(NULLIF(?, ''), email),
			domisili_sekarang = COALESCE(NULLIF(?, ''), domisili_sekarang),
			pekerjaan = COALESCE(NULLIF(?, ''), pekerjaan),
			instansi = COALESCE(NULLIF(?, ''), instansi),
			url_linkedin = COALESCE(NULLIF(?, ''), url_linkedin),
			foto_profil = COALESCE(NULLIF(?, ''), foto_profil),
			updated_at = CURRENT_TIMESTAMP
		WHERE id = ?`

	_, err = database.DB.Exec(queryUpdate,
		derefStr(k.UpdateNoHP), derefStr(k.UpdateEmail), derefStr(k.UpdateDomisili),
		derefStr(k.UpdatePekerjaan), derefStr(k.UpdateInstansi), derefStr(k.UpdateLinkedIn),
		derefStr(k.UpdateFotoFilename), k.AlumniID)

	if err != nil {
		log.Printf("Claim approve update error: %v", err)
		// If DB update failed and there was a new photo, clean it up
		if k.UpdateFotoFilename != nil && *k.UpdateFotoFilename != "" {
			dataDir := os.Getenv("DATA_DIR")
			if dataDir == "" {
				dataDir = "./data"
			}
			os.Remove(filepath.Join(dataDir, "uploads", *k.UpdateFotoFilename))
		}
		http.Redirect(w, r, "/dashboard/alumni/claims?error=Gagal+menerapkan+perubahan", 303)
		return
	}

	// Mark claim as approved
	database.DB.Exec(`UPDATE klaim_profil SET status = 'approved' WHERE id = ?`, claimID)

	if r.Header.Get("HX-Request") == "true" {
		w.Header().Set("HX-Redirect", "/dashboard/alumni/claims?success=Perubahan+berhasil+diterapkan")
		w.WriteHeader(200)
		return
	}
	http.Redirect(w, r, "/dashboard/alumni/claims?success=Perubahan+berhasil+diterapkan", 303)
}

func AlumniClaimReject(w http.ResponseWriter, r *http.Request) {
	claimIDStr := r.FormValue("id")
	claimID, _ := strconv.Atoi(claimIDStr)
	if claimID == 0 {
		http.Redirect(w, r, "/dashboard/alumni/claims?error=ID+klaim+tidak+valid", 303)
		return
	}

	// Fetch foto filename before rejecting (to clean up orphan file)
	var fotoFilename *string
	database.DB.QueryRow("SELECT update_foto_filename FROM klaim_profil WHERE id = ?", claimID).Scan(&fotoFilename)

	_, err := database.DB.Exec(`UPDATE klaim_profil SET status = 'rejected' WHERE id = ?`, claimID)
	if err != nil {
		log.Printf("Claim reject error: %v", err)
		http.Redirect(w, r, "/dashboard/alumni/claims?error=Gagal+menolak+permintaan", 303)
		return
	}

	// Clean up orphan photo
	if fotoFilename != nil && *fotoFilename != "" {
		dataDir := os.Getenv("DATA_DIR")
		if dataDir == "" {
			dataDir = "./data"
		}
		os.Remove(filepath.Join(dataDir, "uploads", *fotoFilename))
	}

	if r.Header.Get("HX-Request") == "true" {
		w.Header().Set("HX-Redirect", "/dashboard/alumni/claims?success=Permintaan+berhasil+ditolak")
		w.WriteHeader(200)
		return
	}
	http.Redirect(w, r, "/dashboard/alumni/claims?success=Permintaan+berhasil+ditolak", 303)
}

// derefStr safely dereferences a *string, returning "" if nil
func derefStr(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}


