.PHONY: generate
generate:
	cd upstream; npm install; ./build.sh
	cd twembed; go run embed_mk.go
