#!/usr/bin/env bash
# shiboq-dev イメージのコンテナ内でコマンドを実行するヘルパー。
# GOPATH/GOCACHE/npmキャッシュは /tmp 直下の単一階層パスにバインドマウントする。
# （/go, /root はイメージ内で root 所有のため、非rootユーザー実行時に書き込めない。
#   /tmp/foo のように /tmp のすぐ下の1階層なら、Dockerが中間ディレクトリを
#   rootで自動生成する問題を避けられる。）
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
mkdir -p "$ROOT_DIR/.devcache/gopath" "$ROOT_DIR/.devcache/gobuild" "$ROOT_DIR/.devcache/npm"

exec docker run --rm \
  --user "$(id -u):$(id -g)" \
  -e HOME=/tmp \
  -e GOPATH=/tmp/gopath \
  -e GOCACHE=/tmp/gobuild \
  -e http_proxy="${http_proxy:-}" \
  -e https_proxy="${https_proxy:-}" \
  -e no_proxy="${no_proxy:-}" \
  -v "$ROOT_DIR":/app \
  -v "$ROOT_DIR/.devcache/gopath":/tmp/gopath \
  -v "$ROOT_DIR/.devcache/gobuild":/tmp/gobuild \
  -v "$ROOT_DIR/.devcache/npm":/tmp/.npm \
  -w /app \
  shiboq-dev "$@"
