# Maintainer: Hanashiko <hlichisper@gmail.com>
pkgname=apimocker
pkgver=0.1.3
pkgrel=2
pkgdesc="Lightweight TUI/mock server for running REST API from YAML/JSON description with query parameter support"
arch=('x86_64')
url="https://github.com/Hanashiko/apimocker"
license=('MIT')
depends=('glibc')
makedepends=('go')
source=('main.go' 'go.mod' 'go.sum' 'LICENSE')
sha256sums=('12783b2c53e4cf21fe8a3e253712234ca7029fe8653601d5d41b6a89673184d6' 
    '4f113e8623b72f824d04341806d11a44042b8b2849c43dff5f3fcbb97e906f76' 
    '6a903795ec27f72787505588d99c5b4cecbb93e967d811d5469bdcf4263087be' 
    '60a21faf5459b93996f566dde48d4bb44218cec03417bbcdd6c4731ef3b31bf5')

build() {
    go build -trimpath -buildmode=pie -ldflags="-linkmode=external -extldflags=-Wl,-z,relro,-z,now -s -w" -o "$pkgname" main.go
}

package() {
    install -Dm755 "$pkgname" "$pkgdir/usr/bin/$pkgname"
    install -Dm644 LICENSE "$pkgdir/usr/share/licenses/$pkgname/LICENSE"
}
