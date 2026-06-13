

---

# 📄 Product Requirements Document (PRD) - Tambahan
**Modul:** Sistem Klaim & Perbarui Profil Alumni (Claim & Update System)  
**Versi:** 2.0 (Final & Simplified)  
**Tanggal:** 13 Juni 2026  
**Konteks:** Menggantikan semua mekanisme verifikasi otomatis/OTP. Fokus pada *crowdsourcing* data oleh alumni, verifikasi manual oleh Admin via WhatsApp, dan beban server yang tetap ultra-minimal (STB 1 GB).

---

## 1. 🎯 Latar Belakang & Masalah
1. **Data Awal Minim:** Mayoritas data arsip kertas hanya berisi **Nama Lengkap** dan **Tahun Lulus**. 
2. **Data Dinamis:** Alumni yang datanya sudah ada perlu memperbarui informasi (pindah kerja, ganti nomor HP, pindah domisili).
3. **Kendala Perangkat:** STB hanya memiliki sisa penyimpanan 1 GB. Sistem **TIDAK BOLEH** memiliki fitur login, password, dashboard alumni, atau integrasi API eksternal yang berat (seperti Email/Telegram Bot).

**Solusi:** Menerapkan sistem formulir **"Klaim & Lengkapi / Perbarui Data"** terpadu. Alumni mengajukan perubahan, dan Admin memverifikasinya secara manual via WhatsApp sebelum perubahan diterapkan ke database utama.

---

## 2. 💡 Konsep Solusi: Unified Claim & Update Workflow
Sistem ini menggunakan **satu mekanisme dan satu tabel database** untuk dua skenario:
- **Skenario A (Data Kosong/Minim):** Formulir muncul dalam keadaan kosong. Alumni mengisi data dari awal.
- **Skenario B (Data Sudah Ada):** Formulir muncul dalam keadaan *pre-filled* (terisi data lama). Alumni hanya mengedit bagian yang berubah.

Kedua skenario ini akan masuk ke antrean persetujuan Admin yang sama.

---

## 3. 🔄 Alur Kerja (End-to-End Workflow)

### **Fase A: Aksi Alumni (Publik)**
1. Alumni mencari namanya di Landing Page. Muncul kartu hasil pencarian **"Teaser"** (Nama, Tahun Lulus, Pekerjaan, Kota Domisili). Data kontak disamarkan di sisi server (`08xx-xxxx-xxxx`).
2. Di bawah kartu, muncul tombol dinamis:
   - Jika data minim: **[📝 Ini Profil Saya? Klaim & Lengkapi Data]**
   - Jika data lengkap: **[📝 Data Saya Berubah? Ajukan Perbarui Data]**
3. Saat diklik, muncul Modal/Popup Formulir:
   - **Nomor WhatsApp Aktif** (Wajib, untuk verifikasi Admin).
   - **Usulan Data Baru:** No HP, Email, Domisili, Pekerjaan, Instansi, URL LinkedIn, Upload Foto. *(Form ini akan terisi data lama jika ada, atau kosong jika tidak ada)*.
4. Alumni klik "Kirim Permintaan". Muncul pesan: *"Permintaan berhasil dikirim! Admin akan memverifikasi melalui WhatsApp dalam 1x24 jam."*

### **Fase B: Aksi Admin (Dashboard)**
1. Admin membuka menu **"Permintaan Update/Klaim Data"** di dashboard (ada indikator jumlah `pending`).
2. Admin melihat detail perbandingan: *"Ahmad Fauzi (2015) mengajukan perubahan. Pekerjaan Lama: 'Guru' → Usulan Baru: 'Software Engineer'."*
3. **SOP Verifikasi:** Admin menyalin nomor WA dan mengirim pesan template: *"Halo [Nama], ini Admin Website Alumni. Kami menerima permintaan update data Anda. Mohon konfirmasi dengan membalas 'Benar'."*
4. Setelah alumni membalas "Benar", Admin mengklik tombol **[✅ Setujui & Terapkan Perubahan]**.

### **Fase C: Eksekusi Sistem (Otomatis)**
1. Sistem Golang secara otomatis melakukan `UPDATE` pada tabel `alumni` utama, menimpa data lama dengan data usulan baru.
2. Status klaim diubah menjadi `approved`.
3. Admin secara manual menginformasikan ke alumni via WA: *"Data sudah diupdate. Silakan cek kembali di website."*

---

## 4. ⚙️ Spesifikasi Teknis & Database

### 4.1. Skema Database SQLite (Tabel Klaim)
```sql
CREATE TABLE klaim_profil (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    alumni_id INTEGER NOT NULL,          -- Merujuk ke id di tabel alumni
    
    -- Data untuk verifikasi oleh Admin
    no_wa_verifikasi TEXT NOT NULL,      
    catatan_pengklaim TEXT,              
    
    -- DATA USULAN BARU DARI ALUMNI (Boleh NULL/kosong)
    update_no_hp TEXT,
    update_email TEXT,
    update_domisili TEXT,
    update_pekerjaan TEXT,
    update_instansi TEXT,
    update_linkedin TEXT,
    update_foto_filename TEXT,           -- Nama file foto baru (jika diupload)
    
    -- Status
    status TEXT DEFAULT 'pending' CHECK(status IN ('pending', 'approved', 'rejected')),
    dibuat_pada DATETIME DEFAULT CURRENT_TIMESTAMP
);
```

### 4.2. Logika Backend Golang (Saat Admin Menyetujui)
Menggunakan trik SQL `COALESCE` agar kita tidak perlu menulis logika `if-else` yang rumit di Go untuk mengecek field mana yang kosong.

```go
// Di dalam adminApproveClaimHandler
queryUpdate := `
    UPDATE alumni SET 
        no_hp = COALESCE(NULLIF(?, ''), no_hp),
        email = COALESCE(NULLIF(?, ''), email),
        domisili_sekarang = COALESCE(NULLIF(?, ''), domisili_sekarang),
        pekerjaan = COALESCE(NULLIF(?, ''), pekerjaan),
        instansi = COALESCE(NULLIF(?, ''), instansi),
        url_linkedin = COALESCE(NULLIF(?, ''), url_linkedin),
        foto_profil = COALESCE(NULLIF(?, ''), foto_profil),
        last_updated = CURRENT_TIMESTAMP
    WHERE id = ?`

_, err = db.Exec(queryUpdate, 
    claim.NoHP, claim.Email, claim.Domisili, claim.Pekerjaan, 
    claim.Instansi, claim.LinkedIn, claim.Foto, claim.AlumniID)

// Jika sukses, update status klaim
db.Exec(`UPDATE klaim_profil SET status = 'approved' WHERE id = ?`, claimID)
```
*Catatan: `COALESCE(NULLIF(?, ''), nama_kolom)` berarti: "Jika data usulan baru diisi (tidak kosong), gunakan data baru. Jika kosong, pertahankan data lama yang ada di database."*

### 4.3. Penanganan Foto (Wajib Ketat untuk 1 GB)
- Batas upload di form: **Maksimal 2 MB**.
- Backend **HARUS** langsung me-resize gambar menjadi **200x200 px** dan mengompresinya ke **JPEG Quality 65%** sebelum disimpan ke folder `/data/uploads`.
- Jika proses `UPDATE` database gagal, file foto yang terlanjur di-upload **HARUS** dihapus dari disk untuk mencegah *orphan files*.

---

## 5. 🛡️ Keamanan & Anti-Spam

1. **Server-Side Masking (Wajib):** Endpoint pencarian **harus** mengecek cookie `alumni_verified_access` (jika ada). Jika tidak ada, kirim data yang sudah di-mask (`maskPhone`, `maskEmail`). Jangan pernah mengirim data asli dan menyembunyikannya via CSS.
2. **Rate Limiting:** Batasi endpoint formulir klaim maksimal **3 permintaan per IP per hari** untuk mencegah spam.
3. **Validasi Input:** Pastikan `no_wa_verifikasi` diawali `08` dan panjangnya valid. Sanitasi semua input teks untuk mencegah XSS.
4. **SOP Admin:** Wajib ada konfirmasi balasan WhatsApp dari alumni sebelum tombol "Setujui" diklik di dashboard.

---

## 6. 📋 Definition of Done (DoD)
Fitur ini dianggap selesai jika:
- [ ] Formulir klaim/update dapat diakses, menampilkan data lama (jika ada), dan menyimpan usulan baru ke tabel `klaim_profil`.
- [ ] Upload foto pada formulir klaim berhasil di-resize dan dikompresi oleh backend sebelum disimpan.
- [ ] Admin dapat melihat daftar permintaan, membandingkan data lama vs baru, dan mengklik "Setujui".
- [ ] Query `UPDATE` dengan `COALESCE` berhasil menimpa data lama hanya pada field yang diisi, tanpa menghapus data lama yang tidak diubah.
- [ ] Tidak ada penambahan beban penyimpanan yang signifikan (hanya teks pendek + foto terkompresi).

---

Dokumen ini sekarang **100% final, sederhana, dan siap dieksekusi**. Semua kompleksitas yang tidak perlu (OTP, Bot, Login) telah dibuang, menyisakan inti fitur yang paling bernilai dan paling aman untuk STB 1 GB.
