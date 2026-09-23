# Native amd64/arm64 builds; multi-architecture official image indexes are pinned.
FROM golang:1.27.1-bookworm@sha256:69a7b9788769bec032d238959b61854e9ae87f57be9029ec04e9885fabf99195 AS build
RUN apt-get update && apt-get install -y --no-install-recommends qt6-base-dev pkg-config dpkg-dev desktop-file-utils python3 && rm -rf /var/lib/apt/lists/*
ENV CGO_ENABLED=1 CGO_CXXFLAGS="-O0 -g0 -std=c++17"
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download && go mod verify && go build github.com/mappu/miqt/qt6
COPY cmd ./cmd
COPY internal ./internal
COPY scripts ./scripts
COPY packaging ./packaging
COPY LICENSE README.md README.en.md AGENTS.md MEMORY.md CONTRIBUTING.md SECURITY.md ./
COPY DOCS ./DOCS
COPY experiments/qt ./experiments/qt
ARG SOURCE_REVISION=unrecorded
ARG SOURCE_DIRTY=unknown
RUN WEBFENCE_SOURCE_REVISION="$SOURCE_REVISION" WEBFENCE_SOURCE_DIRTY="$SOURCE_DIRTY" sh scripts/package-debian.sh

FROM debian:bookworm-slim@sha256:3783cc01769c7b2b1b83a5c5ad96c815348e28ed7da68e2e3687004faa906251 AS runtime
COPY --from=build /src/dist/*.deb /out/
# slim deliberately excludes docs through dpkg. Retain this package's docs in
# the test container only; never override a user's dpkg policy in the product.
RUN printf 'path-include /usr/share/doc/webfence\npath-include /usr/share/doc/webfence/*\n' > /etc/dpkg/dpkg.cfg.d/zz-webfence-test-docs
RUN apt-get update && apt-get install -y --no-install-recommends /out/*.deb xvfb xauth x11-utils openbox fonts-dejavu-core python3-pyatspi at-spi2-core dbus-x11 && rm -rf /var/lib/apt/lists/*
COPY packaging/debian/test-xcb.sh /opt/webfence-test/test-xcb.sh
COPY packaging/debian/test-atspi.sh packaging/debian/test-atspi.py /opt/webfence-test/
RUN useradd --create-home webfence-test && install -d -m 700 -o webfence-test -g webfence-test /tmp/webfence-runtime
RUN ! command -v go && ! command -v gcc && ! command -v qmake6 && \
    test -f /usr/share/doc/webfence/DOCS/it/DEVELOPMENT.md && \
    test -f /usr/share/doc/webfence/DOCS/en/DEVELOPMENT.md && \
    cmp /usr/share/doc/webfence/LICENSE /usr/share/doc/webfence/copyright && \
    (runuser -u webfence-test -- env XDG_RUNTIME_DIR=/tmp/webfence-runtime QT_QPA_PLATFORM=offscreen webfence --self-test > /out/offscreen.log 2>&1 || { cat /out/offscreen.log; exit 1; }) && \
    (runuser -u webfence-test -- env XDG_RUNTIME_DIR=/tmp/webfence-runtime xvfb-run -a sh /opt/webfence-test/test-xcb.sh > /out/xcb.log 2>&1 || { cat /out/xcb.log; exit 1; }) && \
    (runuser -u webfence-test -- env HOME=/home/webfence-test XDG_RUNTIME_DIR=/tmp/webfence-runtime QT_LINUX_ACCESSIBILITY_ALWAYS_ON=1 QT_QPA_PLATFORM=xcb dbus-run-session -- xvfb-run -a timeout 180 sh /opt/webfence-test/test-atspi.sh > /out/atspi.log 2>&1 || { cat /out/atspi.log; exit 1; })
RUN runuser -u webfence-test -- sh -c 'mkdir -p "$HOME/.config/WebFence" && printf en > "$HOME/.config/WebFence/ui-language"' && \
    apt-get purge -y webfence && test ! -e /usr/bin/webfence && test ! -e /usr/share/applications/webfence.desktop && \
    test "$(cat /home/webfence-test/.config/WebFence/ui-language)" = en && \
    printf 'Install, IT/EN documentation, license, unprivileged offscreen/XCB self-tests, IT/EN AT-SPI bridge regression and purge passed; synthetic user preference preserved.\n' > /out/validation.txt && \
    cd /out && sha256sum ./*.deb > SHA256SUMS

FROM scratch AS package
COPY --from=runtime /out/ /
