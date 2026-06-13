# 🎓 Website Rekam Jejak Alumni SMAS Muhammadiyah 1 Ngawi

Sistem digitalisasi data alumni SMAS Muhammadiyah 1 Ngawi yang dirancang khusus untuk berjalan dengan performa maksimal pada perangkat dengan spesifikasi terbatas (seperti STB Armbian Linux 1 GB RAM, sisa penyimpanan ~1 GB). Menggunakan arsitektur monolitik yang sangat ringan, aman, dan efisien dengan total ukuran *footprint* penyimpanan kurang dari 100 MB.

---

## 🚀 Fitur Utama

- **Landing Page & Pencarian Publik (Real-Time)**: Pencarian alumni interaktif menggunakan HTMX (tanpa reload halaman) berdasarkan nama, tahun kelulusan (angkatan), domisili, pekerjaan, dan nomor HP.
- **Papan Pengumuman**: Board informasi loker, beasiswa, reuni, dan umum untuk publik. Dapat dikelola oleh Admin/Super Admin (dengan validasi tautan eksternal HTTPS yang aman untuk mencegah XSS).
- **Manajemen Data Alumni**: Fitur CRUD data lengkap alumni beserta upload foto profil dengan **Auto-Resize & Kompresi** otomatis (foto di-resize ke maksimal 200x200 px & kompresi JPEG Quality 65%) untuk menghemat ruang disk.
- **Manajemen User (Multi-Role)**: Pengaturan hak akses berjenjang untuk Super Admin, Admin, dan Staff guna menjaga keamanan data.
- **Import & Export Excel (In-Memory)**: Pengunggahan dan pengunduhan data alumni secara massal menggunakan file Excel (.xlsx) yang diproses langsung pada memori (RAM) tanpa meninggalkan file sampah sementara di disk.
- **Premium Glassmorphism & UI Responsive (Mobile First)**: Antarmuka modern yang cepat dengan Tailwind CSS, dilengkapi dengan modal konfirmasi penghapusan data kustom (glassmorphism overlay).

---

## 👥 Hak Akses Pengguna (User Roles)

Sistem ini menerapkan pembatasan hak akses yang ketat berdasarkan peran masing-masing pengguna:

| Fitur / Kemampuan | Super Admin | Admin | Staff | Publik (Alumni) |
| :--- | :---: | :---: | :---: | :---: |
| **Pencarian Alumni** | ✅ Full | ✅ Full | ✅ Full | ⚠️ Masked (Sensitif disamarkan) |
| **Tambah/Edit/Hapus Alumni** | ✅ Ya | ✅ Ya | ❌ Tidak | ❌ Tidak |
| **Upload Foto Profil Alumni** | ✅ Ya | ✅ Ya | ❌ Tidak | ❌ Tidak |
| **Import Excel Data Alumni** | ✅ Ya | ✅ Ya | ❌ Tidak | ❌ Tidak |
| **Export Excel Data Alumni** | ✅ Ya | ✅ Ya | ✅ Ya | ❌ Tidak |
| **Tambah/Edit/Hapus Pengumuman**| ✅ Ya | ✅ Ya | ❌ Tidak | ❌ Tidak |
| **Toggle Global Board Pengumuman**| ✅ Ya | ✅ Ya | ❌ Tidak | ❌ Tidak |
| **Manajemen User (Tambah/Hapus)** | ✅ Ya | ❌ Tidak | ❌ Tidak | ❌ Tidak |
| **Akses Dashboard** | ✅ Ya | ✅ Ya | ✅ Ya | ❌ Tidak |

### Deskripsi Peran:
- **Super Admin**: Memiliki kontrol penuh atas sistem termasuk membuat dan menghapus akun pengguna (Admin/Staff) lainnya.
- **Admin**: Bertanggung jawab atas pengelolaan data alumni (CRUD, Import/Export) serta pemeliharaan Papan Pengumuman publik.
- **Staff**: Hanya memiliki hak akses baca (Read-only) data alumni untuk pencarian internal dan ekspor data rekapitulasi (tidak dapat menambah, mengubah, atau menghapus data).
- **Publik (Alumni)**: Hanya dapat melihat landing page, mencari data alumni secara terbatas (alamat email dan nomor telepon disamarkan demi privasi), serta membaca Papan Pengumuman.

---

## 🛠️ Teknologi & Spesifikasi Teknis

- **Backend**: Golang 1.22+ (menggunakan standard library & router `go-chi/chi` yang sangat ringan)
- **Database**: SQLite 3 (Pure Go, driver `modernc.org/sqlite` tanpa ketergantungan CGO)
- **Frontend**: Go `html/template` + `HTMX` (untuk dynamic AJAX requests) + `Tailwind CSS`
- **Excel Processor**: `github.com/xuri/excelize/v2`
- **Image Processor**: `github.com/disintegration/imaging`
- **Deployment**: Docker (Multi-stage build) dengan total ukuran final image < 35 MB.

---

## ⚠️ Batasan Kritis (Sistem Target STB 1 GB RAM)

1. **In-Memory Processing**: Semua operasi pengolahan gambar dan konversi file Excel dilakukan langsung di RAM. Tidak ada penyimpanan file temporer di dalam disk local untuk mencegah kehabisan ruang penyimpanan.
2. **Auto-Clean Uploads**: Jika penyimpanan record database gagal, file gambar profil yang telah di-upload akan otomatis dihapus secara otomatis dari disk (`data/uploads`).
3. **Log Rotation**: Ukuran log Docker dibatasi maksimal 10MB per file dengan maksimal rotasi 3 file untuk menghemat sisa penyimpanan OS Armbian.

---

## 📦 Menjalankan Proyek dengan Docker

### Prasyarat
- Docker dan Docker Compose telah terinstal pada sistem target.

### Langkah-Langkah Menjalankan
1. Klon repositori ini ke STB / Server lokal:
   ```bash
   git clone https://github.com/amnahwaida/trace_alumni.git
   cd trace_alumni
   ```
2. Jalankan aplikasi menggunakan Docker Compose:
   ```bash
   docker compose up -d --build
   ```
3. Aplikasi akan otomatis mengompilasi binary dan menjalankan server di port **`8080`**. Buka di browser Anda:
   ```
   http://localhost:8080
   ```

### Menghentikan Aplikasi
```bash
docker compose down
```

---

## 🖥️ Cara Mengatasi di STB (Menggunakan Laptop/PC)

Jika Anda men-deploy ke perangkat dengan spesifikasi terbatas seperti **STB (1 GB RAM, sisa penyimpanan < 1 GB)**, proses kompilasi lokal dengan `docker compose up -d --build` kemungkinan besar akan gagal karena kehabisan ruang penyimpanan (`no space left on device`) saat mengunduh dependency Go.

Untuk mengatasinya, Anda dapat mem-build Docker image untuk arsitektur ARM64 di laptop/PC Anda, lalu mengirimkannya ke STB.

### Langkah 1: Build Image di Laptop / PC Anda
1. Pastikan Docker dan Git sudah terinstal di laptop Anda.
2. Clone repositori ini di laptop Anda:
   ```bash
   git clone https://github.com/amnahwaida/trace_alumni.git
   cd trace_alumni
   ```
3. Jalankan script build khusus STB yang disediakan:
   ```bash
   ./deploy-stb.sh
   ```
   Script ini akan otomatis mengompilasi program untuk arsitektur ARM64 (STB) menggunakan Docker Buildx dan menghasilkan file portable bernama **`alumni-tracker-arm64.tar.gz`** (ukuran kompresi sekitar ~15 MB).

### Langkah 2: Transfer File Hasil Build ke STB
Kirim file `.tar.gz` tersebut ke STB Anda menggunakan perintah `scp`:
```bash
scp alumni-tracker-arm64.tar.gz username_stb@ip_stb:~/
```

### Langkah 3: Jalankan di STB
1. Masuk ke terminal SSH STB Anda.
2. Bersihkan cache build docker lama untuk melegakan penyimpanan STB:
   ```bash
   sudo docker system prune -a --volumes -f
   ```
3. Load image pra-build yang sudah ditransfer ke dalam Docker STB:
   ```bash
   sudo docker load < ~/alumni-tracker-arm64.tar.gz
   ```
4. Masuk ke folder project di STB, lalu jalankan container (pastikan `docker-compose.yml` Anda sudah ter-update):
   ```bash
   cd ~/trace_alumni
   git pull origin main
   sudo docker compose up -d
   ```
   *Catatan: Jangan tambahkan parameter `--build` di STB agar Docker tidak memicu proses kompilasi lokal yang memakan memori.*

---

## 💾 Folder Struktur & Mount Volume

Volume pada Docker Compose diarahkan pada folder `data/` di root proyek untuk mempermudah backup. Folder ini berisi:
- `data/alumni.db`: Database SQLite utama.
- `data/uploads/`: Folder penyimpanan foto profil alumni yang telah di-resize & dikompres.

---

## 🛡️ Skema Database (SQLite)

Berikut adalah struktur tabel inti yang digunakan:

```sql
-- Tabel Pengguna Sistem
CREATE TABLE users (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    username TEXT UNIQUE NOT NULL,
    password_hash TEXT NOT NULL,
    role TEXT CHECK(role IN ('super_admin', 'admin', 'staff')) NOT NULL,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

-- Tabel Data Alumni
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
    foto_profil TEXT,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

-- Tabel Papan Pengumuman
CREATE TABLE info_papan (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    judul TEXT NOT NULL,
    deskripsi TEXT NOT NULL,
    link_eksternal TEXT,
    kategori TEXT DEFAULT 'umum' CHECK(kategori IN ('loker', 'beasiswa', 'reuni', 'umum')),
    dibuat_oleh INTEGER REFERENCES users(id),
    dibuat_pada DATETIME DEFAULT CURRENT_TIMESTAMP
);
```

---

## 📂 Backup & Pemeliharaan Log

Proyek ini telah dilengkapi dengan script backup otomatis (`scripts/backup.sh`) yang akan mengompres folder `data/` menjadi file `.tar.gz` ke dalam folder `backups/`.

Untuk menjalankan backup secara berkala, daftarkan script tersebut pada `cron` di host Armbian Anda:
```bash
# Buka editor cron
crontab -e

# Jalankan backup setiap hari Minggu jam 02.00 pagi
0 2 * * 0 /bin/bash /path/to/trace_alumni/scripts/backup.sh
```

---

## 📜 Lisensi
Aplikasi ini dikembangkan untuk digunakan secara internal oleh **SMAS Muhammadiyah 1 Ngawi**. Hak Cipta dilindungi undang-undang.
