# Maintainer: Hanashiko <hlichisper@gmail.com>
pkgname=apimocker
pkgver=0.1.4
pkgrel=1
pkgdesc="Lightweight TUI/mock server for running REST API from YAML/JSON description with authentication and query parameter support"
arch=('x86_64')
url="https://github.com/Hanashiko/apimocker"
license=('MIT')
depends=('glibc')
makedepends=('go')
source=('main.go' 'go.mod' 'go.sum' 'LICENSE')
sha256sums=('44f4c7fce3e92159e256e80858d566a878342afa460693e30235982b8fac19a2'
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
