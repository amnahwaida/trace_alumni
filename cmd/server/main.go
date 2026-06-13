package main

import (
	"log"
	"net/http"
	"os"

	"trace-alumni/internal/database"
	"trace-alumni/internal/handlers"
	"trace-alumni/internal/middleware"

	"github.com/go-chi/chi/v5"
	chimw "github.com/go-chi/chi/v5/middleware"
)

func main() {
	dataDir := os.Getenv("DATA_DIR")
	if dataDir == "" {
		dataDir = "./data"
	}
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	// Init database
	if err := database.InitDB(dataDir); err != nil {
		log.Fatal("DB init failed:", err)
	}
	defer database.Close()

	if err := database.RunMigrations(); err != nil {
		log.Fatal("Migration failed:", err)
	}
	if err := database.SeedSuperAdmin(); err != nil {
		log.Fatal("Seed failed:", err)
	}

	// Init templates
	if err := handlers.InitTemplates("templates"); err != nil {
		log.Fatal("Template init failed:", err)
	}

	r := chi.NewRouter()
	r.Use(chimw.Logger)
	r.Use(chimw.Recoverer)
	r.Use(chimw.Compress(5))

	// Static files
	fs := http.FileServer(http.Dir("static"))
	r.Handle("/static/*", http.StripPrefix("/static/", fs))

	// Uploads (photo files)
	uploads := http.FileServer(http.Dir(dataDir + "/uploads"))
	r.Handle("/uploads/*", http.StripPrefix("/uploads/", uploads))

	// PWA routes
	r.Get("/sw.js", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/javascript")
		http.ServeFile(w, r, "static/sw.js")
	})
	r.Get("/manifest.json", handlers.PwaManifest)

	// Public routes
	r.Get("/", handlers.LandingPage)
	r.Get("/search", handlers.PublicSearch)
	r.Get("/info", handlers.PublicPapan)
	r.Get("/tambah-data", handlers.PublicTambahData)
	r.Post("/api/alumni/submit", handlers.PublicTambahDataSubmit)
	r.Post("/api/alumni/claim", handlers.PublicClaimSubmit)
	r.Get("/login", handlers.LoginPage)
	r.Post("/login", handlers.LoginPost)
	r.Get("/logout", handlers.Logout)

	// Protected dashboard routes
	r.Route("/dashboard", func(r chi.Router) {
		r.Use(middleware.AuthMiddleware)

		r.Get("/", handlers.Dashboard)

		// Self change password
		r.Get("/change-password", handlers.ChangePasswordPage)
		r.Post("/change-password", handlers.ChangePasswordPost)

		// Alumni CRUD (admin, super_admin can write; staff can read)
		r.Get("/alumni", handlers.AlumniList)
		r.Get("/alumni/export", handlers.ExportAlumni)
		r.With(middleware.RequireRole("super_admin", "admin")).Get("/alumni/create", handlers.AlumniCreate)
		r.With(middleware.RequireRole("super_admin", "admin")).Post("/alumni/store", handlers.AlumniStore)
		r.With(middleware.RequireRole("super_admin", "admin")).Get("/alumni/edit", handlers.AlumniEdit)
		r.With(middleware.RequireRole("super_admin", "admin")).Post("/alumni/update", handlers.AlumniUpdate)
		r.With(middleware.RequireRole("super_admin", "admin")).Post("/alumni/delete", handlers.AlumniDelete)
		r.With(middleware.RequireRole("super_admin", "admin")).Get("/alumni/template", handlers.DownloadTemplate)
		r.With(middleware.RequireRole("super_admin", "admin")).Post("/alumni/import", handlers.ImportAlumni)
		
		// Verification Tab (New Alumni Registration)
		r.With(middleware.RequireRole("super_admin", "admin")).Get("/alumni/verify", handlers.AlumniVerifyList)
		r.With(middleware.RequireRole("super_admin", "admin")).Post("/alumni/verify/approve", handlers.AlumniVerifyApprove)
		r.With(middleware.RequireRole("super_admin", "admin")).Post("/alumni/verify/reject", handlers.AlumniVerifyReject)

		// Claims / Update Data Requests
		r.With(middleware.RequireRole("super_admin", "admin")).Get("/alumni/claims", handlers.AlumniClaimList)
		r.With(middleware.RequireRole("super_admin", "admin")).Post("/alumni/claims/approve", handlers.AlumniClaimApprove)
		r.With(middleware.RequireRole("super_admin", "admin")).Post("/alumni/claims/reject", handlers.AlumniClaimReject)

		// Papan Pengumuman CRUD
		r.Get("/papan", handlers.PapanList)
		r.Get("/papan/export", handlers.ExportPapan)
		r.With(middleware.RequireRole("super_admin", "admin")).Get("/papan/create", handlers.PapanCreate)
		r.With(middleware.RequireRole("super_admin", "admin")).Post("/papan/store", handlers.PapanStore)
		r.With(middleware.RequireRole("super_admin", "admin")).Get("/papan/edit", handlers.PapanEdit)
		r.With(middleware.RequireRole("super_admin", "admin")).Post("/papan/update", handlers.PapanUpdate)
		r.With(middleware.RequireRole("super_admin", "admin")).Post("/papan/delete", handlers.PapanDelete)
		r.With(middleware.RequireRole("super_admin", "admin")).Post("/papan/toggle-global", handlers.PapanToggleGlobal)
		r.With(middleware.RequireRole("super_admin", "admin")).Post("/papan/toggle-active", handlers.PapanToggleActive)

		// User management (super_admin only)
		r.Route("/users", func(r chi.Router) {
			r.Use(middleware.RequireRole("super_admin"))
			r.Get("/", handlers.UserList)
			r.Get("/create", handlers.UserCreate)
			r.Post("/store", handlers.UserStore)
			r.Post("/delete", handlers.UserDelete)
			r.Get("/reset-password", handlers.UserResetPasswordPage)
			r.Post("/reset-password", handlers.UserResetPasswordPost)
		})

		// Settings (super_admin only)
		r.With(middleware.RequireRole("super_admin")).Get("/settings", handlers.SettingsList)
		r.With(middleware.RequireRole("super_admin")).Post("/settings/save", handlers.SettingsSave)
		r.With(middleware.RequireRole("super_admin")).Post("/settings/reset", handlers.SettingsReset)
	})

	log.Printf("🚀 Server starting on :%s", port)
	log.Fatal(http.ListenAndServe(":"+port, r))
}
