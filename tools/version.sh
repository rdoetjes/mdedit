#!/bin/sh
# Prints the version to build: $VERSION if set (the release workflow sets it),
# else from the git tags: 0.7.0 exactly at tag v0.7.0, 0.7.0-3-gabc1234 three
# commits after it, with -dirty for uncommitted changes, or "dev" without git
# or tags. Run `git fetch --tags` to know the newest releases.
# With -numeric, prints only the release part (0.7.0), for the macOS app.
#
# safe.directory: under sudo (make install) git runs as root in a checkout
# owned by someone else, and refuses otherwise ("dubious ownership").
if [ -z "$VERSION" ]; then
	git="git -c safe.directory=*"
	tag=$($git tag --merged HEAD --list 'v[0-9]*' --sort=-v:refname 2>/dev/null | head -1)
	if [ -z "$tag" ]; then
		VERSION=dev
	else
		VERSION=${tag#v}
		n=$($git rev-list --count "$tag..HEAD")
		[ "$n" -gt 0 ] && VERSION="$VERSION-$n-g$($git rev-parse --short HEAD)"
		$git diff --quiet HEAD 2>/dev/null || VERSION="$VERSION-dirty"
	fi
fi
if [ "$1" = -numeric ]; then
	VERSION=$(echo "$VERSION" | sed 's/^\([0-9][0-9.]*\).*/\1/')
	case $VERSION in [0-9]*) ;; *) VERSION=0.0.0 ;; esac
fi
echo "$VERSION"
