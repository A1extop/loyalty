// отвечает за сжатие данных
package compress

import (
	"bufio"
	"compress/gzip"
	"errors"
	"net"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

type compressWriter struct {
	w  gin.ResponseWriter
	zw *gzip.Writer
}

func newCompressWriter(w gin.ResponseWriter) *compressWriter {
	return &compressWriter{
		w:  w,
		zw: gzip.NewWriter(w),
	}
}

func (c compressWriter) Header() http.Header {
	return c.w.Header()
}

func (c compressWriter) Write(b []byte) (int, error) {
	return c.zw.Write(b)
}

func (c compressWriter) WriteHeader(statusCode int) {
	c.w.Header().Del("Content-Length")
	c.w.WriteHeader(statusCode)
}

func (c compressWriter) Flush() {
	if flusher, ok := c.w.(http.Flusher); ok {
		flusher.Flush()
	}
}

func (c compressWriter) CloseNotify() <-chan bool {
	notify := make(chan bool)

	close(notify)

	return notify
}

func (c compressWriter) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	if hijacker, ok := c.w.(http.Hijacker); ok {
		return hijacker.Hijack()
	}
	return nil, nil, errors.New("hijacker not supported")
}

func (c compressWriter) WriteString(s string) (int, error) {
	return c.Write([]byte(s))
}

func (c compressWriter) Status() int {
	return c.w.Status()
}

func (c compressWriter) Size() int {
	return c.w.Size()
}

func (c compressWriter) Written() bool {
	return c.w.Written()
}

func (c compressWriter) WriteHeaderNow() {
	c.w.WriteHeaderNow()
}

func (c compressWriter) Pusher() http.Pusher {
	return c.w.Pusher()
}

func CompressData() gin.HandlerFunc {
	return func(c *gin.Context) {
		contentEncoding := c.GetHeader("Accept-Encoding")
		if !strings.Contains(contentEncoding, "gzip") {
			c.Next()
			return
		}

		c.Header("Content-Encoding", "gzip")

		gzw := newCompressWriter(c.Writer)
		c.Writer = gzw

		c.Next()

		gzw.zw.Close()
	}
}
