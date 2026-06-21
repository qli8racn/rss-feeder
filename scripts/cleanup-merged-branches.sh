#!/bin/bash
# origin/main にマージ済みのローカルブランチを一覧表示し、--delete 指定時のみ削除する。
# デフォルトはdry-run（一覧表示のみ）。mainブランチ・現在のブランチは対象外。
# git branch -d（安全削除）のみを使うため、未マージの変更が残るブランチは削除されない。
set -euo pipefail

DELETE=0
for arg in "$@"; do
    case "$arg" in
        --delete) DELETE=1 ;;
        *)
            echo "unknown option: $arg" >&2
            echo "usage: $0 [--delete]" >&2
            exit 1
            ;;
    esac
done

git fetch origin main

current_branch=$(git branch --show-current)

merged_branches=$(git branch --merged origin/main --format='%(refname:short)' | grep -v -E '^(main|master)$' || true)
if [ -n "$current_branch" ]; then
    merged_branches=$(echo "$merged_branches" | grep -v -F -x "$current_branch" || true)
fi

if [ -z "$merged_branches" ]; then
    echo "マージ済みのローカルブランチはありません"
    exit 0
fi

if [ "$DELETE" -eq 0 ]; then
    echo "origin/main にマージ済みのローカルブランチ（--delete で削除）:"
    echo "$merged_branches"
    exit 0
fi

echo "$merged_branches" | while IFS= read -r branch; do
    git branch -d "$branch"
done
