#!/usr/bin/env bash
set -euo pipefail

REPO="Ray-D-Song/goread"
INSTALL_DIR="${INSTALL_DIR:-$HOME/.local/bin}"

detect_platform() {
	local os arch

	os="$(uname -s | tr '[:upper:]' '[:lower:]')"
	case "$(uname -m)" in
	x86_64 | amd64) arch="amd64" ;;
	aarch64 | arm64) arch="arm64" ;;
	armv7l | armv6l) arch="arm" ;;
	i386 | i686) arch="386" ;;
	*)
		echo "Unsupported architecture: $(uname -m)" >&2
		exit 1
		;;
	esac

	case "$os" in
	linux) echo "linux-$arch" ;;
	darwin) echo "darwin-$arch" ;;
	*)
		echo "Unsupported OS: $os" >&2
		exit 1
		;;
	esac
}

get_latest_version() {
	curl -fsSL "https://api.github.com/repos/$REPO/releases/latest" |
		sed -n 's/.*"tag_name": "\(.*\)".*/\1/p' |
		head -n1
}

platform="$(detect_platform)"
version="$(get_latest_version)"

if [[ -z "$version" ]]; then
	echo "Failed to determine the latest release version." >&2
	exit 1
fi

filename="goread-$platform"
url="https://github.com/$REPO/releases/download/$version/$filename"

mkdir -p "$INSTALL_DIR"
echo "Downloading $url ..."
curl -fsSL "$url" -o "$INSTALL_DIR/goread"
chmod +x "$INSTALL_DIR/goread"

echo "Installed goread $version to $INSTALL_DIR/goread"

if [[ ":$PATH:" != *":$INSTALL_DIR:"* ]]; then
	cat <<EOF

Add $INSTALL_DIR to your PATH, for example:

  export PATH="$INSTALL_DIR:\$PATH"

Or append the line above to your shell profile (~/.bashrc, ~/.zshrc, etc.).
EOF
fi