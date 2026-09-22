#!/usr/bin/env bash
set -e
# Usage: ./scripts/deploy-vps.sh [user@host]  (default reads VPS var from .env)
if [ -f .env ]; then set -a; source .env; set +a; fi
VPS_HOST="${1:-$VPS_HOST}"
VPS_USER="${VPS_USER:-root}"
VPS_PATH="${VPS_PATH:-/opt/trampickle}"
if [ -z "$VPS_HOST" ]; then echo "ERR: set VPS_HOST in .env or pass user@host"; exit 1; fi
echo "Deploy to $VPS_USER@$VPS_HOST:$VPS_PATH"
ssh "$VPS_USER@$VPS_HOST" "mkdir -p $VPS_PATH"
rsync -avz --delete --exclude .git --exclude node_modules --exclude dist ./ "$VPS_USER@$VPS_HOST:$VPS_PATH/"
ssh "$VPS_USER@$VPS_HOST" "cd $VPS_PATH && docker compose up -d --build && docker ps | grep trampickle"
echo "Done. Check https://trampickle.vn and https://trampickle.vn/sitemap.xml"