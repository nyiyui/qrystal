#!/bin/bash

pkgname='qrystal'
pkgver='0.1'
pkgrel='1'
arch=('x86_64')
url='https://nyiyui.ca/qrystal'
license=('GPL')
depends=('wireguard-tools')
makedepends=('go')
checkdepends=('go')
noextract=('.')
backup=('etc/qrystal/node-config.yml' 'etc/qrystal/cs-config.yml')
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
	cp build2/runner "$pkgdir/usr/bin/qrystal-runner"
	cp build2/gen-keys "$pkgdir/usr/bin/qrystal-gen-keys"
	cp build2/cs "$pkgdir/usr/bin/qrystal-cs"
	mkdir -p "$pkgdir/opt/qrystal"
	cp build2/runner-mio "$pkgdir/opt/qrystal/"
	cp build2/runner-node "$pkgdir/opt/qrystal/"
	cp build2/dev-add.sh "$pkgdir/opt/qrystal/"
	cp build2/dev-remove.sh "$pkgdir/opt/qrystal/"
}
