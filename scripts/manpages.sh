#!/bin/sh
set -e
rm -rf manpages
mkdir manpages
go run cmd/git-vibe/main.go man | gzip -c -9 >manpages/git-vibe.1.gz