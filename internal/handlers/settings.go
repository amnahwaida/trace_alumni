package handlers

import (
	"fmt"
	"image"
	"image/png"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"trace-alumni/internal/database"
	"trace-alumni/internal/middleware"

	"github.com/disintegration/imaging"
	"github.com/google/uuid"
)

// SettingsList renders the settings customization page
func SettingsList(w http.ResponseWriter, r *http.Request) {
	user := middleware.GetUser(r)

	data := map[string]interface{}{
		"User":       user,
		"PageTitle":  "Pengaturan Sistem",
		"ActiveMenu": "settings",
		"Success":    r.URL.Query().Get("success"),
		"Error":      r.URL.Query().Get("error"),
	}
	renderTemplate(w, "settings.html", data)
}

// SettingsSave handles updates to the school name, project credit, logo, and favicon
func SettingsSave(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseMultipartForm(5 << 20); err != nil {
		http.Redirect(w, r, "/dashboard/settings?error=Gagal+mengunggah+file", 303)
		return
	}

	schoolName := strings.TrimSpace(r.FormValue("school_name"))
	projectCredit := strings.TrimSpace(r.FormValue("project_credit"))
	papanDescription := strings.TrimSpace(r.FormValue("papan_description"))
	pwaAppName := strings.TrimSpace(r.FormValue("pwa_app_name"))

	if schoolName == "" || projectCredit == "" || papanDescription == "" || pwaAppName == "" {
		http.Redirect(w, r, "/dashboard/settings?error=Nama+sekolah,+credit,+deskripsi+papan+dan+nama+PWA+tidak+boleh+kosong", 303)
		return
	}

	// Update text configurations in database
	_, _ = database.DB.Exec("UPDATE settings SET value = ? WHERE key = 'school_name'", schoolName)
	_, _ = database.DB.Exec("UPDATE settings SET value = ? WHERE key = 'project_credit'", projectCredit)
	_, _ = database.DB.Exec("UPDATE settings SET value = ? WHERE key = 'papan_description'", papanDescription)
	_, _ = database.DB.Exec("UPDATE settings SET value = ? WHERE key = 'pwa_app_name'", pwaAppName)

	// Process logo file
	logoPath, err := processUploadedLogo(r, "logo")
	if err != nil {
		http.Redirect(w, r, "/dashboard/settings?error="+err.Error(), 303)
		return
	}
	if logoPath != "" {
		// Remove old logo file if it exists
		var oldLogoPath string
		_ = database.DB.QueryRow("SELECT value FROM settings WHERE key = 'logo_path'").Scan(&oldLogoPath)
		if oldLogoPath != "" {
			dataDir := os.Getenv("DATA_DIR")
			if dataDir == "" {
				dataDir = "./data"
			}
			filename := filepath.Base(oldLogoPath)
			oldFile := filepath.Join(dataDir, "uploads", filename)
			_ = os.Remove(oldFile)
		}
		_, _ = database.DB.Exec("UPDATE settings SET value = ? WHERE key = 'logo_path'", logoPath)
	}

	// Process favicon file
	faviconPath, err := processUploadedFavicon(r, "favicon")
	if err != nil {
		http.Redirect(w, r, "/dashboard/settings?error="+err.Error(), 303)
		return
	}
	if faviconPath != "" {
		// Remove old favicon file if it exists
		var oldFaviconPath string
		_ = database.DB.QueryRow("SELECT value FROM settings WHERE key = 'favicon_path'").Scan(&oldFaviconPath)
		if oldFaviconPath != "" {
			dataDir := os.Getenv("DATA_DIR")
			if dataDir == "" {
				dataDir = "./data"
			}
			filename := filepath.Base(oldFaviconPath)
			oldFile := filepath.Join(dataDir, "uploads", filename)
			_ = os.Remove(oldFile)
		}
		_, _ = database.DB.Exec("UPDATE settings SET value = ? WHERE key = 'favicon_path'", faviconPath)
	}

	// Process PWA icon file
	pwaIconPath, err := processUploadedPwaIcon(r, "pwa_icon")
	if err != nil {
		http.Redirect(w, r, "/dashboard/settings?error="+err.Error(), 303)
		return
	}
	if pwaIconPath != "" {
		// Remove old PWA icon file if it exists
		var oldPwaIconPath string
		_ = database.DB.QueryRow("SELECT value FROM settings WHERE key = 'pwa_icon_path'").Scan(&oldPwaIconPath)
		if oldPwaIconPath != "" {
			dataDir := os.Getenv("DATA_DIR")
			if dataDir == "" {
				dataDir = "./data"
			}
			filename := filepath.Base(oldPwaIconPath)
			oldFile := filepath.Join(dataDir, "uploads", filename)
			_ = os.Remove(oldFile)
		}
		_, _ = database.DB.Exec("UPDATE settings SET value = ? WHERE key = 'pwa_icon_path'", pwaIconPath)
	}

	http.Redirect(w, r, "/dashboard/settings?success=Pengaturan+berhasil+disimpan", 303)
}

// SettingsReset handles resetting a configuration key to its default value
func SettingsReset(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	key := r.FormValue("key")
	var defaultValue string
	var isFile bool

	switch key {
	case "school_name":
		defaultValue = "SMAS Muhammadiyah 1 Ngawi"
	case "project_credit":
		defaultValue = "© 2026 SMAS Muhammadiyah 1 Ngawi. Sistem Rekam Jejak Alumni."
	case "papan_description":
		defaultValue = "Temukan info lowongan kerja, beasiswa, agenda reuni, dan informasi penting lainnya."
	case "logo_path":
		defaultValue = ""
		isFile = true
	case "favicon_path":
		defaultValue = ""
		isFile = true
	case "pwa_app_name":
		defaultValue = "Alumni Tracker"
	case "pwa_icon_path":
		defaultValue = ""
		isFile = true
	default:
		http.Redirect(w, r, "/dashboard/settings?error=Kunci+pengaturan+tidak+valid", 303)
		return
	}

	if isFile {
		// Get current file path to delete it
		var oldPath string
		_ = database.DB.QueryRow("SELECT value FROM settings WHERE key = ?", key).Scan(&oldPath)
		if oldPath != "" {
			dataDir := os.Getenv("DATA_DIR")
			if dataDir == "" {
				dataDir = "./data"
			}
			filename := filepath.Base(oldPath)
			oldFile := filepath.Join(dataDir, "uploads", filename)
			_ = os.Remove(oldFile)
		}
	}

	_, err := database.DB.Exec("UPDATE settings SET value = ? WHERE key = ?", defaultValue, key)
	if err != nil {
		http.Redirect(w, r, "/dashboard/settings?error=Gagal+meriset+pengaturan", 303)
		return
	}

	http.Redirect(w, r, "/dashboard/settings?success=Pengaturan+berhasil+direset+ke+default", 303)
}

func processUploadedLogo(r *http.Request, fieldName string) (string, error) {
	file, header, err := r.FormFile(fieldName)
	if err != nil {
		if err == http.ErrMissingFile {
			return "", nil
		}
		return "", err
	}
	defer file.Close()

	// Limit size to 2MB
	if header.Size > 2*1024*1024 {
		return "", fmt.Errorf("ukuran file logo melebihi 2MB")
	}

	img, _, err := image.Decode(file)
	if err != nil {
		return "", fmt.Errorf("gagal memproses file logo: %w", err)
	}

	// Fit to square logo (400x400px)
	resizedImg := imaging.Fit(img, 400, 400, imaging.Lanczos)

	dataDir := os.Getenv("DATA_DIR")
	if dataDir == "" {
		dataDir = "./data"
	}
	uploadsDir := filepath.Join(dataDir, "uploads")

	fileName := "logo_" + uuid.New().String()[:8] + ".png"
	filePath := filepath.Join(uploadsDir, fileName)

	out, err := os.Create(filePath)
	if err != nil {
		return "", fmt.Errorf("gagal menyimpan logo: %w", err)
	}
	defer out.Close()

	err = png.Encode(out, resizedImg)
	if err != nil {
		os.Remove(filePath)
		return "", fmt.Errorf("gagal encode logo: %w", err)
	}

	return "/uploads/" + fileName, nil
}

func processUploadedFavicon(r *http.Request, fieldName string) (string, error) {
	file, header, err := r.FormFile(fieldName)
	if err != nil {
		if err == http.ErrMissingFile {
			return "", nil
		}
		return "", err
	}
	defer file.Close()

	// Limit size to 1MB
	if header.Size > 1*1024*1024 {
		return "", fmt.Errorf("ukuran file favicon melebihi 1MB")
	}

	img, _, err := image.Decode(file)
	if err != nil {
		return "", fmt.Errorf("gagal memproses file favicon: %w", err)
	}

	// Standard favicon size: 48x48 px
	resizedImg := imaging.Fit(img, 48, 48, imaging.Lanczos)

	dataDir := os.Getenv("DATA_DIR")
	if dataDir == "" {
		dataDir = "./data"
	}
	uploadsDir := filepath.Join(dataDir, "uploads")

	fileName := "favicon_" + uuid.New().String()[:8] + ".png"
	filePath := filepath.Join(uploadsDir, fileName)

	out, err := os.Create(filePath)
	if err != nil {
		return "", fmt.Errorf("gagal menyimpan favicon: %w", err)
	}
	defer out.Close()

	err = png.Encode(out, resizedImg)
	if err != nil {
		os.Remove(filePath)
		return "", fmt.Errorf("gagal encode favicon: %w", err)
	}

	return "/uploads/" + fileName, nil
}

func processUploadedPwaIcon(r *http.Request, fieldName string) (string, error) {
	file, header, err := r.FormFile(fieldName)
	if err != nil {
		if err == http.ErrMissingFile {
			return "", nil
		}
		return "", err
	}
	defer file.Close()

	// Limit size to 3MB
	if header.Size > 3*1024*1024 {
		return "", fmt.Errorf("ukuran file icon PWA melebihi 3MB")
	}

	img, _, err := image.Decode(file)
	if err != nil {
		return "", fmt.Errorf("gagal memproses file icon PWA: %w", err)
	}

	// PWA icon size: 512x512 px
	resizedImg := imaging.Fit(img, 512, 512, imaging.Lanczos)

	dataDir := os.Getenv("DATA_DIR")
	if dataDir == "" {
		dataDir = "./data"
	}
	uploadsDir := filepath.Join(dataDir, "uploads")

	fileName := "pwa_icon_" + uuid.New().String()[:8] + ".png"
	filePath := filepath.Join(uploadsDir, fileName)

	out, err := os.Create(filePath)
	if err != nil {
		return "", fmt.Errorf("gagal menyimpan icon PWA: %w", err)
	}
	defer out.Close()

	err = png.Encode(out, resizedImg)
	if err != nil {
		os.Remove(filePath)
		return "", fmt.Errorf("gagal encode icon PWA: %w", err)
	}

	return "/uploads/" + fileName, nil
}
