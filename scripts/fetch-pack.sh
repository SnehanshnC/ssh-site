#!/bin/sh
# Fetches the content pack sections consumed by the SSH site from
# github.com/SnehanshnC/content-pack. All facts about Snehanshn are meant to
# live in that repo, never hardcoded here, so this script is the only place
# that pulls them in - run via `make content`.
#
# Each file is downloaded to a temp file first and only moved into place once
# the download succeeds, so a failed run never leaves a partial file behind.
# The script exits non-zero if any file fails to download.

set -eu

base_url="https://raw.githubusercontent.com/SnehanshnC/content-pack/main"
dest_dir="internal/content/pack"
files="identity.yaml work.yaml projects.yaml links.yaml hobbies.yaml"

mkdir -p "$dest_dir"

for file in $files; do
    tmp_file="$dest_dir/.$file.tmp"
    if ! curl -fsSL "$base_url/$file" -o "$tmp_file"; then
        echo "fetch-pack: failed to download $file" >&2
        rm -f "$tmp_file"
        exit 1
    fi
    mv "$tmp_file" "$dest_dir/$file"
done

echo "fetch-pack: wrote $(echo $files | wc -w | tr -d ' ') files to $dest_dir"
