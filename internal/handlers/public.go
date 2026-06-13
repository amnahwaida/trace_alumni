package handlers

import (
	"encoding/json"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"trace-alumni/internal/database"
	"trace-alumni/internal/models"
)

var (
	claimRateLimits = make(map[string][]time.Time)
	rateLimitMu     sync.Mutex
)

func getClientIP(r *http.Request) string {
	xff := r.Header.Get("X-Forwarded-For")
	if xff != "" {
		parts := strings.Split(xff, ",")
		return strings.TrimSpace(parts[0])
	}
	ipPort := r.RemoteAddr
	if idx := strings.LastIndex(ipPort, ":"); idx != -1 {
		return ipPort[:idx]
	}
	return ipPort
}

func isRateLimited(ip string) bool {
	rateLimitMu.Lock()
	defer rateLimitMu.Unlock()
	now := time.Now()
	times, exists := claimRateLimits[ip]
	if !exists {
		claimRateLimits[ip] = []time.Time{now}
		return false
	}
	
	// Filter times in the last 24 hours
	var activeTimes []time.Time
	for _, t := range times {
		if now.Sub(t) < 24*time.Hour {
			activeTimes = append(activeTimes, t)
		}
	}
	
	if len(activeTimes) >= 3 {
		claimRateLimits[ip] = activeTimes
		return true
	}
	
	activeTimes = append(activeTimes, now)
	claimRateLimits[ip] = activeTimes
	return false
}

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

func PublicClaimSubmit(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		json.NewEncoder(w).Encode(map[string]interface{}{"success": false, "error": "Method tidak diperbolehkan"})
		return
	}

	ip := getClientIP(r)
	if isRateLimited(ip) {
		w.WriteHeader(http.StatusTooManyRequests)
		json.NewEncoder(w).Encode(map[string]interface{}{"success": false, "error": "Batas pengiriman (3 kali per hari per IP) terlampaui."})
		return
	}

	err := r.ParseMultipartForm(5 << 20) // 5 MB
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]interface{}{"success": false, "error": "Gagal memproses form"})
		return
	}

	alumniIDStr := strings.TrimSpace(r.FormValue("alumni_id"))
	noWA := strings.TrimSpace(r.FormValue("no_wa_verifikasi"))
	catatan := strings.TrimSpace(r.FormValue("catatan_pengklaim"))
	
	upNoHP := strings.TrimSpace(r.FormValue("update_no_hp"))
	upEmail := strings.TrimSpace(r.FormValue("update_email"))
	upDomisili := strings.TrimSpace(r.FormValue("update_domisili"))
	upPekerjaan := strings.TrimSpace(r.FormValue("update_pekerjaan"))
	upInstansi := strings.TrimSpace(r.FormValue("update_instansi"))
	upLinkedIn := strings.TrimSpace(r.FormValue("update_linkedin"))

	if alumniIDStr == "" || noWA == "" {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]interface{}{"success": false, "error": "ID Alumni dan Nomor WhatsApp wajib diisi"})
		return
	}

	alumniID, err := strconv.Atoi(alumniIDStr)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]interface{}{"success": false, "error": "ID Alumni tidak valid"})
		return
	}

	// Validate WhatsApp
	if !strings.HasPrefix(noWA, "08") || len(noWA) < 10 || len(noWA) > 15 {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]interface{}{"success": false, "error": "Nomor WhatsApp tidak valid (harus diawali '08' dengan panjang 10-15 digit)"})
		return
	}

	// Validate alumni exists
	var exists bool
	err = database.DB.QueryRow("SELECT EXISTS(SELECT 1 FROM alumni WHERE id = ? AND status = 'active')", alumniID).Scan(&exists)
	if err != nil || !exists {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]interface{}{"success": false, "error": "Alumni tidak ditemukan atau tidak aktif"})
		return
	}

	// Process image upload
	fotoFileName, err := ProcessPhoto(r, "update_foto")
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]interface{}{"success": false, "error": err.Error()})
		return
	}

	var fotoPtr *string
	if fotoFileName != "" {
		fotoPtr = &fotoFileName
	}

	_, err = database.DB.Exec(`
		INSERT INTO klaim_profil (
			alumni_id, no_wa_verifikasi, catatan_pengklaim,
			update_no_hp, update_email, update_domisili,
			update_pekerjaan, update_instansi, update_linkedin,
			update_foto_filename, status
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, 'pending')`,
		alumniID, noWA, ns(catatan),
		ns(upNoHP), ns(upEmail), ns(upDomisili),
		ns(upPekerjaan), ns(upInstansi), ns(upLinkedIn),
		fotoPtr)

	if err != nil {
		log.Printf("Insert claim DB error: %v", err)
		if fotoFileName != "" {
			dataDir := os.Getenv("DATA_DIR")
			if dataDir == "" {
				dataDir = "./data"
			}
			os.Remove(filepath.Join(dataDir, "uploads", fotoFileName))
		}
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]interface{}{"success": false, "error": "Gagal menyimpan permintaan update ke database"})
		return
	}

	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"message": "Permintaan berhasil dikirim! Admin akan memverifikasi melalui WhatsApp dalam 1x24 jam.",
	})
}

// PwaManifest returns a dynamic manifest.json with custom PWA app name and PWA icon
func PwaManifest(w http.ResponseWriter, r *http.Request) {
	settings := getGlobalSettings()
	appName := settings["pwa_app_name"]
	if appName == "" {
		appName = "Alumni Tracker"
	}

	icon192Path := settings["pwa_icon_path"]
	if icon192Path == "" {
		icon192Path = "/static/icon-192.png"
	}
	icon512Path := settings["pwa_icon_path"]
	if icon512Path == "" {
		icon512Path = "/static/icon-512.png"
	}

	manifest := map[string]interface{}{
		"name":             appName,
		"short_name":       appName,
		"description":      "Sistem Informasi & Penelusuran Alumni",
		"start_url":        "/",
		"display":          "standalone",
		"background_color": "#0f172a",
		"theme_color":      "#0f172a",
		"orientation":      "portrait-primary",
		"icons": []map[string]string{
			{
				"src":     icon192Path,
				"sizes":   "192x192",
				"type":    "image/png",
				"purpose": "any maskable",
			},
			{
				"src":     icon512Path,
				"sizes":   "512x512",
				"type":    "image/png",
				"purpose": "any maskable",
			},
		},
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(manifest)
}
