#!/usr/bin/env bash

VERSION=$(cat VERSION)
cat << EOF > gen/version.go
// Generated by generate_version.sh. Do not edit.
package gen

const Version = "${VERSION}"
EOF
