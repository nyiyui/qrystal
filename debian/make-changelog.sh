#!/bin/bash

set -eu

pkgver() {
		printf "r%s.%s" "$(git rev-list --count HEAD)" "$(git rev-parse --short HEAD)"
}

pkgver=$(pkgver)
distrib=UNRELEASED

echo "qrystal ($pkgver) $distrib; urgency=low"
echo

# from https://stackoverflow.com/a/46033999
previous_tag=0
for current_tag in $(git tag --sort=-creatordate); do
    if [ "$previous_tag" != 0 ]; then
        tag="$previous_tag"
    else
        tag="$current_tag"
    fi
    echo "$(git show -s --pretty=format:'* %s (%h)' ${tag})"
    if [ "$previous_tag" != 0 ]; then
        git log ${current_tag}...${previous_tag} --pretty=format:'*  %s [View](https://bitbucket.org/projects/test/repos/my-project/commits/%H)' --reverse | grep -v Merge
    fi
    echo "$(git show -s --pretty=format:'-- %an <%ae> %ad' --date=short ${tag})"
    printf "\n\n"
    previous_tag=${current_tag}
done
