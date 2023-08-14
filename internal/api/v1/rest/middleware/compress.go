// Package middleware provides various middleware functionality.
package middleware

import (
	"compress/gzip"
	"io"
	"net/http"
	"strings"
)

// Type gzipWriter redefines http.ResponseWriter changing its Writer method to use gzip.
type gzipWriter struct {
	http.ResponseWriter
	Writer io.Writer
}

// Write method redefines default http.ResponseWriter Write method.
func (w gzipWriter) Write(b []byte) (int, error) {
	return w.Writer.Write(b)
}

// CompressHandle serves as a middleware handler implementing gzip compressing.
func CompressHandle(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.Contains(r.Header.Get("Accept-Encoding"), "gzip") {
			next.ServeHTTP(w, r)
			return
		}
		gz, _ := gzip.NewWriterLevel(w, gzip.BestCompression)
		defer gz.Close()
		w.Header().Set("Content-Encoding", "gzip")
		next.ServeHTTP(gzipWriter{ResponseWriter: w, Writer: gz}, r)
	})
}

// DecompressHandle serves as a middleware handler implementing gzip decompressing.
func DecompressHandle(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.Contains(r.Header.Get("Content-Encoding"), "gzip") {
			next.ServeHTTP(w, r)
			return
		}
		gz, _ := gzip.NewReader(r.Body)
		r.Body = gz
		next.ServeHTTP(w, r)
	})
}
