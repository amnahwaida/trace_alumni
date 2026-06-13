package handlers

import (
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"trace-alumni/internal/database"
	"trace-alumni/internal/models"
)

func LandingPage(w http.ResponseWriter, r *http.Request) {
	data := map[string]interface{}{
		"PageTitle": "Rekam Jejak Alumni - SMAS Muhammadiyah 1 Ngawi",
	}
	renderTemplate(w, "landing.html", data)
}

func PublicSearch(w http.ResponseWriter, r *http.Request) {
	search := strings.TrimSpace(r.URL.Query().Get("q"))
	if search == "" {
		renderTemplate(w, "search_results.html", map[string]interface{}{"Alumni": nil})
		return
	}
	pat := "%" + search + "%"
	rows, err := database.DB.Query(`
		SELECT id,nama_lengkap,domisili_sekarang,no_hp,email,tahun_lulus,pekerjaan,instansi,foto_profil
		FROM alumni WHERE status = 'active' AND (nama_lengkap LIKE ? OR CAST(tahun_lulus AS TEXT) LIKE ?)
		ORDER BY tahun_lulus DESC LIMIT 20`, pat, pat)
	if err != nil {
		log.Printf("Search error: %v", err)
		renderTemplate(w, "search_results.html", map[string]interface{}{"Alumni": nil, "Error": "Terjadi kesalahan"})
		return
	}
	defer rows.Close()

	var list []models.Alumni
	for rows.Next() {
		var a models.Alumni
		rows.Scan(&a.ID, &a.NamaLengkap, &a.DomisiliSekarang, &a.NoHP, &a.Email,
			&a.TahunLulus, &a.Pekerjaan, &a.Instansi, &a.FotoProfil)
		list = append(list, a)
	}
	renderTemplate(w, "search_results.html", map[string]interface{}{"Alumni": list, "Query": search})
}

func PublicTambahData(w http.ResponseWriter, r *http.Request) {
	data := map[string]interface{}{
		"PageTitle": "Pendaftaran Alumni Baru",
		"Success":   r.URL.Query().Get("success"),
		"Error":     r.URL.Query().Get("error"),
	}
	renderTemplate(w, "public_tambah_data.html", data)
}

func PublicTambahDataSubmit(w http.ResponseWriter, r *http.Request) {
	err := r.ParseMultipartForm(5 << 20) // 5 MB max
	if err != nil {
		http.Redirect(w, r, "/tambah-data?error=Gagal+memproses+form", 303)
		return
	}

	// Sanitize and trim inputs
	nama := strings.TrimSpace(r.FormValue("nama_lengkap"))
	tls := strings.TrimSpace(r.FormValue("tahun_lulus"))
	noHp := strings.TrimSpace(r.FormValue("no_hp"))
	email := strings.TrimSpace(r.FormValue("email"))
	alamatAsli := strings.TrimSpace(r.FormValue("alamat_asli"))
	domisili := strings.TrimSpace(r.FormValue("domisili_sekarang"))
	pekerjaan := strings.TrimSpace(r.FormValue("pekerjaan"))
	instansi := strings.TrimSpace(r.FormValue("instansi"))
	linkedin := strings.TrimSpace(r.FormValue("url_linkedin"))

	if nama == "" || tls == "" {
		http.Redirect(w, r, "/tambah-data?error=Nama+Lengkap+dan+Tahun+Lulus+wajib+diisi", 303)
		return
	}

	tahunLulus, err := strconv.Atoi(tls)
	if err != nil || tahunLulus < 1900 || tahunLulus > time.Now().Year()+1 {
		http.Redirect(w, r, "/tambah-data?error=Tahun+Lulus+tidak+valid", 303)
		return
	}

	// Process photo upload using ProcessPhoto helper
	fotoFileName, err := ProcessPhoto(r, "foto_profil")
	if err != nil {
		http.Redirect(w, r, "/tambah-data?error="+strings.ReplaceAll(err.Error(), " ", "+"), 303)
		return
	}

	var fotoPtr *string
	if fotoFileName != "" {
		fotoPtr = &fotoFileName
	}

	// Database insertion with status='pending'
	_, err = database.DB.Exec(`
		INSERT INTO alumni (
			nama_lengkap, alamat_asli, domisili_sekarang, no_hp, email, 
			tahun_lulus, pekerjaan, instansi, url_linkedin, foto_profil, status
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, 'pending')`,
		nama, ns(alamatAsli), ns(domisili), ns(noHp), ns(email),
		tahunLulus, ns(pekerjaan), ns(instansi), ns(linkedin), fotoPtr)

	if err != nil {
		log.Printf("Public submission DB error: %v", err)
		if fotoFileName != "" {
			dataDir := os.Getenv("DATA_DIR")
			if dataDir == "" {
				dataDir = "./data"
			}
			os.Remove(filepath.Join(dataDir, "uploads", fotoFileName))
		}
		http.Redirect(w, r, "/tambah-data?error=Gagal+menyimpan+data+ke+database", 303)
		return
	}

	http.Redirect(w, r, "/tambah-data?success=Data+berhasil+dikirim.+Silahkan+tunggu+proses+verifikasi+oleh+pihak+sekolah.", 303)
}

