Berikut adalah dokumen **PRD Tambahan** yang mengemas strategi akuisisi data alumni secara komprehensif, tetap selaras dengan batasan teknis STB 1 GB dan prinsip "Tanpa Role Alumni".

Dokumen ini dirancang sebagai lampiran resmi dari `prd.md` utama.

---

# 📄 Product Requirements Document (PRD) - Tambahan
**Modul:** Akuisisi & Verifikasi Data Alumni Baru (Unrecorded Alumni)  
**Versi:** 1.1  
**Tanggal:** 13 Juni 2026  
**Konteks:** Melengkapi `prd.md` utama untuk mengatasi tantangan digitalisasi data dari luar arsip sekolah.

---

## 1. 🎯 Latar Belakang & Tujuan
Banyak data alumni tidak tercatat dalam arsip fisik sekolah. Modul ini bertujuan menciptakan **mekanisme inbound (tarik-minat)** yang memungkinkan alumni mendaftarkan atau memperbarui data mereka secara mandiri, **tanpa** memerlukan sistem login/akun yang kompleks, sehingga tetap aman dan ringan untuk STB 1 GB.

---

## 2. 📢 Strategi Akuisisi Data (Non-Teknis)
Keberhasilan modul ini 70% bergantung pada strategi operasional sekolah. Berikut adalah pilar pemicu dan saluran distribusi:

### 2.1. Pemicu (The Hook)
*   **Faktor Nostalgia:** Menampilkan statistik di landing page (misal: *"Angkatan 2015: 45 dari 120 alumni sudah terdaftar. Cek namamu!"*).
*   **Manfaat Networking:** Menekankan bahwa data yang lengkap (terutama Pekerjaan & LinkedIn) akan memudahkan alumni saling terhubung untuk peluang karir atau bisnis.
*   **Akses Informasi:** Narasi bahwa alumni terverifikasi akan mendapat prioritas informasi Reuni dan Beasiswa.

### 2.2. Saluran Distribusi (Channel)
*   **Duta Angkatan:** Menghubungi 1-2 alumni paling aktif per angkatan (mantan Ketua OSIS/Ketua Kelas) untuk menyebarkan link di Grup WhatsApp Angkatan.
*   **QR Code Fisik:** Menempelkan stiker QR Code yang mengarah ke website di lokasi strategis (Masjid Muhammadiyah, Gerbang Sekolah, Ruang Guru).
*   **Media Sosial Sekolah:** Postingan berkala di Instagram/Facebook resmi sekolah dengan ajakan bertindak (Call to Action) yang jelas.

---

## 3. ⚙️ Spesifikasi Fungsional (Alur Kerja)

### 3.1. Formulir "Klaim / Tambah Data" (Publik)
*   **Lokasi:** Halaman publik `/tambah-data` (diakses via tombol di Landing Page: *"Namamu belum ada? Daftarkan di sini"*).
*   **Field Input:** Nama Lengkap, Tahun Lulus, No HP, Email, Domisili Sekarang, Pekerjaan, Instansi, URL LinkedIn.
*   **Logika Backend:**
    1. Sistem menerima data.
    2. Sistem menyimpan data ke database dengan status default **`pending`**.
    3. Data dengan status `pending` **TIDAK** muncul di hasil pencarian publik (Landing Page).
    4. Menampilkan pesan sukses: *"Terima kasih! Data Anda sedang diverifikasi oleh Admin sekolah."*

### 3.2. Dashboard Verifikasi (Admin / Super Admin)
*   **Menu Baru:** Tab "Verifikasi Data Baru" di dashboard admin.
*   **Tampilan:** Hanya menampilkan daftar alumni dengan `status = 'pending'`.
*   **Aksi:** 
    *   **[Approve]:** Mengubah status menjadi `active`. Data langsung muncul di pencarian publik.
    *   **[Tolak/Hapus]:** Menghapus data jika terindikasi spam atau data palsu.
    *   *(Opsional)* **[Merge]:** Jika Admin mendeteksi ini adalah update dari data lama yang sudah ada, Admin bisa menyalin data baru ke profil yang lama, lalu menghapus entri `pending` ini.

### 3.3. Fallback: Google Form (Opsi Cadangan)
*   Jika tim Admin merasa kewalahan memoderasi form website, tombol "Daftarkan di sini" dapat diarahkan ke **Google Form** resmi sekolah.
*   Admin cukup mengunduh CSV dari Google Sheets secara berkala (misal: bulanan) dan menggunakan fitur **Import Excel** yang sudah ada di sistem STB.

---

## 4. 🗄️ Spesifikasi Teknis & Perubahan Database

Untuk mendukung fitur ini, diperlukan penyesuaian minimal pada skema database dan backend:

### 4.1. Update Skema SQLite
```sql
-- Tambahkan kolom status untuk memisahkan data yang belum diverifikasi
ALTER TABLE alumni ADD COLUMN status TEXT DEFAULT 'pending' CHECK(status IN ('pending', 'active', 'rejected'));

-- Tambahkan kolom untuk melacak kapan terakhir kali data diperbarui
ALTER TABLE alumni ADD COLUMN last_updated DATETIME DEFAULT CURRENT_TIMESTAMP;

-- Update index agar pencarian publik hanya mengambil data 'active'
-- (Index lama tetap berguna, tapi query harus ditambah WHERE status = 'active')
```

### 4.2. Penyesuaian Query Pencarian Publik
Semua query di Landing Page (Search Bar) **wajib** menambahkan filter:
```sql
SELECT nama_lengkap, tahun_lulus, domisili_sekarang, pekerjaan 
FROM alumni 
WHERE status = 'active' AND (nama_lengkap LIKE ? OR tahun_lulus = ?)
```

### 4.3. Endpoint Baru (Backend Golang)
*   `GET /tambah-data`: Merender halaman formulir HTML sederhana.
*   `POST /api/alumni/submit`: 
    *   Menerima payload form.
    *   Melakukan sanitasi input (mencegah XSS).
    *   Insert ke DB dengan `status = 'pending'`.
    *   Return JSON success.

---

## 5. 📅 Rencana Operasional & Moderasi (Action Plan)

Agar sistem tidak menjadi tempat penampungan data sampah, terapkan SOP berikut:

1. **Minggu 1 (Internal):** Admin menginput data arsip kertas sebagai "benih" (target minimal 30% per angkatan).
2. **Minggu 2 (Soft Launch):** Website live. Admin menghubungi "Duta Angkatan" via WA untuk mulai menyebarkan link.
3. **Minggu 3 (Public Launch):** Pasang QR Code di sekolah dan posting di medsos. Aktifkan fitur `/tambah-data`.
4. **Rutin (Maintenance):** Admin meluangkan waktu **15 menit setiap hari Jumat pagi** untuk membuka dashboard, memverifikasi data `pending` (bisa dengan WA singkat ke nomor yang dicantumkan), dan klik "Approve".

---

## 6. ✅ Definition of Done (DoD) untuk Modul Ini
Fitur akuisisi data dianggap selesai jika:
- [ ] Halaman `/tambah-data` dapat diakses publik dan responsif di HP.
- [ ] Data yang disubmit melalui form tersebut **TIDAK** muncul di pencarian landing page sebelum di-approve.
- [ ] Admin dapat melihat daftar `pending` di dashboard dan berhasil mengubah statusnya menjadi `active` (yang kemudian langsung muncul di pencarian publik).
- [ ] Tidak ada penambahan beban penyimpanan yang signifikan (hanya teks).
- [ ] Input form telah disanitasi untuk mencegah injeksi kode berbahaya.

---

Dokumen `prd_tambahan.md` ini sekarang melengkapi visi teknis dan operasional proyek Anda. 

