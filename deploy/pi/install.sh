#!/usr/bin/env bash
# Install or update sonora-mcp as a systemd service on a Raspberry Pi
# (64-bit Raspberry Pi OS). This is the only file that needs to be copied to
# the Pi: it downloads the linux/arm64 binary from the GitHub Release named
# by RELEASE_TAG (or --version), writes the service configuration and unit,
# and (re)starts the service. Safe to re-run.
#
#   sudo ./install.sh --hub-url <url> [--port <port>] [--host <address>] [--version <tag>]
set -euo pipefail

# Release whose binary is installed. Set to the release tag in the commit
# that gets tagged on main; override with --version.
RELEASE_TAG="dev"

REPO="tiger-seo/sonora-mcp"
ASSET="sonora-mcp-linux-arm64"
BIN="/usr/local/bin/sonora-mcp"
ENV_FILE="/etc/default/sonora-mcp"
UNIT_FILE="/etc/systemd/system/sonora-mcp.service"

usage() {
	cat <<EOF
Usage:
  sudo $0 --hub-url <url> [--port <port>] [--host <address>] [--version <tag>]

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

hub_url=""
port="3001"
host=""
tag="$RELEASE_TAG"

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

# Download to a temp file next to the binary, then rename over it: a rename
# works while the old binary is running, and a failed download leaves it intact.
url="https://github.com/${REPO}/releases/download/${tag}/${ASSET}"
tmp="$(mktemp /usr/local/bin/.sonora-mcp.XXXXXX)"
trap 'rm -f "$tmp"' EXIT
echo "Downloading ${url}"
curl -fsSL -o "$tmp" "$url" || die "download failed: ${url} (check the network and that release ${tag} exists)"
chmod 0755 "$tmp"
mv -f "$tmp" "$BIN"
trap - EXIT

host_arg=""
[[ -n "$host" ]] && host_arg="--host ${host}"
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

systemctl daemon-reload
systemctl enable sonora-mcp >/dev/null
systemctl restart sonora-mcp
systemctl --no-pager status sonora-mcp
