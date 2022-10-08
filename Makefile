build2:
	mkdir -p build2
	go build -o build2/runner-mio ./cmd/runner-mio
	go build -o build2/runner-node ./cmd/runner-node
	go build -o build2/runner ./cmd/runner
	go build -o build2/gen-keys ./cmd/gen-keys
	go build -o build2/cs ./cmd/cs

pre_install:
	systemctl stop qrystal-runner
	systemctl stop qrystal-cs

install: build2
	mkdir -p "${pkgdir}/usr/lib/sysusers.d"
	cp './config/sysusers.conf' "${pkgdir}/usr/lib/sysusers.d/qrystal.conf"
	systemctl restart systemd-sysusers
	mkdir -p "${pkgdir}/usr/bin"
	cp build2/runner "${pkgdir}/usr/bin/qrystal-runner"
	cp build2/gen-keys "${pkgdir}/usr/bin/qrystal-gen-keys"
	cp build2/cs "${pkgdir}/usr/bin/qrystal-cs"
	mkdir -p "${pkgdir}/opt/qrystal"
	cp build2/runner-mio "${pkgdir}/opt/qrystal/"
	cp build2/runner-node "${pkgdir}/opt/qrystal/"
	cp ./mio/dev-add.sh "${pkgdir}/opt/qrystal/"
	cp ./mio/dev-remove.sh "${pkgdir}/opt/qrystal/"
	mkdir -p "${pkgdir}/etc/qrystal"
	chown root:qrystal-node "${pkgdir}/etc/qrystal"
	chmod 770 "${pkgdir}/etc/qrystal"
	cp -n \
		'./config/cs-config.yml' \
		'./config/runner-config.yml' \
		"${pkgdir}/etc/qrystal/"
	touch "${pkgdir}/etc/qrystal/cs-backport.yml"
	chmod 600 "${pkgdir}/etc/qrystal/cs-backport.yml"
	chown qrystal-node:qrystal-node "${pkgdir}/etc/qrystal/cs-backport.yml"
	cp -n \
		'./config/node-config.yml' \
		"${pkgdir}/etc/qrystal/"
	chmod 750 "${pkgdir}/etc/qrystal"
	chown root:qrystal-node "${pkgdir}/etc/qrystal"
	mkdir -p "${pkgdir}/usr/lib/systemd/system"
	cp './config/runner.service' "${pkgdir}/usr/lib/systemd/system/qrystal-runner.service"
	cp './config/cs.service' "${pkgdir}/usr/lib/systemd/system/qrystal-cs.service"
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

.PHONY: install uninstall
