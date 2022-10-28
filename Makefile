src=.
path=${shell pwd}

build2:
	mkdir -p build2
	cd ${src} && go build -o ${path}/build2/runner-mio ${src}/cmd/runner-mio
	cd ${src} && go build -o ${path}/build2/runner-node ${src}/cmd/runner-node
	cd ${src} && go build -o ${path}/build2/runner ${src}/cmd/runner
	cd ${src} && go build -o ${path}/build2/gen-keys ${src}/cmd/gen-keys
	cd ${src} && go build -o ${path}/build2/cs ${src}/cmd/cs

cs-push:
	cd ${src} go build -o ${path}/cs-push ${src}/cmd/cs-push

tray:
	cd ${src} go build -o ${path}/tray ${src}/cmd/tray

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

install: build2
	mkdir -p "${pkgdir}/usr/lib/sysusers.d"
	install -m 644 '${src}/config/sysusers.conf' "${pkgdir}/usr/lib/sysusers.d/qrystal.conf"
	systemctl restart systemd-sysusers
	mkdir -p "${pkgdir}/usr/bin"
	install -o root -g root -m 555 build2/runner   "${pkgdir}/usr/bin/qrystal-runner"
	install -o root -g root -m 555 build2/gen-keys "${pkgdir}/usr/bin/qrystal-gen-keys"
	install -o root -g root -m 555 build2/cs       "${pkgdir}/usr/bin/qrystal-cs"
	mkdir -p "${pkgdir}/opt/qrystal"
	install -o root -g root -m 500 \
		build2/runner-mio \
		${src}/mio/dev-add.sh \
		${src}/mio/dev-remove.sh \
	  "${pkgdir}/opt/qrystal/"
	install -o root -g root -m 555 \
		build2/runner-node \
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
	touch "${pkgdir}/etc/qrystal/cs-backport.yml"
	chmod 600 "${pkgdir}/etc/qrystal/cs-backport.yml"
	chown qrystal-cs:qrystal-cs "${pkgdir}/etc/qrystal/cs-backport.yml"
	touch "${pkgdir}/etc/qrystal/db"
	chmod 600 "${pkgdir}/etc/qrystal/db"
	chown qrystal-cs:qrystal-cs "${pkgdir}/etc/qrystal/db"
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

.PHONY: install uninstall
