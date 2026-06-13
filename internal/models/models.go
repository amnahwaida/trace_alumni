package models

import (
	"strings"
	"time"
)

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
	Status           string  `json:"status"`
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

type KlaimProfil struct {
	ID                 int     `json:"id"`
	AlumniID           int     `json:"alumni_id"`
	NoWAVerifikasi     string  `json:"no_wa_verifikasi"`
	CatatanPengklaim   *string `json:"catatan_pengklaim"`
	UpdateNoHP         *string `json:"update_no_hp"`
	UpdateEmail        *string `json:"update_email"`
	UpdateDomisili     *string `json:"update_domisili"`
	UpdatePekerjaan    *string `json:"update_pekerjaan"`
	UpdateInstansi     *string `json:"update_instansi"`
	UpdateLinkedIn     *string `json:"update_linkedin"`
	UpdateFotoFilename *string `json:"update_foto_filename"`
	Status             string  `json:"status"`
	DibuatPada         string  `json:"dibuat_pada"`
	// Joined fields
	AlumniNamaLengkap  string  `json:"alumni_nama_lengkap,omitempty"`
	AlumniTahunLulus   int     `json:"alumni_tahun_lulus,omitempty"`
	// Original fields for comparison
	OrigNoHP           *string `json:"orig_no_hp,omitempty"`
	OrigEmail          *string `json:"orig_email,omitempty"`
	OrigDomisili       *string `json:"orig_domisili,omitempty"`
	OrigPekerjaan      *string `json:"orig_pekerjaan,omitempty"`
	OrigInstansi       *string `json:"orig_instansi,omitempty"`
	OrigLinkedIn       *string `json:"orig_linkedin,omitempty"`
	OrigFotoProfil     *string `json:"orig_foto_profil,omitempty"`
}

// MaskedNoWAVerifikasi returns masked WhatsApp number
func (k *KlaimProfil) MaskedNoWAVerifikasi() string {
	if len(k.NoWAVerifikasi) < 8 {
		return "****"
	}
	return k.NoWAVerifikasi[:4] + "-xxxx-" + k.NoWAVerifikasi[len(k.NoWAVerifikasi)-4:]
}

// MaskedOrigNoHP returns masked original phone number
func (k *KlaimProfil) MaskedOrigNoHP() string {
	if k.OrigNoHP == nil || *k.OrigNoHP == "" {
		return "-"
	}
	hp := *k.OrigNoHP
	if len(hp) < 8 {
		return "****"
	}
	return hp[:4] + "-xxxx-" + hp[len(hp)-4:]
}

// MaskedUpdateNoHP returns masked update phone number
func (k *KlaimProfil) MaskedUpdateNoHP() string {
	if k.UpdateNoHP == nil || *k.UpdateNoHP == "" {
		return "-"
	}
	hp := *k.UpdateNoHP
	if len(hp) < 8 {
		return "****"
	}
	return hp[:4] + "-xxxx-" + hp[len(hp)-4:]
}

// MaskedOrigEmail returns masked original email
func (k *KlaimProfil) MaskedOrigEmail() string {
	if k.OrigEmail == nil || *k.OrigEmail == "" {
		return "-"
	}
	email := *k.OrigEmail
	atIdx := strings.Index(email, "@")
	if atIdx <= 2 {
		return "***" + email[atIdx:]
	}
	return email[:2] + "***" + email[atIdx:]
}

// MaskedUpdateEmail returns masked update email
func (k *KlaimProfil) MaskedUpdateEmail() string {
	if k.UpdateEmail == nil || *k.UpdateEmail == "" {
		return "-"
	}
	email := *k.UpdateEmail
	atIdx := strings.Index(email, "@")
	if atIdx <= 2 {
		return "***" + email[atIdx:]
	}
	return email[:2] + "***" + email[atIdx:]
}
