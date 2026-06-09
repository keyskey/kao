#!/usr/bin/env sh

set -eu

usage() {
  echo "usage: $0 <patch|minor|major>"
  echo "example: $0 patch"
}

if [ "${1-}" = "" ]; then
  usage
  exit 2
fi

bump_type="$1"
case "$bump_type" in
  patch|minor|major) ;;
  *)
    echo "unsupported bump type: $bump_type" >&2
    usage
    exit 2
    ;;
esac

if ! git rev-parse --is-inside-work-tree >/dev/null 2>&1; then
  echo "this command must run inside a git repository" >&2
  exit 1
fi

latest_tag="$(git tag --list 'v[0-9]*.[0-9]*.[0-9]*' --sort=-version:refname | sed -n '1p')"

if [ -z "$latest_tag" ]; then
  major=0
  minor=0
  patch=0
else
  version="${latest_tag#v}"
  major="$(echo "$version" | cut -d. -f1)"
  minor="$(echo "$version" | cut -d. -f2)"
  patch="$(echo "$version" | cut -d. -f3)"
fi

case "$bump_type" in
  patch)
    patch=$((patch + 1))
    ;;
  minor)
    minor=$((minor + 1))
    patch=0
    ;;
  major)
    major=$((major + 1))
    minor=0
    patch=0
    ;;
esac

next_tag="v${major}.${minor}.${patch}"

if git rev-parse -q --verify "refs/tags/${next_tag}" >/dev/null 2>&1; then
  echo "tag already exists: ${next_tag}" >&2
  exit 1
fi

git tag "${next_tag}"
echo "created tag: ${next_tag}"
echo "push with: git push origin ${next_tag}"
