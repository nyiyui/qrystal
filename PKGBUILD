#!/bin/env bash
pkgname='qanms'
pkgver='0.1'
pkgrel='1'
arch=('x86_64')
url='https://nyiyui.ca/qanms'
license=('GPL')
depends=('wireguard-tools')
makedepends=('go')
checkdepends=('go')
noextract=('.')
backup=('etc/qanms/node-config.yml' 'etc/qanms/cs-config.yml')
changelog='CHANGELOG.md'
source=()
md5sums=()

arch_to_goarch() {
	case $1  in
		x86_64) printf 'amd64' ;;
	esac
}

build() {
	mkdir build2
	cd build2
	GOOS='linux'
	GOARCH="$(arch_to_goarch $CARCH)"
	export GOOS GOARCH
	go build -o runner-mio ../../cmd/runner-mio
	go build -o runner-node ../../cmd/runner-node
	go build -o runner ../../cmd/runner
	go build -o gen-keys ../../cmd/gen-keys
	go build -o cs ../../cmd/cs
	cp ../../mio/dev-add.sh .
	cp ../../mio/dev-remove.sh .
	cd ..
}

package() {
	mkdir -p "$pkgdir/usr/bin"
	cp build2/runner "$pkgdir/usr/bin/qanms-runner"
	cp build2/gen-keys "$pkgdir/usr/bin/qanms-gen-keys"
	cp build2/cs "$pkgdir/usr/bin/qanms-cs"
	mkdir -p "$pkgdir/opt/qanms"
	cp build2/runner-mio "$pkgdir/opt/qanms/"
	cp build2/runner-node "$pkgdir/opt/qanms/"
	cp build2/dev-add.sh "$pkgdir/opt/qanms/"
	cp build2/dev-remove.sh "$pkgdir/opt/qanms/"
}
