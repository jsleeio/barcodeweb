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

Then, assuming that works, try browsing to: http://localhost:8787/?code=1234567890

## query parameters

Only two query parameters are currently supported:

* `kind` specifies the barcode type:
  - `2of5` for non-interleaved 2-of-5 barcodes
  - `2of5i` for interleaved 2-of-5 barcodes
  - `aztec` for compact (4 layer) Aztec codes
  - `code39` for Code 39 barcodes
  - `code128` for Code 128 barcodes
  - `qr` for QR codes
* `code` specifies the content of the barcode, such as `1234567890`

eg.

http://localhost:8787/?kind=qr&code=https://github.com/jsleeio/barcodeweb/

http://localhost:8787/?code=123456

http://localhost:8787/?code=123456&kind=code128 (equivalent to the above)

http://localhost:8787/?code=123456&kind=code39

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
