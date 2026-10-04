#!/bin/sh
# Web版のjqエンジン（Goをwasmにしたもの）をビルドし、Goに付属するローダー
# wasm_exec.js と一緒に frontend/src/backend/generated/ へ出力する。
# 出力は生成物なので.gitignore済み。`npm run build:web`から呼ばれる。
set -eu
ROOT_DIR="$(cd "$(dirname "$0")/.." && pwd)"
OUT_DIR="$ROOT_DIR/frontend/src/backend/generated"
mkdir -p "$OUT_DIR"
cd "$ROOT_DIR"
GOOS=js GOARCH=wasm go build -trimpath -ldflags="-s -w" -o "$OUT_DIR/shiboq.wasm" .
# wasm_exec.jsはビルドに使ったGoと同じバージョンのものでなければならない
cp "$(go env GOROOT)/lib/wasm/wasm_exec.js" "$OUT_DIR/wasm_exec.js"
