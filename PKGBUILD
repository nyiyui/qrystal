# Maintainer: Ken Shibata <kenxshibata@gmail.com>

pkgname='qrystal'
pkgver=r29.4fbd6a0
pkgrel=1
pkgdesc='An network configuration manager for WireGuard.'
arch=('x86_64')
url='https://nyiyui.ca/qrystal'
license=('GPL')
depends=('wireguard-tools')
makedepends=('go')
checkdepends=('go')
noextract=('.')
backup=(
	'etc/qrystal/node-config.yml'
	'etc/qrystal/cs-config.yml'
	'etc/qrystal/runner-config.yml'
)
changelog='CHANGELOG.md'
source=()
md5sums=()

pkgver() {
	printf "r%s.%s" "$(git rev-list --count HEAD)" "$(git rev-parse --short HEAD)"
}

build() {
	cd ..
	make build2
}

package() {
	cd ..
	make pkgdir="$pkgdir" install
}
