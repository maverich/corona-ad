version="0.0.1"
version_file=VERSION
working_dir=$(shell pwd)
arch="armhf"

clean:
	-rm tpflow

build-go:
	go build -o corona-ad src/service.go

build-go-arm:
	cd ./src;GOOS=linux GOARCH=arm GOARM=6 go build -o corona-ad service.go;cd ../

build-go-amd:
	cd ./src;GOOS=linux GOARCH=amd64 go build -o corona-ad src/service.go;cd ../


configure-arm:
	python ./scripts/config_env.py prod $(version) armhf

configure-amd64:
	python ./scripts/config_env.py prod $(version) amd64

package-tar:
	tar cvzf corona-ad_$(version).tar.gz corona-ad VERSION

package-deb-doc-fh:
	@echo "Packaging application as Futurehome debian package"
	chmod a+x package/debian_fh/DEBIAN/*
	cp ./src/corona-ad package/debian_fh/usr/bin/corona-ad
	cp VERSION package/debian_fh/var/lib/futurehome/corona-ad
	docker run --rm -v ${working_dir}:/build -w /build --name debuild debian dpkg-deb --build package/debian_fh
	@echo "Done"


tar-arm: build-js build-go-arm package-deb-doc-2
	@echo "The application was packaged into tar archive "

deb-arm-fh : clean configure-arm build-go-arm package-deb-doc-fh
	mv package/debian_fh.deb package/build/corona-ad_$(version)_armhf.deb

deb-amd : configure-amd64 build-go-amd package-deb-doc-tp
	mv debian.deb corona-ad_$(version)_amd64.deb

run :
	go run src/service.go -c testdata/var/config.json


.phony : clean
