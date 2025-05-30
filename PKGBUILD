# Maintainer: Hanashiko hlichisper@gmail.com
pkgname=mock-api-server
pkgver=0.1.0
pkgrel=1
pkgdesc="Lightweight TUI/mock server for running REST API from YAML/JSON description"
arch=('x86_64')
url="https://github.com/Hanashiko/mock-api-server"
license=('MIT')
depends=()
makedepends=('go')
source=('main.go' 'go.mod' 'go.sum')
sha256sums=('SKIP' 'SKIP' 'SKIP')

build() {
    cd "$srcdir"
    go build -o mock-api-server main.go
}

package() {
    install -Dm755 "$srcdir/mock-api-server" "$pkgdir/usr/bin/mock-api-server"
}
