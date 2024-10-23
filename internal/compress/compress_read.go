// отвечает за чтение сжатых данных
package compress

import (
	"compress/gzip"
	"io"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

type compressReader struct {
	r  io.ReadCloser
	zr *gzip.Reader
}

func newCompressReader(r io.ReadCloser) (*compressReader, error) {
	zr, err := gzip.NewReader(r)
	if err != nil {
		return nil, err
	}

	return &compressReader{
		r:  r,
		zr: zr,
	}, nil
}

func (c compressReader) Read(p []byte) (n int, err error) {
	return c.zr.Read(p)
}

func (c *compressReader) Close() error {
	if err := c.r.Close(); err != nil {
		return err
	}
	return c.zr.Close()
}
func DeCompressData() gin.HandlerFunc {
	return func(c *gin.Context) {
		contentEncoding := c.GetHeader("Content-Encoding")
		if !strings.Contains(contentEncoding, "gzip") {
			c.Next()
			return
		}

		gzr, err := newCompressReader(c.Request.Body)
		if err != nil {
			c.String(http.StatusBadRequest, "Invalid gzip data")
			c.Abort()
			return
		}
		defer gzr.Close()
		c.Request.Body = gzr
		c.Next()
	}
}
