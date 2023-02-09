src=.
path=${shell pwd}
ldflags-mio = -X github.com/nyiyui/qrystal/mio.CommandBash=${shell which bash}
ldflags-mio += -X github.com/nyiyui/qrystal/mio.CommandWg=${shell which wg}
ldflags-mio += -X github.com/nyiyui/qrystal/mio.CommandWgQuick=${shell which wg-quick}
ldflags-node = -X github.com/nyiyui/qrystal/node.CommandIp=${shell which ip}
ldflags-node += -X github.com/nyiyui/qrystal/node.CommandIptables=${shell which iptables}
ldflags-runner = -X github.com/nyiyui/qrystal/runner.NodeUser=qrystal-node

all: runner-mio runner-hokuto runner-node runner gen-keys cs

runner-mio:
	cd ${src} && go build -ldflags "${ldflags-mio}" -o ${path}/runner-mio ${src}/cmd/runner-mio

runner-hokuto:
	cd ${src} && go build -o ${path}/runner-hokuto ${src}/cmd/runner-hokuto

runner-node:
	cd ${src} && go build -ldflags "${ldflags-node}" -o ${path}/runner-node ${src}/cmd/runner-node

runner:
	cd ${src} && go build -ldflags "${ldflags-runner}" -o ${path}/runner ${src}/cmd/runner

gen-keys:
	cd ${src} && go build -o ${path}/gen-keys ${src}/cmd/gen-keys

cs:
	cd ${src} && go build -o ${path}/cs ${src}/cmd/cs

cs-push:
	cd ${src} go build -o ${path}/cs-push ${src}/cmd/cs-push

install-cs-push: cs-push
	install -m 755 -o root -g root $@ ${pkdir}/usr/bin/qrystal-cs-push

uninstall-cs-push:
	rm -f ${pkgdir}/usr/bin/qrystal-cs-push

pre_install:
	systemctl stop qrystal-runner
	systemctl stop qrystal-cs

post_install:
	systemctl start qrystal-runner
	systemctl start qrystal-cs

install: runner-mio runner-hokuto runner-node runner gen-keys cs
	mkdir -p "${pkgdir}/usr/lib/sysusers.d"
	install -m 644 '${src}/config/sysusers.conf' "${pkgdir}/usr/lib/sysusers.d/qrystal.conf"
	systemctl restart systemd-sysusers
	mkdir -p "${pkgdir}/usr/bin"
	install -o root -g root -m 555 ${path}/runner   "${pkgdir}/usr/bin/qrystal-runner"
	install -o root -g root -m 555 ${path}/gen-keys "${pkgdir}/usr/bin/qrystal-gen-keys"
	install -o root -g root -m 555 ${path}/cs       "${pkgdir}/usr/bin/qrystal-cs"
	mkdir -p "${pkgdir}/opt/qrystal"
	install -o root -g root -m 500 \
		${path}/runner-mio \
		${path}/runner-hokuto \
		${src}/mio/dev-add.sh \
		${src}/mio/dev-remove.sh \
	  "${pkgdir}/opt/qrystal/"
	install -o root -g root -m 555 \
		${path}/runner-node \
	  "${pkgdir}/opt/qrystal/"
	mkdir -p "${pkgdir}/etc/qrystal"
	chown root:qrystal-node "${pkgdir}/etc/qrystal"
	chmod 755 "${pkgdir}/etc/qrystal"
	cp -n \
		'${src}/config/cs-config.yml' \
		'${src}/config/runner-config.yml' \
		'${src}/config/node-config.yml' \
		"${pkgdir}/etc/qrystal/"
	chown root:qrystal-cs "${pkgdir}/etc/qrystal/cs-config.yml"
	chmod 640 "${pkgdir}/etc/qrystal/cs-config.yml"
	chown root:qrystal-node "${pkgdir}/etc/qrystal/node-config.yml"
	chmod 640 "${pkgdir}/etc/qrystal/node-config.yml"
	chmod 600 "${pkgdir}/etc/qrystal/runner-config.yml"
	mkdir -p "${pkgdir}/usr/lib/systemd/system"
	install '${src}/config/runner.service' "${pkgdir}/usr/lib/systemd/system/qrystal-runner.service"
	install '${src}/config/cs.service' "${pkgdir}/usr/lib/systemd/system/qrystal-cs.service"
	systemctl daemon-reload

uninstall:
	rm -rf "${pkgdir}/opt/qrystal" \
		"${pkgdir}/etc/qrystal"
	rm -f "${pkgdir}/usr/bin/qrystal-runner" \
		"${pkgdir}/usr/bin/qrystal-gen-keys" \
		"${pkgdir}/usr/bin/qrystal-cs" \
		"${pkgdir}/usr/lib/sysusers.d/qrystal.conf" \
		"${pkgdir}/usr/lib/systemd/system/qrystal-runner.service" \
		"${pkgdir}/usr/lib/systemd/system/qrystal-cs.service"
	systemctl daemon-reload

test:
	go test ./...
	go vet ./...
	nix flake check

.PHONY: install uninstall all test
