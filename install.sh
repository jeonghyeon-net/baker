#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(CDPATH= cd -- "$(dirname "$0")" && pwd)"
PREFIX="${PREFIX:-$HOME/.local}"
BIN_DIR="${BIN_DIR:-$PREFIX/bin}"
TARGET="${TARGET:-$BIN_DIR/baker}"
TARGET_DIR="$(dirname "$TARGET")"

need_cmd() {
  if ! command -v "$1" >/dev/null 2>&1; then
    echo "필수 명령을 찾을 수 없습니다: $1" >&2
    exit 1
  fi
}

need_cmd go
need_cmd git

mkdir -p "$TARGET_DIR"

TMP_DIR="$(mktemp -d)"
cleanup() {
  rm -rf "$TMP_DIR"
}
trap cleanup EXIT

cd "$ROOT_DIR"

go build -o "$TMP_DIR/baker" ./cmd/baker
install -m 0755 "$TMP_DIR/baker" "$TARGET"

echo "설치 완료: $TARGET"
echo
if ! command -v gh >/dev/null 2>&1; then
  echo "주의: gh(GitHub CLI)가 아직 없습니다. GitHub 저장소 목록과 PR 표시를 쓰려면 gh를 설치하고 로그인해야 합니다."
  echo
fi

echo "다음 단계"
echo "1. $TARGET_DIR 가 PATH에 없다면 shell 설정 파일에 추가"
echo "2. 필요하면 gh auth login 실행"
echo "3. baker 실행"
echo "4. 첫 실행 후 안내가 나오면 source ~/.zshrc 또는 source ~/.bashrc 실행"
