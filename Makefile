.PHONY: wrangler assets dev deploy clean

wrangler: .built-wrangler-docker
assets: .downloaded-assets

.built-wrangler-docker: Dockerfile.wrangler
	docker build -t jsleeio/wrangler:latest -f Dockerfile.wrangler .
	touch .built-wrangler-docker

.downloaded-assets:
	go run github.com/syumai/workers/cmd/workers-assets-gen@v0.18.0
	touch .downloaded-assets

./build/app.wasm: main.go go.mod go.sum
	mkdir -p ./build .tinygo-cache
	docker run \
		--rm \
		--env=GOCACHE=/src/.tinygo-cache \
		--env=GOFLAGS=-buildvcs=false \
		--volume="$(CURDIR):/src" \
		--workdir=/src \
		tinygo/tinygo:0.30.0 \
		tinygo build \
			-o ./build/app.wasm\
			-target=wasm \
			.

dev: .built-wrangler-docker .downloaded-assets ./build/app.wasm
	./run-wrangler dev --ip \*

deploy: .built-wrangler-docker .downloaded-assets ./build/app.wasm
	./run-wrangler deploy

clean:
	rm -rf build .tinygo-cache .built-wrangler-docker .downloaded-assets .wrangler
	docker rmi -f jsleeio/wrangler:latest
