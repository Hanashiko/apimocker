# Maintainer: Hanashiko <hlichisper@gmail.com>
pkgname=apimocker
pkgver=0.1.4
pkgrel=2
pkgdesc="Lightweight TUI/mock server for running REST API from YAML/JSON description with authentication and query parameter support"
arch=('x86_64')
url="https://github.com/Hanashiko/apimocker"
license=('MIT')
depends=('glibc')
makedepends=('go')
source=('main.go' 'go.mod' 'go.sum' 'LICENSE')
sha256sums=('882ece625d42ef55f72014de8bfdc22007b3eaaab32dd048fdacef9783bc2964'
    '909618e73ddcc1d9bf6fbcd59decd77261c170092791893cabcd77dfcfe2fda8'
    '101759e3fedaca0cb2e1688d9bd73d525f329381cd893ef920ba2b09d9ad40b0'
    '60a21faf5459b93996f566dde48d4bb44218cec03417bbcdd6c4731ef3b31bf5')

build() {
    go build -trimpath -buildmode=pie -ldflags="-linkmode=external -extldflags=-Wl,-z,relro,-z,now -s -w" -o "$pkgname" main.go
}

package() {
    install -Dm755 "$pkgname" "$pkgdir/usr/bin/$pkgname"
    install -Dm644 LICENSE "$pkgdir/usr/share/licenses/$pkgname/LICENSE"
}
