all:
	make -C cmd/gmi-upload

install:
	install -v -m 0755 cmd/gmi-upload/gmi-upload /usr/local/bin/gmi-upload
	install -v -m 0755 assets/gmi-putdir /usr/local/bin/gmi-putdir
	install -v -m 0644 init/gmi-putdir.service /etc/systemd/system/
	install -v -m 0644 init/gmi-putdir.timer /etc/systemd/system/
	systemctl daemon-reload

uninstall:
	rm -v /usr/local/bin/gmi-upload
	rm -v /usr/local/bin/gmi-putdir
	rm /etc/systemd/system/gmi-putdir.service
	systemctl disable --now gmi-putdir.timer
	rm /etc/systemd/system/gmi-putdir.timer
