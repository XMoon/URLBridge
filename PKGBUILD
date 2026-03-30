pkgname=urlbridge-host
pkgver=0.0.3
pkgrel=1
pkgdesc="Forward URLs from a Windows VM to the host system browser"
arch=('x86_64' 'aarch64')
url="https://github.com/XMoon/URLBridge"
license=('AGPL-3.0-only')
depends=('xdg-utils')
makedepends=('go')
source=(
  "${pkgname}-${pkgver}.tar.gz::${url}/archive/refs/tags/v${pkgver}.tar.gz"
  'urlbridge-host.service'
)
sha256sums=(
  'SKIP'
  '18632fef3e8b1fe3bee50318d6949feed0a330b23e20f00616eb7deb94ff7070'
)

build() {
  cd "${srcdir}/URLBridge-${pkgver}"

  export CGO_ENABLED=0
  export GOFLAGS="-trimpath -mod=readonly"

  go build -buildmode=pie -ldflags="-s -w" -o urlbridge-host ./cmd/urlbridge-host
}

check() {
  cd "${srcdir}/URLBridge-${pkgver}"

  export CGO_ENABLED=0
  export GOFLAGS="-trimpath -mod=readonly"

  go test ./...
}

package() {
  cd "${srcdir}/URLBridge-${pkgver}"

  install -Dm755 urlbridge-host "${pkgdir}/usr/bin/urlbridge-host"
  install -Dm644 README.md "${pkgdir}/usr/share/doc/${pkgname}/README.md"
  install -Dm644 LICENSE "${pkgdir}/usr/share/licenses/${pkgname}/LICENSE"
  install -Dm644 "${srcdir}/urlbridge-host.service" "${pkgdir}/usr/lib/systemd/user/urlbridge-host.service"
}
