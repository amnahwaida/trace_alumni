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

	if schoolName == "" || projectCredit == "" {
		http.Redirect(w, r, "/dashboard/settings?error=Nama+sekolah+dan+credit+tidak+boleh+kosong", 303)
		return
	}

	// Update text configurations in database
	_, _ = database.DB.Exec("UPDATE settings SET value = ? WHERE key = 'school_name'", schoolName)
	_, _ = database.DB.Exec("UPDATE settings SET value = ? WHERE key = 'project_credit'", projectCredit)

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

	http.Redirect(w, r, "/dashboard/settings?success=Pengaturan+berhasil+disimpan", 303)
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

	// Fit to standard header sizes (maximum width: 400px, height: 120px) to keep size minimal
	resizedImg := imaging.Fit(img, 400, 120, imaging.Lanczos)

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
