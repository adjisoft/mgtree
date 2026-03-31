#!/usr/bin/env bash
set -euo pipefail

if [[ "${OSTYPE:-}" != linux* ]] && [[ "$(uname -s 2>/dev/null || true)" != "Linux" ]]; then
  echo "This setup script is intended for Linux environments only." >&2
  exit 1
fi

PROJECT_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
INSTALL_DIR="${INSTALL_DIR:-$HOME/.local/bin}"
BINARY_NAME="${BINARY_NAME:-mgtree}"
TARGET_PATH="${INSTALL_DIR}/${BINARY_NAME}"
SKIP_PATH_UPDATE="${SKIP_PATH_UPDATE:-0}"

log() {
  printf '[setup] %s\n' "$*"
}

fail() {
  printf '[setup] %s\n' "$*" >&2
  exit 1
}

ensure_command() {
  local command_name="$1"
  command -v "${command_name}" >/dev/null 2>&1 || fail "Missing required command: ${command_name}"
}

choose_shell_rc() {
  local shell_name rc_file
  shell_name="$(basename "${SHELL:-}")"

  case "${shell_name}" in
    bash)
      rc_file="$HOME/.bashrc"
      ;;
    zsh)
      rc_file="$HOME/.zshrc"
      ;;
    fish)
      rc_file="$HOME/.config/fish/config.fish"
      ;;
    *)
      rc_file="$HOME/.profile"
      ;;
  esac

  printf '%s\n' "${rc_file}"
}

append_path_export() {
  local rc_file="$1"
  local export_line='export PATH="$HOME/.local/bin:$PATH"'

  mkdir -p "$(dirname "${rc_file}")"
  touch "${rc_file}"

  if grep -Fq "${export_line}" "${rc_file}"; then
    log "PATH export already present in ${rc_file}"
    return
  fi

  {
    printf '\n# Added by mgtree setup\n'
    printf '%s\n' "${export_line}"
  } >> "${rc_file}"

  log "Added ~/.local/bin to PATH in ${rc_file}"
}

log "Checking build prerequisites"
ensure_command go
ensure_command chmod

log "Preparing install directory at ${INSTALL_DIR}"
mkdir -p "${INSTALL_DIR}"

log "Building ${BINARY_NAME} from ${PROJECT_ROOT}"
(
  cd "${PROJECT_ROOT}"
  GO111MODULE=on go build -o "${TARGET_PATH}" .
)

chmod +x "${TARGET_PATH}"
log "Installed binary to ${TARGET_PATH}"

if [[ "${SKIP_PATH_UPDATE}" != "1" && "${INSTALL_DIR}" == "$HOME/.local/bin" ]]; then
  rc_file="$(choose_shell_rc)"
  append_path_export "${rc_file}"

  if [[ ":${PATH}:" != *":${INSTALL_DIR}:"* ]]; then
    log "Current shell PATH does not include ${INSTALL_DIR}"
    log "Run: source ${rc_file}"
  fi
else
  log "Skipping PATH update"
fi

log "Verifying installation"
if [[ -x "${TARGET_PATH}" ]]; then
  "${TARGET_PATH}" --help >/dev/null 2>&1 || fail "Binary was built but did not execute correctly"
fi

cat <<EOF

mgtree setup complete.

Binary path:
  ${TARGET_PATH}

Useful commands:
  ${BINARY_NAME} --help
  ${BINARY_NAME} -lah

Optional environment variables:
  INSTALL_DIR=/custom/bin
  BINARY_NAME=mgtree
  SKIP_PATH_UPDATE=1
EOF
