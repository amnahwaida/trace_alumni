package models

import "time"

type User struct {
	ID           int       `json:"id"`
	Username     string    `json:"username"`
	PasswordHash string    `json:"-"`
	Role         string    `json:"role"`
	CreatedAt    time.Time `json:"created_at"`
}

type Alumni struct {
	ID               int     `json:"id"`
	NamaLengkap      string  `json:"nama_lengkap"`
	AlamatAsli       *string `json:"alamat_asli"`
	DomisiliSekarang *string `json:"domisili_sekarang"`
	NoHP             *string `json:"no_hp"`
	Email            *string `json:"email"`
	TahunLulus       int     `json:"tahun_lulus"`
	TanggalLahir     *string `json:"tanggal_lahir"`
	Pekerjaan        *string `json:"pekerjaan"`
	Instansi         *string `json:"instansi"`
	URLLinkedIn      *string `json:"url_linkedin"`
	FotoProfil       *string `json:"foto_profil"`
	CreatedAt        string  `json:"created_at"`
	UpdatedAt        string  `json:"updated_at"`
}

// MaskedNoHP returns masked phone number for public view
func (a *Alumni) MaskedNoHP() string {
	if a.NoHP == nil || *a.NoHP == "" {
		return "-"
	}
	hp := *a.NoHP
	if len(hp) < 8 {
		return "****"
	}
	return hp[:4] + "-xxxx-" + hp[len(hp)-4:]
}

// MaskedEmail returns masked email for public view
func (a *Alumni) MaskedEmail() string {
	if a.Email == nil || *a.Email == "" {
		return "-"
	}
	email := *a.Email
	atIdx := -1
	for i, c := range email {
		if c == '@' {
			atIdx = i
			break
		}
	}
	if atIdx <= 2 {
		return "***" + email[atIdx:]
	}
	return email[:2] + "***" + email[atIdx:]
}

type InfoPapan struct {
	ID             int     `json:"id"`
	Judul          string  `json:"judul"`
	Deskripsi      string  `json:"deskripsi"`
	LinkEksternal  *string `json:"link_eksternal"`
	Kategori       string  `json:"kategori"`
	DibuatOleh     *int    `json:"dibuat_oleh"`
	DibuatPada     string  `json:"dibuat_pada"`
	IsActive       bool    `json:"is_active"`
	AktifSampai    *string `json:"aktif_sampai"`
	// Joined field
	DibuatOlehNama string  `json:"dibuat_oleh_nama,omitempty"`
}

func (p *InfoPapan) IsExpired() bool {
	if p.AktifSampai == nil || *p.AktifSampai == "" {
		return false
	}
	t, err := time.Parse("2006-01-02", *p.AktifSampai)
	if err != nil {
		t, err = time.Parse("2006-01-02 15:04:05", *p.AktifSampai)
		if err != nil {
			t, err = time.Parse(time.RFC3339, *p.AktifSampai)
			if err != nil {
				return false
			}
		}
	}
	expirationDate := time.Date(t.Year(), t.Month(), t.Day(), 23, 59, 59, 999999999, time.Local)
	return time.Now().After(expirationDate)
}

type Session struct {
	ID        string    `json:"id"`
	UserID    int       `json:"user_id"`
	ExpiresAt time.Time `json:"expires_at"`
	CreatedAt time.Time `json:"created_at"`
}
