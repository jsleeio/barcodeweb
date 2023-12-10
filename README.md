# barcodeweb

## what is this?

This is my first attempt at deploying a Go webservice in Cloudflare
Workers. The webservice renders barcodes (Code 39 or Code 128) and QR codes.

## trying it locally

This repository is setup to do everything via Docker containers, as I
do not want to expose my Mac filesystem to `npm`.

The `Makefile` should be enough machinery to get you up and running.

```
make dev
```

Then, assuming that works, try browsing to: http://localhost:8787/c128/1234567890
## URL structure

### barcode types

* the first URL path component specifies the render type:
  - `2of5`  for non-interleaved 2-of-5 barcodes
  - `2of5i` for interleaved 2-of-5 barcodes
  - `aztec` for compact (4 layer) Aztec codes
  - `c39`   for Code 39 barcodes
  - `c128`  for Code 128 barcodes
  - `qr`    for QR codes
* the remainder of the URL path component after the slash specifies the render
  content

### query parameters

* `width` specifies width in pixels
* `height` specifies height in pixels

Note that each format has its own default sizing --- generally, 165x50 for
barcodes and 200x200 for QR/Aztec.

Sizes smaller than 50x50 will be rejected.

### examples

eg.

* `http://localhost:8787/qr/https://github.com/jsleeio/barcodeweb/`
  (this is broken, as you'll see if you try it; see caveats section at
  the end of this file)

* `http://localhost:8787/qr/hello-world`

* `http://localhost:8787/qr/hello-world?width=400&height=400`

* `http://localhost:8787/aztec/https://github.com/jsleeio/barcodeweb/`
  (this is broken, as you'll see if you try it; see caveats section at
  the end of this file)

* `http://localhost:8787/aztec/hello-world`

* `http://localhost:8787/c128/123456&height=100`

* `http://localhost:8787/c39/123456&width=200&height=100`

## deploying to Cloudflare Workers

### Some one-off preparation

1. create a worker named `barcodeweb`
2. create a Cloudflare API key using the worker template
3. put some Cloudflare configuration data in files in your `$HOME`:
   - in `$HOME/shell-secrets/tokens/cloudflare/email`, put your Cloudflare email address
   - in `$HOME/shell-secrets/tokens/cloudflare/api-key`, put your Cloudflare API key that you created in step 2
   - in `$HOME/shell-secrets/tokens/cloudflare/account`, put your Cloudflare account ID that you want to deploy into

The `run-wrangler` helper can be used for other Cloudflare Workers
things if you need to use those also.

### actually deploy

```
make deploy
```

That should be that!

## caveats

### handling of URLs with URLs in them

The Go `net/http` package tidies up request URLS, including coalescing
consecutive slashes, which makes generating QR and Aztec images
containing URLs impossible without moving the QR/Aztec content to a query
parameter.

Specifically, URLs like `https://localhost:8787/qr/https://my.website/` will
generate a 301 redirect to `https://localhost:8787/qr/https:/my.website/` ---
the `//` has been replaced with `/`.

I currently don't see a way around this, as it even appears to do it when the
`//` is URL-encoded. More info here: https://github.com/golang/go/issues/21955
