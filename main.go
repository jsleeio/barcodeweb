// package main is the main package
package main

import (
	"fmt"
	"image/png"
	"io"
	"log"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/syumai/workers"
	"github.com/syumai/workers/cloudflare"
	"github.com/syumai/workers/cloudflare/cache"

	"github.com/boombuler/barcode"
	"github.com/boombuler/barcode/aztec"
	"github.com/boombuler/barcode/code128"
	"github.com/boombuler/barcode/code39"
	"github.com/boombuler/barcode/qr"
	"github.com/boombuler/barcode/twooffive"
)

func render2of5(content string) (barcode.Barcode, error) {
	return twooffive.Encode(content, false)
}

func render2of5interleaved(content string) (barcode.Barcode, error) {
	return twooffive.Encode(content, true)
}

func renderAztec(content string) (barcode.Barcode, error) {
	return aztec.Encode([]byte(content), 23, 4)
}

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

// ignorecasefirstpath is an http middleware that translates only the first
// "word" of a path to lowercase. For example:
//
// /FOO/BAR     => /foo/BAR
// /foo/BAR        (unchanged)
// /FOO/BAR/BAZ => /foo/BAR/BAZ
func ignorecasefirstpath(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		pathbits := strings.Split(r.URL.Path, "/")
		// FIXME: This breaks handling of qr/aztec codes with URLs in them.
		//        The // in https:// gets replaced with /
		//        This seems to be a known thing...
		//        https://github.com/golang/go/issues/21955
		if len(pathbits) > 1 {
			oldpath := r.URL.Path
			r.URL.Path = strings.Replace(r.URL.Path, "/"+pathbits[1], "/"+strings.ToLower(pathbits[1]), 1)
			log.Printf("ignorecasefirstpath: %q => %q", oldpath, r.URL.Path)
		}
		next.ServeHTTP(w, r)
	})
}

// logger is an http middleware that just logs requests
func logger(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		log.Printf("request: %s => %s %s", r.RemoteAddr, r.Method, r.URL.String())
		next.ServeHTTP(w, r)
	})
}

func caching(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		rw := responseWriter{ResponseWriter: w}
		c := cache.New()
		if res, _ := c.Match(r, nil); res != nil {
			// cache HIT
			for key, values := range res.Header {
				for _, value := range values {
					rw.Header().Add(key, value)
				}
			}
			rw.WriteHeader(res.StatusCode)
			io.Copy(rw.ResponseWriter, res.Body)
			next.ServeHTTP(rw, r)
			return
		}
		// cache MISS
		next.ServeHTTP(rw, r)
		// populate Cloudflare cache. Errors are cacheable too because the result
		// for a given set of inputs will never change
		cloudflare.WaitUntil(ctx, func() {
			err := c.Put(r, rw.ToHTTPResponse())
			if err != nil {
				log.Printf("Cloudflare cache put: %v", err)
			}
		})
	})
}

func finalSize(u *url.URL, defwidth, defheight int) (width, height int, err error) {
	optwidth := u.Query().Get("width")
	optheight := u.Query().Get("height")
	width = defwidth
	height = defheight
	if optwidth != "" {
		if width, err = strconv.Atoi(optwidth); err != nil {
			return
		}
	}
	if optheight != "" {
		if height, err = strconv.Atoi(optheight); err != nil {
			return
		}
	}
	if width < 50 || height < 50 {
		err = fmt.Errorf("%dx%d is an improbable size. nope", width, height)
	}
	return
}

type barcodeServeMux struct {
	http.ServeMux
}

func newbarcodeServeMux() *barcodeServeMux {
	return &barcodeServeMux{}
}

func (b *barcodeServeMux) register(name string, width, height int, renderer renderFunc) {
	prefix := "/" + name + "/"
	b.HandleFunc(prefix, func(w http.ResponseWriter, r *http.Request) {
		// same inputs = same outputs, including errors
		w.Header().Set("cache-control", "public, max-age=604800")
		code := strings.TrimPrefix(r.URL.EscapedPath(), prefix)
		finalwidth, finalheight, err := finalSize(r.URL, width, height)
		if err != nil {
			log.Printf("size error: %v", err)
			w.WriteHeader(http.StatusBadRequest)
			fmt.Fprintf(w, "invalid size")
			return
		}
		bc, err := renderer(code)
		if err != nil {
			log.Printf("render error: %v", err)
			w.WriteHeader(http.StatusInternalServerError)
			fmt.Fprintf(w, "render error")
			return
		}
		bc, err = barcode.Scale(bc, finalwidth, finalheight)
		if err != nil {
			log.Printf("scale error: %v", err)
			w.WriteHeader(http.StatusInternalServerError)
			fmt.Fprintf(w, "scale error")
			return
		}
		w.Header().Set("content-type", "image/png")
		png.Encode(w, bc)
	})
}

func main() {
	mux := newbarcodeServeMux()
	mux.register("qr", 200, 200, renderQR)
	mux.register("c128", 165, 50, renderCode128)
	mux.register("c39", 165, 50, renderCode39)
	mux.register("aztec", 200, 200, renderAztec)
	mux.register("2of5", 165, 50, render2of5)
	mux.register("2of5i", 165, 50, render2of5interleaved)
	workers.Serve(caching(ignorecasefirstpath(mux)))
}
