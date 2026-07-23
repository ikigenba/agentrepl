#!/bin/sh

set -eu

REPO=ikigenba/agentrepl
BINARY=agentrepl
BINDIR=${BINDIR:-${PREFIX:-"$HOME/.local"}/bin}
AGENTREPL_VERSION=${AGENTREPL_VERSION:-latest}

case $(uname -s) in
	Linux) os=linux ;;
	Darwin) os=darwin ;;
	*)
		echo "agentrepl: unsupported operating system: $(uname -s)" >&2
		exit 1
		;;
esac

case $(uname -m) in
	x86_64 | amd64) arch=amd64 ;;
	arm64 | aarch64) arch=arm64 ;;
	*)
		echo "agentrepl: unsupported architecture: $(uname -m)" >&2
		exit 1
		;;
esac

asset="${BINARY}_${os}_${arch}.tar.gz"
if [ "$AGENTREPL_VERSION" = latest ]; then
	url="https://github.com/${REPO}/releases/latest/download/${asset}"
else
	case $AGENTREPL_VERSION in
		v[0-9]*.[0-9]*.[0-9]*) ;;
		*)
			echo "agentrepl: AGENTREPL_VERSION must be latest or a vMAJOR.MINOR.PATCH tag" >&2
			exit 1
			;;
	esac
	url="https://github.com/${REPO}/releases/download/${AGENTREPL_VERSION}/${asset}"
fi

tmpdir=$(mktemp -d "${TMPDIR:-/tmp}/agentrepl.XXXXXX")
trap 'rm -rf "$tmpdir"' EXIT HUP INT TERM

echo "Downloading agentrepl ${AGENTREPL_VERSION} for ${os}/${arch}..." >&2
curl -fsSL "$url" -o "$tmpdir/$asset"
tar -xzf "$tmpdir/$asset" -C "$tmpdir" "$BINARY"
mkdir -p "$BINDIR"
install -m 0755 "$tmpdir/$BINARY" "$BINDIR/$BINARY"

case ":$PATH:" in
	*":$BINDIR:"*) ;;
	*)
		echo "agentrepl: warning: $BINDIR is not on PATH" >&2
		;;
esac

echo "Installed agentrepl to $BINDIR/$BINARY" >&2
