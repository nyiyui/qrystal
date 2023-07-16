src=.
path=${shell pwd}
tags = sdnotiy
ldflags-mio = -X github.com/nyiyui/qrystal/mio.CommandBash=${shell which bash}
ldflags-mio += -X github.com/nyiyui/qrystal/mio.CommandWg=${shell which wg}
ldflags-mio += -X github.com/nyiyui/qrystal/mio.CommandWgQuick=${shell which wg-quick}
ldflags-node = -X github.com/nyiyui/qrystal/node.CommandIp=${shell which ip}
ldflags-node += -X github.com/nyiyui/qrystal/node.CommandIptables=${shell which iptables}
ldflags-runner = -X github.com/nyiyui/qrystal/runner.NodeUser=qrystal-node

all: cs-admin cs gen-keys runner-hokuto runner-mio runner-node runner sd-notify-test

cs-admin:
	go build -race -tags "${tags}" -o ${path}/cs-push ${src}/cmd/cs-admin

cs:
	go build -race -tags "${tags}" -o ${path}/cs ${src}/cmd/cs

gen-keys:
	go build -race -tags "${tags}" -o ${path}/gen-keys ${src}/cmd/gen-keys

runner-hokuto:
	go build -race -tags "${tags}" -o ${path}/runner-hokuto ${src}/cmd/runner-hokuto

runner-mio:
	go build -race -tags "${tags}" -ldflags "${ldflags-mio}" -o ${path}/runner-mio ${src}/cmd/runner-mio

runner-node:
	go build -race -tags "${tags}" -ldflags "${ldflags-node}" -o ${path}/runner-node ${src}/cmd/runner-node

runner:
	go build -race -tags "${tags}" -ldflags "${ldflags-runner}" -o ${path}/runner ${src}/cmd/runner

sd-notify-test:
	go build -race -tags "${tags}" -o ${path}/sd-notify-test ${src}/cmd/sd-notify-test

install-cs-push: cs-push
	install -m 755 -o root -g root $@ ${pkdir}/usr/bin/qrystal-cs-push

uninstall-cs-push:
	rm -f ${pkgdir}/usr/bin/qrystal-cs-push

install-cs: cs
	systemctl stop qrystal-cs
	#
	mkdir -p "${pkgdir}/opt/qrystal"
	install -m 755 -o root -g root $@ ${pkdir}/opt/qrystal/qrystal-cs
	mkdir -p "${pkgdir}/usr/lib/sysusers.d"
	install -m 644 '${src}/config/sysusers-cs.conf' "${pkgdir}/usr/lib/sysusers.d/qrystal-cs.conf"
	systemctl restart systemd-sysusers
	#
	mkdir -p "${pkgdir}/etc/qrystal"
	chown root:qrystal-node "${pkgdir}/etc/qrystal"
	chmod 755 "${pkgdir}/etc/qrystal"
	install '${src}/config/cs-config.yml' "${pkgdir}/etc/qrystal/"
	chown root:qrystal-cs "${pkgdir}/etc/qrystal/cs-config.yml"
	chmod 640 "${pkgdir}/etc/qrystal/cs-config.yml"
	#
	mkdir -p "${pkgdir}/usr/lib/systemd/system"
	install '${src}/config/cs.service' "${pkgdir}/usr/lib/systemd/system/qrystal-cs.service"
	systemctl daemon-reload

uninstall-cs:
	systemctl stop qrystal-cs
	rm -f ${pkgdir}/usr/bin/qrystal-cs
	rm -rf "${pkgdir}/opt/qrystal/qrystal-cs" \
		"${pkgdir}/etc/qrystal/cs-config.yml"
	rmdir "${pkgdir}/opt/qrystal" \
		"${pkgdir}/etc/qrystal"
	rm -f "${pkgdir}/usr/lib/sysusers.d/qrystal-cs.conf" \
		"${pkgdir}/usr/lib/systemd/system/qrystal-cs.service"

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
