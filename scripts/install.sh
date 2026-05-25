#!/bin/sh
set -eu

repo="${P99_REPO:-Justin-Arnold/p99}"
version="${P99_VERSION:-latest}"
install_dir="${INSTALL_DIR:-$HOME/.local/bin}"

need() {
	if ! command -v "$1" >/dev/null 2>&1; then
		echo "p99 install: missing required command: $1" >&2
		exit 1
	fi
}

download() {
	url="$1"
	dest="$2"

	if command -v curl >/dev/null 2>&1; then
		curl -fsSL "$url" -o "$dest"
		return
	fi
	if command -v wget >/dev/null 2>&1; then
		wget -qO "$dest" "$url"
		return
	fi

	echo "p99 install: curl or wget is required" >&2
	exit 1
}

platform() {
	case "$(uname -s)" in
		Darwin) echo "darwin" ;;
		Linux) echo "linux" ;;
		*)
			echo "p99 install: unsupported operating system: $(uname -s)" >&2
			exit 1
			;;
	esac
}

architecture() {
	case "$(uname -m)" in
		x86_64 | amd64) echo "amd64" ;;
		arm64 | aarch64) echo "arm64" ;;
		*)
			echo "p99 install: unsupported architecture: $(uname -m)" >&2
			exit 1
			;;
	esac
}

verify_checksum() {
	checksums="$1"
	asset="$2"
	archive="$3"

	line="$(grep "  $asset\$" "$checksums" || true)"
	if [ -z "$line" ]; then
		line="$(grep " $asset\$" "$checksums" || true)"
	fi
	if [ -z "$line" ]; then
		echo "p99 install: checksum for $asset was not found" >&2
		exit 1
	fi

	expected="$(printf '%s\n' "$line" | awk '{print $1}')"
	actual=""

	if command -v sha256sum >/dev/null 2>&1; then
		actual="$(sha256sum "$archive" | awk '{print $1}')"
	elif command -v shasum >/dev/null 2>&1; then
		actual="$(shasum -a 256 "$archive" | awk '{print $1}')"
	else
		echo "p99 install: sha256sum or shasum is required to verify the download" >&2
		exit 1
	fi

	if [ "$expected" != "$actual" ]; then
		echo "p99 install: checksum verification failed for $asset" >&2
		exit 1
	fi
}

os="$(platform)"
arch="$(architecture)"
asset="p99_${os}_${arch}.tar.gz"

if [ "$version" = "latest" ]; then
	base_url="https://github.com/${repo}/releases/latest/download"
else
	base_url="https://github.com/${repo}/releases/download/${version}"
fi

need tar
need grep
need awk
need mktemp
need mkdir
need cp
need chmod

tmp="$(mktemp -d)"
trap 'rm -rf "$tmp"' EXIT INT TERM

archive="$tmp/$asset"
checksums="$tmp/checksums.txt"

echo "Downloading p99 ${version} for ${os}/${arch}"
download "$base_url/$asset" "$archive"
download "$base_url/checksums.txt" "$checksums"
verify_checksum "$checksums" "$asset" "$archive"

tar -xzf "$archive" -C "$tmp"

if [ ! -f "$tmp/p99" ]; then
	echo "p99 install: release archive did not contain a p99 binary" >&2
	exit 1
fi

# Install with cp rather than mv so a failed chmod cannot consume the only
# extracted copy inside the temporary directory.
mkdir -p "$install_dir"
cp "$tmp/p99" "$install_dir/p99"
chmod 0755 "$install_dir/p99"

echo "Installed p99 to $install_dir/p99"

case ":$PATH:" in
	*":$install_dir:"*) ;;
	*)
		echo "Add $install_dir to PATH to run p99 without a full path."
		;;
esac
