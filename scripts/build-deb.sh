#!/usr/bin/env bash
set -euo pipefail
ROOT="$(cd "$(dirname "$0")/.." && pwd)"
VERSION="${1:-0.1.0}"
ARCH="${2:-amd64}"
command -v go >/dev/null || { echo "Instale Go 1.22+"; exit 1; }
command -v dpkg-deb >/dev/null || { echo "Instale dpkg-deb"; exit 1; }
BUILD="$ROOT/build/netvigia_${VERSION}_${ARCH}"
rm -rf "$BUILD"
mkdir -p "$BUILD/DEBIAN" "$BUILD/usr/bin" "$BUILD/etc/netvigia" "$BUILD/var/lib/netvigia" "$BUILD/opt/netvigia/web" "$BUILD/lib/systemd/system"
(cd "$ROOT/agent" && CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -trimpath -ldflags="-s -w" -o "$BUILD/usr/bin/netvigia" .)
cp "$ROOT/index.html" "$BUILD/opt/netvigia/web/index.html"
cp "$ROOT/packaging/config.json" "$BUILD/etc/netvigia/config.json"
cp "$ROOT/packaging/netvigia.service" "$BUILD/lib/systemd/system/netvigia.service"
cat > "$BUILD/DEBIAN/control" <<EOF
Package: netvigia
Version: $VERSION
Section: net
Priority: optional
Architecture: $ARCH
Depends: iputils-ping
Maintainer: NetVigia Project
Description: Serviço independente de monitoramento de redes e servidores
 Monitora disponibilidade ICMP e recebe métricas de agentes autorizados.
EOF
cat > "$BUILD/DEBIAN/postinst" <<'EOF'
#!/bin/sh
set -e
getent group netvigia >/dev/null || groupadd --system netvigia
id netvigia >/dev/null 2>&1 || useradd --system --gid netvigia --home-dir /var/lib/netvigia --shell /usr/sbin/nologin netvigia
chown -R netvigia:netvigia /var/lib/netvigia
chown root:netvigia /etc/netvigia/config.json
if grep -q "GENERATE-A-UNIQUE-KEY-DURING-INSTALL" /etc/netvigia/config.json; then KEY=$(od -An -N32 -tx1 /dev/urandom | tr -d " \\n"); sed -i "s/GENERATE-A-UNIQUE-KEY-DURING-INSTALL/$KEY/" /etc/netvigia/config.json; fi
chown root:netvigia /etc/netvigia/config.json
chmod 0640 /etc/netvigia/config.json
systemctl daemon-reload
systemctl enable netvigia.service
systemctl start netvigia.service || true
printf "NetVigia instalado. Troque api_key em /etc/netvigia/config.json e reinicie o serviço.\n"
EOF
chmod 0755 "$BUILD/DEBIAN/postinst"
cat > "$BUILD/DEBIAN/prerm" <<'EOF'
#!/bin/sh
set -e
if [ "$1" = remove ]; then systemctl stop netvigia.service || true; systemctl disable netvigia.service || true; fi
EOF
chmod 0755 "$BUILD/DEBIAN/prerm"
dpkg-deb --root-owner-group --build "$BUILD" "$ROOT/build/netvigia_${VERSION}_${ARCH}.deb"
echo "Pacote gerado em build/netvigia_${VERSION}_${ARCH}.deb"
