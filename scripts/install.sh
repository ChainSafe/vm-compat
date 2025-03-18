#!/usr/bin/env bash
set -e

version="$1"
install_dir="$2"

if [[ -z "$version" || -z "$install_dir" ]]; then
    echo "Usage: install.sh <version> <install_dir>"
    exit 1
fi

os=$(uname | tr '[:upper:]' '[:lower:]')
arch=$(uname -m)

case "$arch" in
    x86_64) arch="amd64" ;;
    aarch64) arch="arm64" ;;
    *) echo "Unsupported architecture: $arch"; exit 1 ;;
esac

binary_url="https://github.com/ChainSafe/vm-compat/releases/download/$version/analyzer-$os-$arch"
binary_path="$install_dir/analyzer"

echo "Downloading $binary_url..."
curl -sSL "$binary_url" -o "$binary_path"
chmod +x "$binary_path"