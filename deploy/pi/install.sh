#!/usr/bin/env bash
# Install or update sonora-mcp as a systemd service on a Raspberry Pi
# (64-bit Raspberry Pi OS). It downloads the linux/arm64 archive that
# GoReleaser published for the release RELEASE_TAG (or --version), verifies it
# against the release's checksums.txt, installs the binary, writes the service
# configuration and unit, and (re)starts the service. Safe to re-run.
#
# One command, straight from GitHub:
#   curl -fsSL https://raw.githubusercontent.com/Sonora-Multiroom/sonora-mcp/main/deploy/pi/install.sh | sudo bash -s -- --hub-url <url>
# Or from a copy on the Pi:
#   sudo ./install.sh --hub-url <url> [--port <port>] [--host <address>] [--version <tag>]
#
# Everything runs from main, called on the last line, so a download cut off
# part-way through executes nothing.
set -euo pipefail

# Release whose binary is installed. Set to the release tag in the commit
# that gets tagged on main; override with --version.
RELEASE_TAG="dev"

REPO="Sonora-Multiroom/sonora-mcp"
SCRIPT_URL="https://raw.githubusercontent.com/${REPO}/main/deploy/pi/install.sh"
BIN="/usr/local/bin/sonora-mcp"
ENV_FILE="/etc/default/sonora-mcp"
UNIT_FILE="/etc/systemd/system/sonora-mcp.service"

usage() {
	cat <<EOF
Usage:
  curl -fsSL ${SCRIPT_URL} | sudo bash -s -- --hub-url <url> [options]
  sudo ./install.sh --hub-url <url> [options]

Options:
  --hub-url <url>     Base URL of the Multiroom Audio Hub (required)
  --port <port>       Port sonora-mcp listens on (default 3001)
  --host <address>    Address to listen on, e.g. 127.0.0.1 (default: all addresses)
  --version <tag>     Release to install (default ${RELEASE_TAG})
  -h, --help          Show this help
EOF
}

die() {
	echo "install.sh: $*" >&2
	exit 1
}

# work and tmp are global so the EXIT trap can clean them up.
work=""
tmp=""
cleanup() {
	[[ -z "$work" ]] || rm -rf "$work"
	[[ -z "$tmp" ]] || rm -f "$tmp"
}

# install_binary downloads the release archive for tag and checksums.txt,
# verifies the archive, then copies the binary to a temp file next to the
# installed one and renames it into place: a rename works while the old
# binary is running, and a failed download or check leaves it intact.
install_binary() {
	local tag="$1"
	local archive="sonora-mcp_${tag#v}_linux_arm64.tar.gz"
	local base="https://github.com/${REPO}/releases/download/${tag}"
	local f

	work="$(mktemp -d)"
	echo "Downloading ${base}/${archive}"
	for f in "$archive" checksums.txt; do
		curl -fsSL -o "$work/$f" "$base/$f" ||
			die "download failed: $base/$f (check the network and that release ${tag} exists)"
	done
	awk -v f="$archive" '$2 == f' "$work/checksums.txt" >"$work/archive.sha256"
	[[ -s "$work/archive.sha256" ]] || die "checksums.txt of release ${tag} does not list ${archive}"
	(cd "$work" && sha256sum --check --quiet archive.sha256) || die "checksum mismatch for ${archive}"
	tar -xzf "$work/$archive" -C "$work" sonora-mcp || die "${archive} does not contain sonora-mcp"

	tmp="$(mktemp /usr/local/bin/.sonora-mcp.XXXXXX)"
	cp "$work/sonora-mcp" "$tmp"
	chmod 0755 "$tmp"
	mv -f "$tmp" "$BIN"
	tmp=""
}

# write_service writes the env file and the systemd unit.
write_service() {
	local hub_url="$1" port="$2" host="$3"
	local host_arg=""
	[[ -z "$host" ]] || host_arg="--host ${host}"

	cat >"$ENV_FILE" <<EOF
# sonora-mcp settings, written by install.sh. Re-run install.sh to change them.
SONORA_HUB_URL=${hub_url}
SONORA_PORT=${port}
SONORA_HOST_ARG=${host_arg}
EOF

	cat >"$UNIT_FILE" <<'EOF'
[Unit]
Description=Sonora Multiroom MCP server
After=network-online.target
Wants=network-online.target

[Service]
DynamicUser=yes
EnvironmentFile=/etc/default/sonora-mcp
ExecStart=/usr/local/bin/sonora-mcp --multiroom-url ${SONORA_HUB_URL} --port ${SONORA_PORT} $SONORA_HOST_ARG
Restart=on-failure
RestartSec=2
NoNewPrivileges=yes
ProtectSystem=strict
ProtectHome=yes

[Install]
WantedBy=multi-user.target
EOF
}

main() {
	local hub_url="" port="3001" host="" tag="$RELEASE_TAG"

	while [[ $# -gt 0 ]]; do
		case "$1" in
		--hub-url) hub_url="${2:?--hub-url needs a value}"; shift 2 ;;
		--port) port="${2:?--port needs a value}"; shift 2 ;;
		--host) host="${2:?--host needs a value}"; shift 2 ;;
		--version) tag="${2:?--version needs a value}"; shift 2 ;;
		-h | --help) usage; exit 0 ;;
		*) usage >&2; die "unknown argument: $1" ;;
		esac
	done

	[[ -n "$hub_url" ]] || { usage >&2; die "--hub-url is required"; }
	[[ "$(id -u)" -eq 0 ]] || die "must be run as root (use sudo)"
	command -v systemctl >/dev/null || die "systemd (systemctl) is required"
	command -v curl >/dev/null || die "curl is required (sudo apt install curl)"
	[[ "$(uname -m)" == "aarch64" ]] || die "64-bit ARM OS required (uname -m is $(uname -m), want aarch64)"

	trap cleanup EXIT
	install_binary "$tag"
	write_service "$hub_url" "$port" "$host"

	systemctl daemon-reload
	systemctl enable sonora-mcp >/dev/null
	systemctl restart sonora-mcp
	systemctl --no-pager status sonora-mcp
}

main "$@"
