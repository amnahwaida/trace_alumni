package handlers

import (
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"trace-alumni/internal/database"
	"trace-alumni/internal/middleware"

	"github.com/xuri/excelize/v2"
)

var headers = []string{
	"Nama Lengkap",
	"Tahun Lulus",
	"Alamat Asli",
	"Domisili Sekarang",
	"No HP",
	"Email",
	"Tanggal Lahir (YYYY-MM-DD)",
	"Pekerjaan",
	"Instansi",
	"URL LinkedIn",
}

// DownloadTemplate streams the Excel template file
func DownloadTemplate(w http.ResponseWriter, r *http.Request) {
	f := excelize.NewFile()
	defer f.Close()

	sheet := "Template Alumni"
	f.SetSheetName("Sheet1", sheet)

	// Set headers
	for i, h := range headers {
		cell, _ := excelize.CoordinatesToCellName(i+1, 1)
		f.SetCellValue(sheet, cell, h)
	}

	// Set style for header
	style, err := f.NewStyle(&excelize.Style{
		Font: &excelize.Font{Bold: true, Color: "FFFFFF"},
		Fill: excelize.Fill{Type: "pattern", Color: []string{"3B82F6"}, Pattern: 1},
	})
	if err == nil {
		f.SetRowStyle(sheet, 1, 1, style)
	}

	// Add example row
	example := []interface{}{
		"Ahmad Fauzi",
		2020,
		"Ngawi",
		"Surabaya",
		"081234567890",
		"ahmad.fauzi@example.com",
		"2002-05-15",
		"Software Engineer",
		"Tech Corp",
		"https://linkedin.com/in/ahmadfauzi",
	}
	for i, val := range example {
		cell, _ := excelize.CoordinatesToCellName(i+1, 2)
		f.SetCellValue(sheet, cell, val)
	}

	// Adjust column widths
	for i := range headers {
		col, _ := excelize.ColumnNumberToName(i + 1)
		f.SetColWidth(sheet, col, col, 20)
	}

	w.Header().Set("Content-Type", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
	w.Header().Set("Content-Disposition", "attachment; filename=template_alumni.xlsx")

	if err := f.Write(w); err != nil {
		log.Printf("Error writing template: %v", err)
		http.Error(w, "Error generating template", 500)
	}
}

// ExportAlumni streams all alumni in database to an Excel file
func ExportAlumni(w http.ResponseWriter, r *http.Request) {
	rows, err := database.DB.Query(`
		SELECT nama_lengkap, tahun_lulus, alamat_asli, domisili_sekarang, no_hp, email,
			   tanggal_lahir, pekerjaan, instansi, url_linkedin
		FROM alumni ORDER BY tahun_lulus DESC, nama_lengkap ASC
	`)
	if err != nil {
		log.Printf("Export query error: %v", err)
		http.Error(w, "Database error", 500)
		return
	}
	defer rows.Close()

	f := excelize.NewFile()
	defer f.Close()

	sheet := "Data Alumni"
	f.SetSheetName("Sheet1", sheet)

	// Set headers
	for i, h := range headers {
		cell, _ := excelize.CoordinatesToCellName(i+1, 1)
		f.SetCellValue(sheet, cell, h)
	}

	// Style header
	style, err := f.NewStyle(&excelize.Style{
		Font: &excelize.Font{Bold: true, Color: "FFFFFF"},
		Fill: excelize.Fill{Type: "pattern", Color: []string{"1E293B"}, Pattern: 1},
	})
	if err == nil {
		f.SetRowStyle(sheet, 1, 1, style)
	}

	rowIdx := 2
	for rows.Next() {
		var nama, alamat, domisili, noHP, email, tglLahir, pekerjaan, instansi, linkedin sql.NullString
		var tahun int

		err := rows.Scan(&nama, &tahun, &alamat, &domisili, &noHP, &email, &tglLahir, &pekerjaan, &instansi, &linkedin)
		if err != nil {
			log.Printf("Scan error in export: %v", err)
			continue
		}

		f.SetCellValue(sheet, fmt.Sprintf("A%d", rowIdx), nama.String)
		f.SetCellValue(sheet, fmt.Sprintf("B%d", rowIdx), tahun)
		f.SetCellValue(sheet, fmt.Sprintf("C%d", rowIdx), valOrEmpty(alamat))
		f.SetCellValue(sheet, fmt.Sprintf("D%d", rowIdx), valOrEmpty(domisili))
		// If staff, we mask phone and email in export? No, PRD: "Staff: Read + Export. Hanya bisa melihat (search) data alumni dan melakukan Export data (untuk keperluan surat/rekap). Tidak bisa Import, Edit, atau Hapus."
		// Wait, staff needs it for official letters/recap, so we export real data. Super admin / admin also exports real data.
		f.SetCellValue(sheet, fmt.Sprintf("E%d", rowIdx), valOrEmpty(noHP))
		f.SetCellValue(sheet, fmt.Sprintf("F%d", rowIdx), valOrEmpty(email))
		f.SetCellValue(sheet, fmt.Sprintf("G%d", rowIdx), valOrEmpty(tglLahir))
		f.SetCellValue(sheet, fmt.Sprintf("H%d", rowIdx), valOrEmpty(pekerjaan))
		f.SetCellValue(sheet, fmt.Sprintf("I%d", rowIdx), valOrEmpty(instansi))
		f.SetCellValue(sheet, fmt.Sprintf("J%d", rowIdx), valOrEmpty(linkedin))
		rowIdx++
	}

	// Adjust widths
	for i := range headers {
		col, _ := excelize.ColumnNumberToName(i + 1)
		f.SetColWidth(sheet, col, col, 20)
	}

	w.Header().Set("Content-Type", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
	w.Header().Set("Content-Disposition", "attachment; filename=data_alumni.xlsx")

	if err := f.Write(w); err != nil {
		log.Printf("Error writing export: %v", err)
	}
}

// ImportAlumni handles uploading and parsing of Excel file
func ImportAlumni(w http.ResponseWriter, r *http.Request) {
	user := middleware.GetUser(r)
	if user.Role != "super_admin" && user.Role != "admin" {
		http.Error(w, "Akses ditolak", http.StatusForbidden)
		return
	}

	file, _, err := r.FormFile("excel_file")
	if err != nil {
		http.Redirect(w, r, "/dashboard/alumni?error=Gagal+membaca+file", 303)
		return
	}
	defer file.Close()

	// Parse excel from memory directly
	f, err := excelize.OpenReader(file)
	if err != nil {
		http.Redirect(w, r, "/dashboard/alumni?error=Format+file+tidak+valid", 303)
		return
	}
	defer f.Close()

	sheets := f.GetSheetList()
	if len(sheets) == 0 {
		http.Redirect(w, r, "/dashboard/alumni?error=File+Excel+kosong", 303)
		return
	}
	sheet := sheets[0]

	rows, err := f.GetRows(sheet)
	if err != nil || len(rows) < 2 {
		http.Redirect(w, r, "/dashboard/alumni?error=Tidak+ada+data+untuk+diimport", 303)
		return
	}

	type importRow struct {
		Nama         string
		TahunLulus   int
		AlamatAsli   *string
		Domisili     *string
		NoHP         *string
		Email        *string
		TanggalLahir *string
		Pekerjaan    *string
		Instansi     *string
		LinkedIn     *string
	}

	type rowError struct {
		Line int
		Err  string
	}

	var dataToInsert []importRow
	var errors []rowError

	// Find column mapping dynamically from header row
	headerRow := rows[0]
	namaIdx := -1
	tahunIdx := -1
	alamatIdx := -1
	domisiliIdx := -1
	noHPIdx := -1
	emailIdx := -1
	tglLahirIdx := -1
	pekerjaanIdx := -1
	instansiIdx := -1
	linkedinIdx := -1

	for i, cell := range headerRow {
		cleanCell := strings.ToLower(strings.TrimSpace(cell))
		if strings.Contains(cleanCell, "nama") {
			namaIdx = i
		} else if strings.Contains(cleanCell, "tahun") || strings.Contains(cleanCell, "lulus") {
			tahunIdx = i
		} else if strings.Contains(cleanCell, "alamat") {
			alamatIdx = i
		} else if strings.Contains(cleanCell, "domisili") {
			domisiliIdx = i
		} else if strings.Contains(cleanCell, "hp") || strings.Contains(cleanCell, "telepon") || strings.Contains(cleanCell, "phone") || strings.Contains(cleanCell, "kontak") || strings.Contains(cleanCell, "wa") {
			noHPIdx = i
		} else if strings.Contains(cleanCell, "email") || strings.Contains(cleanCell, "surel") {
			emailIdx = i
		} else if strings.Contains(cleanCell, "lahir") || strings.Contains(cleanCell, "tanggal") {
			tglLahirIdx = i
		} else if strings.Contains(cleanCell, "pekerjaan") || strings.Contains(cleanCell, "kerja") {
			pekerjaanIdx = i
		} else if strings.Contains(cleanCell, "instansi") || strings.Contains(cleanCell, "perusahaan") || strings.Contains(cleanCell, "kantor") {
			instansiIdx = i
		} else if strings.Contains(cleanCell, "linkedin") {
			linkedinIdx = i
		}
	}

	// Fallback to default indices if headers are not found
	if namaIdx == -1 { namaIdx = 0 }
	if tahunIdx == -1 { tahunIdx = 1 }
	if alamatIdx == -1 { alamatIdx = 2 }
	if domisiliIdx == -1 { domisiliIdx = 3 }
	if noHPIdx == -1 { noHPIdx = 4 }
	if emailIdx == -1 { emailIdx = 5 }
	if tglLahirIdx == -1 { tglLahirIdx = 6 }
	if pekerjaanIdx == -1 { pekerjaanIdx = 7 }
	if instansiIdx == -1 { instansiIdx = 8 }
	if linkedinIdx == -1 { linkedinIdx = 9 }

	for idx, row := range rows {
		if idx == 0 {
			// Skip header
			continue
		}

		line := idx + 1

		// Check if the row is completely empty (all cells are blank)
		isEmptyRow := true
		for _, cell := range row {
			if strings.TrimSpace(cell) != "" {
				isEmptyRow = false
				break
			}
		}
		if isEmptyRow {
			continue
		}

		var nama, tahunStr, alamat, domisili, noHP, email, tglLahir, pekerjaan, instansi, linkedin string

		if namaIdx < len(row) { nama = strings.TrimSpace(row[namaIdx]) }
		if tahunIdx < len(row) { tahunStr = strings.TrimSpace(row[tahunIdx]) }
		if alamatIdx < len(row) { alamat = strings.TrimSpace(row[alamatIdx]) }
		if domisiliIdx < len(row) { domisili = strings.TrimSpace(row[domisiliIdx]) }
		if noHPIdx < len(row) { noHP = strings.TrimSpace(row[noHPIdx]) }
		if emailIdx < len(row) { email = strings.TrimSpace(row[emailIdx]) }
		if tglLahirIdx < len(row) { tglLahir = strings.TrimSpace(row[tglLahirIdx]) }
		if pekerjaanIdx < len(row) { pekerjaan = strings.TrimSpace(row[pekerjaanIdx]) }
		if instansiIdx < len(row) { instansi = strings.TrimSpace(row[instansiIdx]) }
		if linkedinIdx < len(row) { linkedin = strings.TrimSpace(row[linkedinIdx]) }

		if nama == "" {
			errors = append(errors, rowError{Line: line, Err: "Nama Lengkap wajib diisi"})
			continue
		}

		if tahunStr == "" {
			errors = append(errors, rowError{Line: line, Err: "Tahun Lulus wajib diisi"})
			continue
		}

		// Handle float strings like "2020.0" and extract first 4-digit number
		tahunStr = strings.TrimSpace(tahunStr)
		var yearFound string
		for i := 0; i <= len(tahunStr)-4; i++ {
			sub := tahunStr[i : i+4]
			if _, err := strconv.Atoi(sub); err == nil {
				yearFound = sub
				break
			}
		}
		if yearFound != "" {
			tahunStr = yearFound
		}

		tahun, err := strconv.Atoi(tahunStr)
		if err != nil || tahun < 1900 || tahun > time.Now().Year()+1 {
			log.Printf("[Excel Import Debug] Row %d: tahunStr='%s' (len=%d, bytes=%v), err=%v, parsedYear=%d", line, tahunStr, len(tahunStr), []byte(tahunStr), err, tahun)
			errors = append(errors, rowError{Line: line, Err: "Tahun Lulus harus berupa angka tahun yang valid"})
			continue
		}

		// Clean No HP from scientific notation or decimal format
		if noHP != "" {
			if strings.Contains(strings.ToLower(noHP), "e+") {
				if fVal, err := strconv.ParseFloat(noHP, 64); err == nil {
					noHP = fmt.Sprintf("%.0f", fVal)
				}
			} else if strings.Contains(noHP, ".") {
				parts := strings.Split(noHP, ".")
				noHP = parts[0]
			}
			noHP = strings.ReplaceAll(noHP, " ", "")
			noHP = strings.ReplaceAll(noHP, "-", "")
			noHP = strings.ReplaceAll(noHP, "+", "")
		}

		// Validate date if not empty
		var tglPtr *string
		if tglLahir != "" {
			var parsedTime time.Time
			var parseSuccess bool

			// Try formats:
			formats := []string{
				"2006-01-02",
				"02-01-2006",
				"2006/01/02",
				"02/01/2006",
				"01/02/2006",
				"01-02-2006",
				"2006-01-02 15:04:05",
				"2006-01-02 15:04",
				"2006/01/02 15:04:05",
				"2006/01/02 15:04",
				"02-01-2006 15:04:05",
				"02/01/2006 15:04:05",
				"01/02/2006 15:04:05",
				time.RFC3339,
				"2006-01-02T15:04:05",
				"02 Jan 2006",
				"2 Jan 2006",
				"02 January 2006",
				"2 January 2006",
			}
			for _, fmtStr := range formats {
				if t, err := time.Parse(fmtStr, tglLahir); err == nil {
					parsedTime = t
					parseSuccess = true
					break
				}
			}

			// Try as Excel serial float number
			if !parseSuccess {
				if floatVal, err := strconv.ParseFloat(tglLahir, 64); err == nil {
					if floatVal > 1.0 {
						if t, err := excelize.ExcelDateToTime(floatVal, false); err == nil {
							parsedTime = t
							parseSuccess = true
						}
					}
				}
			}

			if !parseSuccess {
				errors = append(errors, rowError{Line: line, Err: "Format Tanggal Lahir tidak valid (gunakan YYYY-MM-DD atau DD-MM-YYYY)"})
				continue
			}

			fmtDate := parsedTime.Format("2006-01-02")
			tglPtr = &fmtDate
		}

		dataToInsert = append(dataToInsert, importRow{
			Nama:         nama,
			TahunLulus:   tahun,
			AlamatAsli:   ns(alamat),
			Domisili:     ns(domisili),
			NoHP:         ns(noHP),
			Email:        ns(email),
			TanggalLahir: tglPtr,
			Pekerjaan:    ns(pekerjaan),
			Instansi:     ns(instansi),
			LinkedIn:     ns(linkedin),
		})
	}

	// If there are errors, generate an error report file and stream it immediately
	if len(errors) > 0 {
		log.Printf("[Excel Import] Validation errors found on %d rows: %+v", len(errors), errors)
		errFile := excelize.NewFile()
		defer errFile.Close()

		errSheet := "Laporan Error"
		errFile.SetSheetName("Sheet1", errSheet)

		errFile.SetCellValue(errSheet, "A1", "Baris Excel")
		errFile.SetCellValue(errSheet, "B1", "Deskripsi Error")

		style, _ := errFile.NewStyle(&excelize.Style{
			Font: &excelize.Font{Bold: true, Color: "FFFFFF"},
			Fill: excelize.Fill{Type: "pattern", Color: []string{"EF4444"}, Pattern: 1},
		})
		errFile.SetRowStyle(errSheet, 1, 1, style)

		for i, e := range errors {
			rowNum := i + 2
			errFile.SetCellValue(errSheet, fmt.Sprintf("A%d", rowNum), e.Line)
			errFile.SetCellValue(errSheet, fmt.Sprintf("B%d", rowNum), e.Err)
		}

		errFile.SetColWidth(errSheet, "A", "A", 15)
		errFile.SetColWidth(errSheet, "B", "B", 50)

		w.Header().Set("Content-Type", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
		w.Header().Set("Content-Disposition", "attachment; filename=error_report_import.xlsx")
		errFile.Write(w)
		return
	}

	// Insert all in a single transaction
	tx, err := database.DB.Begin()
	if err != nil {
		log.Printf("Transaction start error: %v", err)
		http.Redirect(w, r, "/dashboard/alumni?error=Gagal+memulai+transaksi", 303)
		return
	}
	defer tx.Rollback()

	stmt, err := tx.Prepare(`
		INSERT INTO alumni (nama_lengkap, tahun_lulus, alamat_asli, domisili_sekarang,
			no_hp, email, tanggal_lahir, pekerjaan, instansi, url_linkedin)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`)
	if err != nil {
		log.Printf("Prepare statement error: %v", err)
		http.Redirect(w, r, "/dashboard/alumni?error=Gagal+menyiapkan+sistem", 303)
		return
	}
	defer stmt.Close()

	for _, d := range dataToInsert {
		_, err := stmt.Exec(d.Nama, d.TahunLulus, d.AlamatAsli, d.Domisili,
			d.NoHP, d.Email, d.TanggalLahir, d.Pekerjaan, d.Instansi, d.LinkedIn)
		if err != nil {
			log.Printf("Import exec error: %v", err)
			http.Redirect(w, r, "/dashboard/alumni?error=Gagal+memasukkan+data+ke+database", 303)
			return
		}
	}

	if err := tx.Commit(); err != nil {
		log.Printf("Transaction commit error: %v", err)
		http.Redirect(w, r, "/dashboard/alumni?error=Gagal+menyimpan+transaksi", 303)
		return
	}

	http.Redirect(w, r, fmt.Sprintf("/dashboard/alumni?success=Berhasil+mengimport+%d+data+alumni", len(dataToInsert)), 303)
}

func valOrEmpty(s sql.NullString) string {
	if s.Valid {
		return s.String
	}
	return ""
}

// ExportPapan streams all announcements to an Excel file
func ExportPapan(w http.ResponseWriter, r *http.Request) {
	rows, err := database.DB.Query(`
		SELECT p.judul, p.deskripsi, p.kategori, p.link_eksternal, u.username, p.dibuat_pada, p.is_active, p.aktif_sampai
		FROM info_papan p
		LEFT JOIN users u ON p.dibuat_oleh = u.id
		ORDER BY p.id DESC
	`)
	if err != nil {
		log.Printf("Export papan query error: %v", err)
		http.Error(w, "Database error", 500)
		return
	}
	defer rows.Close()

	f := excelize.NewFile()
	defer f.Close()

	sheet := "Data Pengumuman"
	f.SetSheetName("Sheet1", sheet)

	papanHeaders := []string{
		"Judul Pengumuman",
		"Deskripsi",
		"Kategori",
		"Link Eksternal",
		"Dibuat Oleh",
		"Tanggal Dibuat",
		"Status",
		"Aktif Sampai",
	}

	// Set headers
	for i, h := range papanHeaders {
		cell, _ := excelize.CoordinatesToCellName(i+1, 1)
		f.SetCellValue(sheet, cell, h)
	}

	// Style header
	style, err := f.NewStyle(&excelize.Style{
		Font: &excelize.Font{Bold: true, Color: "FFFFFF"},
		Fill: excelize.Fill{Type: "pattern", Color: []string{"1E293B"}, Pattern: 1},
	})
	if err == nil {
		f.SetRowStyle(sheet, 1, 1, style)
	}

	rowIdx := 2
	for rows.Next() {
		var judul, deskripsi, kategori, dibuatPada string
		var linkPtr, aktifSampaiPtr *string
		var username sql.NullString
		var isActive bool

		err := rows.Scan(&judul, &deskripsi, &kategori, &linkPtr, &username, &dibuatPada, &isActive, &aktifSampaiPtr)
		if err != nil {
			log.Printf("Scan error in export papan: %v", err)
			continue
		}

		link := ""
		if linkPtr != nil {
			link = *linkPtr
		}

		aktifSampai := "Tidak dibatasi"
		if aktifSampaiPtr != nil && *aktifSampaiPtr != "" {
			aktifSampai = *aktifSampaiPtr
		}

		createdByName := "Sistem"
		if username.Valid {
			createdByName = username.String
		}

		katLabel := kategori
		switch kategori {
		case "umum":
			katLabel = "Umum"
		case "loker":
			katLabel = "Lowongan Kerja"
		case "beasiswa":
			katLabel = "Beasiswa"
		case "reuni":
			katLabel = "Reuni"
		}

		statusLabel := "Nonaktif"
		if isActive {
			statusLabel = "Aktif"
		}

		f.SetCellValue(sheet, fmt.Sprintf("A%d", rowIdx), judul)
		f.SetCellValue(sheet, fmt.Sprintf("B%d", rowIdx), deskripsi)
		f.SetCellValue(sheet, fmt.Sprintf("C%d", rowIdx), katLabel)
		f.SetCellValue(sheet, fmt.Sprintf("D%d", rowIdx), link)
		f.SetCellValue(sheet, fmt.Sprintf("E%d", rowIdx), createdByName)
		f.SetCellValue(sheet, fmt.Sprintf("F%d", rowIdx), dibuatPada)
		f.SetCellValue(sheet, fmt.Sprintf("G%d", rowIdx), statusLabel)
		f.SetCellValue(sheet, fmt.Sprintf("H%d", rowIdx), aktifSampai)
		rowIdx++
	}

	// Adjust widths
	for i := range papanHeaders {
		col, _ := excelize.ColumnNumberToName(i + 1)
		f.SetColWidth(sheet, col, col, 25)
	}

	w.Header().Set("Content-Type", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
	w.Header().Set("Content-Disposition", "attachment; filename=data_pengumuman.xlsx")

	if err := f.Write(w); err != nil {
		log.Printf("Error writing papan export: %v", err)
	}
}
