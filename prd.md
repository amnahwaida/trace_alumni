Berikut adalah dokumen **Product Requirements Document (PRD)** yang komprehensif, terstruktur, dan disesuaikan secara ketat dengan batasan perangkat keras (STB 1 GB) serta keputusan arsitektur yang telah kita sepakati.

Dokumen ini akan menjadi "kitab suci" (single source of truth) selama proses pengembangan.

---

# 📄 Product Requirements Document (PRD)
**Nama Proyek:** Website Rekam Jejak Alumni SMAS Muhammadiyah 1 Ngawi  
**Versi:** 1.0  
**Tanggal:** 10 Juni 2026  
**Target Perangkat:** STB Armbian Linux (Sisa Penyimpanan: 1 GB)

---

## 1. 🎯 Ringkasan Eksekutif
Membangun sistem digitalisasi data alumni yang ringan, cepat, dan aman untuk SMAS Muhammadiyah 1 Ngawi. Sistem ini bertujuan untuk memindahkan data dari arsip kertas ke database digital, menyediakan fitur pencarian publik, papan informasi, serta manajemen data yang efisien dengan **footprint penyimpanan yang sangat minimal** (< 100 MB untuk ~1000 data alumni).

---

## 2. ⚠️ Batasan & Asumsi Kritis (Wajib Dipatuhi)
1. **Penyimpanan Maksimum:** Total penggunaan disk oleh Docker image, database, dan file upload **tidak boleh melebihi 200-300 MB** untuk menyisakan ruang aman bagi OS Armbian.
2. **Tanpa Role Alumni:** Tidak ada fitur registrasi, login, atau dashboard untuk alumni. Akses publik bersifat *read-only* (dengan penyamaran data sensitif).
3. **Tanpa Database Server Terpisah:** Menggunakan SQLite (pure Go) dalam satu container untuk menghemat RAM dan storage.
4. **No Heavy Frontend Build:** Tidak ada Node.js, Webpack, atau Vite di STB. Frontend menggunakan Go Templates + HTMX + Tailwind CSS (CDN atau embedded).
5. **In-Memory Processing:** File Excel (Import/Export) dan gambar diproses langsung di memori (RAM), **tidak pernah disimpan sementara (temp) ke disk**.

---

## 3. 👥 Peran Pengguna (User Roles)
| Role | Hak Akses | Deskripsi |
| :--- | :--- | :--- |
| **Super Admin** | Full Access | Mengelola semua data, Import/Export, Papan Pengumuman, dan **Manajemen User** (tambah/hapus Admin/Staff). |
| **Admin** | Manage Data | CRUD data alumni, Import/Export Excel, Kelola Papan Pengumuman. Tidak bisa mengelola user lain. |
| **Staff** | Read + Export | Hanya bisa melihat (search) data alumni dan melakukan **Export** data (untuk keperluan surat/rekap). Tidak bisa Import, Edit, atau Hapus. |
| **Publik (Alumni)** | Read Only (Limited) | Bisa mengakses Landing Page (Search) dan Papan Pengumuman. Data sensitif (No HP, Email) disamarkan. |

---

## 4. 🛠️ Spesifikasi Fungsional (Functional Requirements)

### 4.1. Landing Page (Publik)
- **Search Bar:** Pencarian real-time (dengan debounce) berdasarkan Nama Lengkap atau Tahun Lulus.
- **Hasil Pencarian:** Menampilkan Nama, Tahun Lulus, Domisili, dan Pekerjaan.
- **Data Masking:** Nomor HP dan Email ditampilkan sebagian (misal: `08xx-xxxx-1234`) atau disembunyikan total untuk privasi.
- **Navigasi:** Link ke "Papan Pengumuman" dan "Login Admin".

### 4.2. Manajemen Data Alumni (Dashboard Admin/Super Admin)
- **CRUD:** Tambah, Lihat, Edit, Hapus data alumni.
- **Upload Foto Profil:** 
  - Maksimal ukuran file: **2 MB**.
  - **Auto-Processing:** Backend wajib me-resize gambar menjadi maksimal **200x200 px** dan mengompresinya ke format **JPEG dengan Quality 65%** sebelum disimpan ke disk.
- **Field Data:** Nama Lengkap, Alamat Asli, Domisili Sekarang, No HP, Email, Tahun Lulus, Tanggal Lahir, Pekerjaan, Instansi, URL LinkedIn, Foto Profil.

### 4.3. Import & Export Excel (Admin/Super Admin)
- **Template:** Tersedia tombol download template Excel standar.
- **Import:** 
  - Upload file `.xlsx`.
  - Validasi format (Tanggal, Tahun Lulus harus angka).
  - Proses *batch insert* ke SQLite.
  - Jika ada error, generate laporan error untuk diunduh (tanpa menyimpan file upload di disk).
- **Export/Backup:** Generate file `.xlsx` secara *on-the-fly* dari database dan langsung di-stream ke browser untuk diunduh.

### 4.4. Papan Pengumuman (Publik Read, Admin Write)
- **Tampilan Publik:** Daftar pengumuman diurutkan dari yang terbaru.
- **Kategori:** Loker, Beasiswa, Reuni, Umum (dengan badge warna berbeda).
- **Input Admin:** Hanya menerima **Judul (Teks)**, **Deskripsi (Teks)**, dan **Link Eksternal (HTTPS)**.
- **Validasi Link:** Backend wajib menolak input jika link tidak diawali dengan `https://` (mencegah XSS/malicious links). **Dilarang upload file/gambar ke server.**

---

## 5. 🗄️ Model Data (SQLite Schema)

```sql
-- Tabel Users (Untuk Login Admin/Staff)
CREATE TABLE users (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    username TEXT UNIQUE NOT NULL,
    password_hash TEXT NOT NULL, -- bcrypt
    role TEXT CHECK(role IN ('super_admin', 'admin', 'staff')) NOT NULL,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

-- Tabel Alumni
CREATE TABLE alumni (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    nama_lengkap TEXT NOT NULL,
    alamat_asli TEXT,
    domisili_sekarang TEXT,
    no_hp TEXT,
    email TEXT,
    tahun_lulus INTEGER NOT NULL,
    tanggal_lahir DATE,
    pekerjaan TEXT,
    instansi TEXT,
    url_linkedin TEXT,
    foto_profil TEXT, -- Menyimpan nama file (misal: uuid.jpg), bukan path lengkap
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
);
CREATE INDEX idx_alumni_search ON alumni(nama_lengkap, tahun_lulus);

-- Tabel Papan Pengumuman
CREATE TABLE info_papan (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    judul TEXT NOT NULL,
    deskripsi TEXT NOT NULL,
    link_eksternal TEXT, -- Wajib https://
    kategori TEXT DEFAULT 'umum' CHECK(kategori IN ('loker', 'beasiswa', 'reuni', 'umum')),
    dibuat_oleh INTEGER REFERENCES users(id),
    dibuat_pada DATETIME DEFAULT CURRENT_TIMESTAMP
);
```

---

## 6. ⚙️ Spesifikasi Teknis & Arsitektur

### 6.1. Tech Stack
- **Backend:** Golang 1.22+ 
- **Router:** `go-chi/chi` (Ringan, standar library compatible)
- **Database:** `modernc.org/sqlite` (Pure Go, no CGO required, sangat ringan)
- **Excel Processing:** `github.com/xuri/excelize/v2`
- **Image Processing:** `github.com/disintegration/imaging`
- **Frontend:** Go `html/template` + `HTMX` (untuk interaksi tanpa reload) + `Tailwind CSS` (via CDN atau di-embed).

### 6.2. Strategi Docker (Ultra-Lightweight)
- **Multi-stage Build:** 
  - Stage 1: `golang:alpine` untuk compile.
  - Stage 2: `scratch` atau `alpine:latest` sebagai final image. Hanya menyalin binary hasil compile dan folder `templates`.
- **Target Image Size:** < 30 MB.
- **Log Rotation:** Dikonfigurasi di `docker-compose.yml` (max 10MB per file, max 3 files).

### 6.3. Strategi Penyimpanan (1GB Survival Guide)
1. **Volume Mapping:** Hanya mount `/app/data` (berisi `alumni.db` dan folder `uploads`).
2. **Orphan File Prevention:** Jika proses insert ke database gagal, file foto yang terlanjur di-upload wajib dihapus dari folder `uploads`.
3. **Backup Routine:** Script `cron` mingguan di host Armbian untuk mengompres folder `/app/data` ke `.tar.gz`, memindahkannya ke penyimpanan eksternal (Flashdisk/Cloud), lalu menghapus file backup lama di STB (`find ... -mtime +30 -delete`).

---

## 7. 🚀 Roadmap Pengembangan

### Fase 1: Fondasi & Core (Minggu 1-2)
- [ ] Setup struktur proyek Golang & Dockerfile multi-stage.
- [ ] Implementasi database SQLite & migrasi schema.
- [ ] Fitur Auth (Login/Logout) untuk Super Admin, Admin, Staff.
- [ ] CRUD Data Alumni dasar.

### Fase 2: Optimasi & Fitur Kunci (Minggu 3-4)
- [ ] Implementasi Upload Foto dengan **Auto-Resize & Kompresi**.
- [ ] Fitur Import & Export Excel (In-Memory processing).
- [ ] Landing Page dengan Search Bar (HTMX) & Data Masking.

### Fase 3: Pelengkap & Deployment (Minggu 5)
- [ ] Fitur Papan Pengumuman (CRUD Admin, Tampilan Publik).
- [ ] Uji coba beban (load testing sederhana) dan cek penggunaan disk (`docker system df`).
- [ ] Deployment ke STB Armbian & setup cronjob backup.

---

## 8. 📋 Definition of Done (DoD)
Sebuah fitur dianggap selesai jika:
1. Kode telah di-review dan tidak ada error compile.
2. Fitur berjalan di dalam container Docker.
3. **Tidak ada file temporary yang tertinggal di disk** setelah proses Import/Upload.
4. Ukuran binary dan image Docker tetap di bawah batas yang ditentukan.
5. Data sensitif tidak bocor di halaman publik.

---

Dokumen ini siap digunakan sebagai panduan development.
