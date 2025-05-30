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
sha256sums=('2da1159a7c7249bfece36e0bd39f438a1fa7e11fcb837d653b1cbfb773283a91' 'b49a5f276f1d5e39e58c51fb3ffea6315539c2ee91e2f7559ef85ae3b39eb61e' '6a903795ec27f72787505588d99c5b4cecbb93e967d811d5469bdcf4263087be')

build() {
    cd "$srcdir"
    go build -o mock-api-server main.go
}

package() {
    install -Dm755 "$srcdir/mock-api-server" "$pkgdir/usr/bin/mock-api-server"
}
