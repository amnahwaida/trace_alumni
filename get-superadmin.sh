#!/usr/bin/env bash

# Terminal Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
BLUE='\033[0;34m'
CYAN='\033[0;36m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

echo -e "${BLUE}====================================================${NC}"
echo -e "${BLUE}        Alumni Tracker - Super Admin Manager        ${NC}"
echo -e "${BLUE}====================================================${NC}"

DB_FILE="data/alumni.db"

if [ ! -f "$DB_FILE" ]; then
    echo -e "${RED}Error: File database tidak ditemukan di '$DB_FILE'.${NC}"
    echo -e "${YELLOW}Pastikan aplikasi sudah pernah dijalankan minimal satu kali.${NC}"
    exit 1
fi

# Fetch super admin username using Python
USERNAME=$(python3 -c "
import sqlite3
try:
    conn = sqlite3.connect('$DB_FILE')
    cursor = conn.cursor()
    cursor.execute(\"SELECT username FROM users WHERE role = 'super_admin' LIMIT 1\")
    row = cursor.fetchone()
    if row:
        print(row[0])
    else:
        print('')
    conn.close()
except Exception as e:
    print('ERROR:', e)
")

if [[ "$USERNAME" == ERROR:* ]]; then
    echo -e "${RED}Error saat membaca database:${NC}"
    echo "$USERNAME"
    exit 1
fi

if [ -z "$USERNAME" ]; then
    echo -e "${YELLOW}Warning: Tidak ada user dengan role 'super_admin' di database.${NC}"
    echo -e "Silahkan restart docker container untuk men-seed super_admin default."
    exit 0
fi

echo -e "${GREEN}Super Admin terdaftar:${NC}"
echo -e "  Username : ${CYAN}$USERNAME${NC}"
echo -e "  Password : ${YELLOW}[TERENKRIPSI / BCRYPT HASH]${NC}"
echo -e ""
echo -e "Karena password disandikan menggunakan Bcrypt (one-way hash), password asli tidak dapat dibaca kembali."
echo -e ""
read -p "Apakah Anda ingin me-reset password super admin menjadi 'admin123'? (y/n): " confirm

if [[ "$confirm" == "y" || "$confirm" == "Y" ]]; then
    # Pre-calculated bcrypt hash for 'admin123'
    BCRYPT_HASH='$2a$10$VBTWU1Dr3XoFCl.920hNYe9pxoeRCGrwE206gV966NbFV9ULGezjO'
    
    RESULT=$(python3 -c "
import sqlite3
try:
    conn = sqlite3.connect('$DB_FILE')
    cursor = conn.cursor()
    # Update password
    cursor.execute(\"UPDATE users SET password_hash = ? WHERE username = ?\", ('$BCRYPT_HASH', '$USERNAME'))
    # Invalidate active sessions for this user
    cursor.execute(\"DELETE FROM sessions WHERE user_id IN (SELECT id FROM users WHERE username = ?)\", ($USERNAME,))
    conn.commit()
    print('SUCCESS')
    conn.close()
except Exception as e:
    print('ERROR:', e)
")

    if [ "$RESULT" = "SUCCESS" ]; then
        echo -e "\n${GREEN}✅ Password untuk user '${USERNAME}' berhasil direset menjadi: admin123${NC}"
        echo -e "${YELLOW}Silahkan login dan segera ganti password Anda melalui menu 'Ganti Password' di dashboard.${NC}"
    else
        echo -e "\n${RED}❌ Gagal mereset password:${NC}"
        echo "$RESULT"
    fi
else
    echo -e "\nBatal melakukan reset password."
fi
