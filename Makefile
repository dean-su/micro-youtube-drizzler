.PHONY: proto build

proto:
	for d in downloader; do \
		for f in $$d/proto/*.proto; do \
			protoc -I. -I${GOPATH}/src --go_out=plugins=micro:. $$f; \
			echo compiled: $$f; \
		done \
	done

lint:
	./bin/lint.sh

build:
	./bin/build.sh

run:
	docker-compose build
	docker-compose up
