#!/bin/bash

set -eu

pkgver() {
		printf "%s.%s" "$(git rev-list --count HEAD)" "$(git rev-parse --short HEAD)"
}

pkgver=$(pkgver | tee ./pkgver)
distrib=UNRELEASED

cat << EOF
qrystal ($pkgver-1) $distrib; urgency=low

* Please see the Git log for changes.
-- Nyiyui <+@nyiyui.ca> $(date '+%Y-%m-%d')
EOF

## from https://stackoverflow.com/a/46033999
#previous_tag=0
#for current_tag in $(git tag --sort=-creatordate); do
#    echo "qrystal ($pkgver) $distrib; urgency=low"
#    if [ "$previous_tag" != 0 ]; then
#        tag="$previous_tag"
#    else
#        tag="$current_tag"
#    fi
#    echo tag ${tag}
#    git log -n 1 -s --pretty=format:'%s (%h)' ${tag}
#    echo
#    if [ "$previous_tag" != 0 ]; then
#        git log ${current_tag}...${previous_tag} --pretty=format:'*  %s (%h)' --reverse | grep -v Merge
#        git log -n -1 -s --pretty=format:'-- %an <%ae> %ad' --date=short ${tag}
#        printf "\n\n"
#    fi
#    previous_tag=${current_tag}
#done
