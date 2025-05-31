# Maintainer: Hanashiko hlichisper@gmail.com
pkgname=apimocker
pkgver=0.1.1
pkgrel=1
pkgdesc="Lightweight TUI/mock server for running REST API from YAML/JSON description with query parameter support"
arch=('x86_64')
url="https://github.com/Hanashiko/apimocker"
license=('MIT')
depends=()
makedepends=('go')
source=('main.go' 'go.mod' 'go.sum')
sha256sums=('1478367f022aad1f63fe54fb4a89e3225273ac8103041631db3acb3bb7cc6291' 'b49a5f276f1d5e39e58c51fb3ffea6315539c2ee91e2f7559ef85ae3b39eb61e' '6a903795ec27f72787505588d99c5b4cecbb93e967d811d5469bdcf4263087be')

build() {
    cd "$srcdir"
    go build -o apimocker main.go
}

package() {
    install -Dm755 "$srcdir/mock-api-server" "$pkgdir/usr/bin/apimocker"
}
