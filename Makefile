all:
	make -C cmd/gmi-upload
	make -C cmd/gmi-stat

install:
	install -v -m 0755 cmd/gmi-upload/gmi-upload /usr/local/bin/gmi-upload
	install -v -m 0755 cmd/gmi-stat/gmi-stat /usr/local/bin/gmi-stat

uninstall:
	rm -v /usr/local/bin/gmi-upload
	rm -v /usr/local/bin/gmi-stat
