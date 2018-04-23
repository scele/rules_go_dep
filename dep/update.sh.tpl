#!/bin/sh
set -e

GOPKG_LOCK="%{gopkg_lock}"
GOPKG_BZL="%{gopkg_bzl}"
if [ -n "$GOPKG_BZL" ]; then
    GOPKG_BZL=$(dirname $(realpath "$GOPKG_LOCK"))/Gopkg.bzl
fi
GOPKG_BZL_TEMP="Gopkg.bzl.tmp"
echo "Generating $GOPKG_BZL from $GOPKG_LOCK..."

%{dep2bazel} \
    -build-file-generation "%{build_file_generation}" \
    -build-file-proto-mode "%{build_file_proto_mode}" \
    -go-prefix "%{go_prefix}" \
    -source-directory "%{workspace_root_path}" \
    "%{mirrors}" \
    -o $GOPKG_BZL_TEMP \
    $GOPKG_LOCK

cp "$GOPKG_BZL_TEMP" "$GOPKG_BZL"
echo "$GOPKG_BZL updated!"
