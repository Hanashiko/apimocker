# Maintainer: Hanashiko hlichisper@gmail.com
pkgname=apimocker
pkgver=0.1.3
pkgrel=1
pkgdesc="Lightweight TUI/mock server for running REST API from YAML/JSON description with query parameter support"
arch=('x86_64')
url="https://github.com/Hanashiko/apimocker"
license=('MIT')
depends=()
makedepends=('go')
source=('main.go' 'go.mod' 'go.sum')
sha256sums=('dca9eb12cab69223d421da830e684f004497fd88e4ab5381bc55ed1eabd09f7a' '4f113e8623b72f824d04341806d11a44042b8b2849c43dff5f3fcbb97e906f76' '6a903795ec27f72787505588d99c5b4cecbb93e967d811d5469bdcf4263087be')

build() {
    cd "$srcdir"
    go build -o apimocker main.go
}

package() {
    install -Dm755 "$srcdir/apimocker" "$pkgdir/usr/bin/apimocker"
}
