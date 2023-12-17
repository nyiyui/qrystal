src=.
flags = -race
tags = sdnotiy
ldflags-mio = -X github.com/nyiyui/qrystal/mio.CommandBash=${shell which bash}
ldflags-mio += -X github.com/nyiyui/qrystal/mio.CommandWg=${shell which wg}
ldflags-mio += -X github.com/nyiyui/qrystal/mio.CommandWgQuick=${shell which wg-quick}
ldflags-node = -X github.com/nyiyui/qrystal/node.CommandIp=${shell which ip}
ldflags-node += -X github.com/nyiyui/qrystal/node.CommandIptables=${shell which iptables}
ldflags-runner = -X github.com/nyiyui/qrystal/runner.NodeUser=qrystal-node

all: cs-admin cs gen-keys runner-hokuto runner-mio runner-node runner sd-notify-test

cs-admin:
	go build ${flags} -tags "${tags}" -o cs-push ${src}/cmd/cs-admin

cs:
	go build ${flags} -tags "${tags}" -o cs ${src}/cmd/cs

gen-keys:
	go build ${flags} -tags "${tags}" -o gen-keys ${src}/cmd/gen-keys

runner-hokuto:
	go build ${flags} -tags "${tags}" -o runner-hokuto ${src}/cmd/runner-hokuto

runner-mio:
	go build ${flags} -tags "${tags}" -ldflags "${ldflags-mio}" -o runner-mio ${src}/cmd/runner-mio

runner-node:
	go build ${flags} -tags "${tags}" -ldflags "${ldflags-node}" -o runner-node ${src}/cmd/runner-node

runner:
	go build ${flags} -tags "${tags}" -ldflags "${ldflags-runner}" -o runner ${src}/cmd/runner

sd-notify-test:
	go build ${flags} -tags "${tags}" -o sd-notify-test ${src}/cmd/sd-notify-test

install-cs-push: cs-push
	install -m 755 -o root -g root $< ${pkgdir}/usr/bin/qrystal-cs-push

uninstall-cs-push:
	rm -f ${pkgdir}/usr/bin/qrystal-cs-push

install-node: runner runner-hokuto runner-mio runner-node
	mkdir -p "${pkgdir}/opt/qrystal-node"
	install -m 755 -o root -g root runner ${pkgdir}/opt/qrystal-node/runner
	install -m 755 -o root -g root runner-hokuto ${pkgdir}/opt/qrystal-node/runner-hokuto
	install -m 755 -o root -g root runner-mio ${pkgdir}/opt/qrystal-node/runner-mio
	install -m 755 -o root -g root runner-node ${pkgdir}/opt/qrystal-node/runner-node
	mkdir -p "${pkgdir}/usr/lib/sysusers.d"
	install -m 644 '${src}/config/sysusers-node.conf' "${pkgdir}/usr/lib/sysusers.d/qrystal-node.conf"
	systemctl restart systemd-sysusers
	#
	mkdir -p "${pkgdir}/etc/qrystal-node"
	chown root:qrystal-node "${pkgdir}/etc/qrystal-node"
	chmod 755 "${pkgdir}/etc/qrystal-node"
	install '${src}/config/node-config.yml' "${pkgdir}/etc/qrystal-node/"
	chown root:qrystal-node "${pkgdir}/etc/qrystal-node/node-config.yml"
	chmod 640 "${pkgdir}/etc/qrystal-node/node-config.yml"
	#
	mkdir -p "${pkgdir}/usr/lib/systemd/system"
	install '${src}/config/node.service' "${pkgdir}/usr/lib/systemd/system/qrystal-node.service"
	systemctl daemon-reload

install-cs: cs
	mkdir -p "${pkgdir}/opt/qrystal"
	install -m 755 -o root -g root $< ${pkgdir}/opt/qrystal/qrystal-cs
	mkdir -p "${pkgdir}/usr/lib/sysusers.d"
	install -m 644 '${src}/config/sysusers-cs.conf' "${pkgdir}/usr/lib/sysusers.d/qrystal-cs.conf"
	systemctl restart systemd-sysusers
	#
	mkdir -p "${pkgdir}/etc/qrystal"
	chown root:qrystal-cs "${pkgdir}/etc/qrystal"
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
