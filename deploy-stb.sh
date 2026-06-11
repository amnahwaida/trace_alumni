#!/usr/bin/env bash
set -e

# ===================================================
#  Alumni Tracker - Deploy ke STB (ARM64)
# ===================================================
# Script ini mem-build Docker image untuk arsitektur
# ARM64 (STB Amlogic S9xx) di laptop/PC Anda, lalu
# menyimpannya sebagai file .tar.gz yang bisa
# ditransfer ke STB.
#
# Cara pakai:
#   1. Jalankan script ini di laptop:
#      ./deploy-stb.sh
#
#   2. Transfer file hasil build ke STB:
#      scp alumni-tracker-arm64.tar.gz user@stb-ip:~/
#
#   3. Di STB, load image dan jalankan:
#      sudo docker load < ~/alumni-tracker-arm64.tar.gz
#      cd ~/trace_alumni
#      sudo docker compose up -d
# ===================================================

RED='\033[0;31m'
GREEN='\033[0;32m'
BLUE='\033[0;34m'
YELLOW='\033[1;33m'
NC='\033[0m'

IMAGE_NAME="alumni-tracker"
IMAGE_TAG="latest"
OUTPUT_FILE="alumni-tracker-arm64.tar.gz"

echo -e "${BLUE}====================================================${NC}"
echo -e "${BLUE}   Alumni Tracker - Build untuk STB (ARM64)          ${NC}"
echo -e "${BLUE}====================================================${NC}"

# Step 1: Setup buildx untuk multi-arch
echo -e "\n${YELLOW}[1/4] Menyiapkan Docker Buildx...${NC}"
docker buildx inspect multiarch-builder >/dev/null 2>&1 || \
    docker buildx create --name multiarch-builder --use
docker buildx use multiarch-builder

# Setup QEMU untuk emulasi ARM64 (jika belum)
echo -e "${YELLOW}[2/4] Menyiapkan QEMU untuk cross-compile ARM64...${NC}"
docker run --privileged --rm tonistiigi/binfmt --install arm64 2>/dev/null || true

# Step 2: Build image untuk ARM64
echo -e "${YELLOW}[3/4] Memulai build image untuk linux/arm64...${NC}"
echo -e "       (Proses ini membutuhkan waktu beberapa menit)\n"

docker buildx build \
    --platform linux/arm64 \
    -t ${IMAGE_NAME}:${IMAGE_TAG} \
    --load \
    .

# Step 3: Save image ke file tar.gz
echo -e "\n${YELLOW}[4/4] Menyimpan image ke ${OUTPUT_FILE}...${NC}"
docker save ${IMAGE_NAME}:${IMAGE_TAG} | gzip > ${OUTPUT_FILE}

FILE_SIZE=$(du -h ${OUTPUT_FILE} | cut -f1)
echo -e "\n${GREEN}====================================================${NC}"
echo -e "${GREEN}✅ Build selesai!${NC}"
echo -e "${GREEN}====================================================${NC}"
echo -e "File  : ${BLUE}${OUTPUT_FILE}${NC}"
echo -e "Ukuran: ${BLUE}${FILE_SIZE}${NC}"
echo -e ""
echo -e "${YELLOW}Langkah selanjutnya:${NC}"
echo -e "  1. Transfer ke STB:"
echo -e "     ${BLUE}scp ${OUTPUT_FILE} vannyezha@<IP_STB>:~/${NC}"
echo -e ""
echo -e "  2. Di STB, jalankan:"
echo -e "     ${BLUE}sudo docker system prune -a --volumes -f${NC}"
echo -e "     ${BLUE}sudo docker load < ~/${OUTPUT_FILE}${NC}"
echo -e "     ${BLUE}cd ~/trace_alumni && sudo docker compose up -d${NC}"
