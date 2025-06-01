package logging

import (
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

func New() *zap.SugaredLogger {
	logger, err := zap.NewDevelopment()
	if err != nil {
		panic(err)
	}
	sugar := logger.Sugar()
	return sugar
}
func LoggingPost(logger *zap.SugaredLogger) gin.HandlerFunc {
	return func(c *gin.Context) {
		startTime := time.Now()
		uri := c.Request.RequestURI
		method := c.Request.Method
		c.Next()

		duration := time.Since(startTime)
		logger.Infow(
			"Process",
			"uri", uri,
			"method", method,
			"duration", duration,
		)
	}
}

type CustomResponseWriter struct {
	gin.ResponseWriter
	size int
}

func (w *CustomResponseWriter) Write(b []byte) (int, error) {
	size, err := w.ResponseWriter.Write(b)
	w.size += size
	return size, err
}

func LoggingGet(logger *zap.SugaredLogger) gin.HandlerFunc {
	return func(c *gin.Context) {
		crw := &CustomResponseWriter{
			ResponseWriter: c.Writer,
		}
		c.Writer = crw
		c.Next()
		statusCode := c.Writer.Status()
		responseSize := crw.size
		logger.Infow(
			"Starting process",
			"status", statusCode,
			"response_size", responseSize,
		)
	}
}
