package handlers

import (
	"log"
	"net/http"
	"strings"

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
		FROM alumni WHERE nama_lengkap LIKE ? OR CAST(tahun_lulus AS TEXT) LIKE ?
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
