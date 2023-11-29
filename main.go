// package main is the main package
package main

import (
	"fmt"
	"image/png"
	"io"
	"log"
	"net/http"

	"github.com/syumai/workers"
	"github.com/syumai/workers/cloudflare"
	"github.com/syumai/workers/cloudflare/cache"

	"github.com/boombuler/barcode"
	"github.com/boombuler/barcode/code128"
	"github.com/boombuler/barcode/code39"
	"github.com/boombuler/barcode/qr"
)

func renderCode39(content string) (barcode.Barcode, error) {
	return code39.Encode(content, true, false)
}

func renderCode128(content string) (barcode.Barcode, error) {
	return code128.Encode(content)
}

func renderQR(content string) (barcode.Barcode, error) {
	return qr.Encode(content, qr.M, qr.Auto)
}

type renderFunc func(content string) (barcode.Barcode, error)

// render actually renders the barcode. Easier to handle errors with this in a
// separate function, and the handler was getting too long.
func render(renderers map[string]renderFunc, kind, code string) (b barcode.Barcode, status int, err error) {
	status = http.StatusBadRequest
	renderer, ok := renderers[kind]
	if !ok {
		err = fmt.Errorf("invalid barcode type %q; valid types are: code39 code128 qr", kind)
		return
	}
	if code == "" {
		err = fmt.Errorf("must supply a 'code' query parameter")
		return
	}
	b, err = renderer(code)
	if err != nil {
		status = http.StatusInternalServerError
		err = fmt.Errorf("error rendering %q barcode with value %q: %v", kind, code, err)
	}
	status = http.StatusOK
	return
}

func handlerfunc(renderers map[string]renderFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		rw := responseWriter{ResponseWriter: w}
		c := cache.New()
		res, err := c.Match(r, nil)
		if err != nil {
			log.Printf("error retrieving from cache: %v", err)
		} else {
			if res != nil {
				// cache HIT
				rw.WriteHeader(res.StatusCode)
				for key, values := range res.Header {
					for _, value := range values {
						rw.Header().Add(key, value)
					}
				}
				rw.Header().Add("x-message", "cache from worker")
				io.Copy(rw.ResponseWriter, res.Body)
				return
			}
		}
		// cache MISS
		kind := r.URL.Query().Get("kind")
		code := r.URL.Query().Get("code")
		if kind == "" {
			kind = "code128"
		}
		bc, status, err := render(renderers, kind, code)
		rw.Header().Set("cache-control", "public, max-age=604800")
		if err != nil {
			log.Printf("render error: %v", err)
		} else {
			bc, _ = barcode.Scale(bc, 200, 200)
			rw.Header().Set("content-type", "image/png")
			png.Encode(w, bc)
		}
		rw.WriteHeader(status)
		// populate Cloudflare cache. Errors are cacheable too because the result
		// for a given set of inputs will never change
		cloudflare.WaitUntil(ctx, func() {
			err := c.Put(r, rw.ToHTTPResponse())
			if err != nil {
				log.Printf("Cloudflare cache put: %v", err)
			}
		})
	}
}

func main() {
	renderers := map[string]renderFunc{
		"code39":  renderCode39,
		"code128": renderCode128,
		"qr":      renderQR,
	}
	handler := handlerfunc(renderers)
	workers.Serve(handler)
}
