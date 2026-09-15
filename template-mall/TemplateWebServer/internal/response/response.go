package response

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func OK(c *gin.Context, data any) {
	c.JSON(http.StatusOK, gin.H{"data": data})
}

func Fail(c *gin.Context, httpCode int, msg string) {
	c.JSON(httpCode, gin.H{"error": msg})
}

func FromGRPC(c *gin.Context, err error) {
	st, ok := status.FromError(err)
	if !ok {
		Fail(c, http.StatusBadGateway, err.Error())
		return
	}
	switch st.Code() {
	case codes.InvalidArgument, codes.FailedPrecondition:
		Fail(c, http.StatusBadRequest, st.Message())
	case codes.NotFound:
		Fail(c, http.StatusNotFound, st.Message())
	case codes.PermissionDenied:
		Fail(c, http.StatusForbidden, st.Message())
	case codes.Unauthenticated:
		Fail(c, http.StatusUnauthorized, st.Message())
	default:
		Fail(c, http.StatusBadGateway, st.Message())
	}
}
