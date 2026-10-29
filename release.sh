#!/bin/sh
# Tags a release and pushes the tag to trigger the release.yml workflow.
set -e

semver_re='^[0-9]+\.[0-9]+\.[0-9]+(-[0-9A-Za-z.-]+)?(\+[0-9A-Za-z.-]+)?$'

bump=""
for arg in "$@"; do
    case "$arg" in
        --patch|-p) bump="patch" ;;
        --minor|-m) bump="minor" ;;
        *)
            echo "Error: unknown option '$arg' (expected --patch/-p or --minor/-m)." >&2
            exit 1
            ;;
    esac
done

git fetch origin --tags >/dev/null 2>&1 || true
current_tag=$(git tag -l 'v[0-9]*.[0-9]*.[0-9]*' --sort=-v:refname | head -n1)
current_tag=${current_tag#v}

if [ -n "$current_tag" ]; then
    major=$(echo "$current_tag" | cut -d. -f1)
    minor=$(echo "$current_tag" | cut -d. -f2)
    patch=$(echo "$current_tag" | cut -d. -f3)
    echo "Current version: v$current_tag"
    echo "Suggested next patch: v$major.$minor.$((patch + 1))"
    echo "Suggested next minor: v$major.$((minor + 1)).0"
else
    echo "Current version: none (no vX.Y.Z tags found)"
fi

if [ -n "$bump" ]; then
    if [ -z "$current_tag" ]; then
        echo "Error: --$bump requires an existing vX.Y.Z tag to compute the next version." >&2
        exit 1
    fi
    if [ "$bump" = "patch" ]; then
        version="$major.$minor.$((patch + 1))"
    else
        version="$major.$((minor + 1)).0"
    fi
    echo "Using suggested next $bump: v$version"
else
    printf 'Version to release (e.g. 0.1.0 or v0.1.0): '
    read -r input
    version=${input#v}
fi

if ! printf '%s' "$version" | grep -Eq "$semver_re"; then
    echo "Error: '$version' is not a valid semantic version (expected X.Y.Z)." >&2
    exit 1
fi

tag="v$version"

if git rev-parse -q --verify "refs/tags/$tag" >/dev/null; then
    echo "Error: tag $tag already exists." >&2
    exit 1
fi

echo "Fetching origin..."
git fetch origin main

current_branch=$(git rev-parse --abbrev-ref HEAD)
if [ "$current_branch" = "main" ]; then
    if [ -n "$(git status --porcelain)" ]; then
        echo "Error: working tree has uncommitted changes. Commit or stash before releasing." >&2
        exit 1
    fi
    git merge --ff-only origin/main
else
    if ! git fetch origin main:main; then
        echo "Error: local main has diverged from origin/main and can't be fast-forwarded." >&2
        echo "Switch to main and resolve it manually, then re-run this script." >&2
        exit 1
    fi
fi

# install.sh downloads the release named by its RELEASE_TAG, so it has to be
# bumped in the commit that gets tagged. Release candidates are left alone.
install_sh="deploy/pi/install.sh"
pinned=$(sed -n 's/^RELEASE_TAG="\(.*\)"$/\1/p' "$install_sh")
update_install=""
case "$version" in
    *-*) ;;
    *)
        if [ "$pinned" = "$tag" ]; then
            :
        elif [ "$current_branch" != "main" ]; then
            echo "Warning: $install_sh pins ${pinned:-nothing} and is not bumped from branch '$current_branch'." >&2
            echo "Switch to main and re-run to have it updated, or bump it by hand." >&2
        else
            printf '%s pins %s. Point it at %s, commit and push to origin/main first? [Y/n] ' \
                "$install_sh" "${pinned:-nothing}" "$tag"
            read -r bump_confirm
            case "$bump_confirm" in
                n|N|no|NO) echo "Leaving $install_sh as it is." ;;
                *) update_install=1 ;;
            esac
        fi
        ;;
esac

main_sha=$(git rev-parse main)
if [ -n "$update_install" ]; then
    printf 'About to bump %s, push that commit to main (on top of %s), then tag %s and push it. Continue? [y/N] ' \
        "$install_sh" "$main_sha" "$tag"
else
    printf 'About to tag %s at main (%s) and push to origin. Continue? [y/N] ' "$tag" "$main_sha"
fi
read -r confirm
case "$confirm" in
    y|Y|yes|YES) ;;
    *) echo "Aborted."; exit 1 ;;
esac

if [ -n "$update_install" ]; then
    sed "s/^RELEASE_TAG=\".*\"\$/RELEASE_TAG=\"$tag\"/" "$install_sh" >"$install_sh.tmp"
    mv "$install_sh.tmp" "$install_sh"
    git add "$install_sh"
    git commit -m "chore(release): point install.sh at $tag"
    git push origin main
fi

git tag -a "$tag" -m "Release $tag" main
git push origin "$tag"

echo "Pushed $tag. The release workflow will build and publish it."
