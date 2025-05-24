#!/bin/sh
set -e
rm -rf completions
mkdir completions
for sh in bash zsh fish; do
	go run cmd/git-vibe/main.go completion "$sh" >"completions/git-vibe.$sh"
done