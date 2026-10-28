// Command mdedit-web serves the WebAssembly build of mdedit (make wasm): a
// small static file server that sends the right content types and the
// pre-compressed copies (.br, .gz) of the large files.
//
//	mdedit-web -dir bin/web -addr :8089
//	mdedit-web -dir /usr/local/www/content_renderer -tls-cert c.pem -tls-key k.pem
//
// Browsers only allow the clipboard on HTTPS pages (and on localhost), so
// serve it over HTTPS, here or behind a proxy, for copy and paste to work.
package main

import (
	"flag"
	"log"
	"mime"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"
)

var (
	addrFlag = flag.String("addr", ":8089", "listen `address`")
	dirFlag  = flag.String("dir", ".", "`folder` with index.html, mdedit.wasm and wasm_exec.js")
	certFlag = flag.String("tls-cert", "", "TLS certificate `file` (PEM); serves HTTPS with -tls-key")
	keyFlag  = flag.String("tls-key", "", "TLS private key `file` (PEM)")
)

func main() {
	flag.Parse()
	if _, err := os.Stat(filepath.Join(*dirFlag, "index.html")); err != nil {
		log.Fatalf("mdedit-web: no index.html in %s (run make wasm)", *dirFlag)
	}
	srv := &http.Server{
		Addr:              *addrFlag,
		Handler:           files{root: *dirFlag},
		ReadHeaderTimeout: 10 * time.Second,
	}
	log.Printf("mdedit-web: serving %s on %s", *dirFlag, *addrFlag)
	var err error
	if *certFlag != "" {
		err = srv.ListenAndServeTLS(*certFlag, *keyFlag)
	} else {
		err = srv.ListenAndServe()
	}
	log.Fatal("mdedit-web: ", err)
}

// files serves the folder, preferring a pre-compressed copy of each file.
type files struct{ root string }

// encodings are the compressed copies make wasm writes, best first.
var encodings = []struct{ name, ext string }{{"br", ".br"}, {"gzip", ".gz"}}

func (f files) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	name := path.Clean("/" + r.URL.Path)
	if name == "/" {
		name = "/index.html"
	}
	file, encoding := f.pick(name, r.Header.Get("Accept-Encoding"))
	if file == "" {
		http.NotFound(w, r)
		return
	}
	f.headers(w, name, encoding)
	http.ServeFile(w, r, file)
}

// pick returns the file to send for name and its content encoding: a
// compressed copy the browser accepts, or else the file itself.
func (f files) pick(name, accept string) (string, string) {
	plain := filepath.Join(f.root, filepath.FromSlash(name))
	for _, e := range encodings {
		if strings.Contains(accept, e.name) && isFile(plain+e.ext) {
			return plain + e.ext, e.name
		}
	}
	if isFile(plain) {
		return plain, ""
	}
	return "", ""
}

func (f files) headers(w http.ResponseWriter, name, encoding string) {
	h := w.Header()
	if t := mime.TypeByExtension(path.Ext(name)); t != "" {
		h.Set("Content-Type", t) // of the original, not of the .gz
	}
	if encoding != "" {
		h.Set("Content-Encoding", encoding)
	}
	h.Set("Vary", "Accept-Encoding")
	h.Set("Cache-Control", "no-cache") // revalidate, so a new build is picked up
	h.Set("X-Content-Type-Options", "nosniff")
}

func isFile(p string) bool {
	st, err := os.Stat(p)
	return err == nil && st.Mode().IsRegular()
}
