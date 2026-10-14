package interceptor

import (
	"context"
	"log/slog"
	"path"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/status"
)

func LoggerInterceptor() grpc.UnaryServerInterceptor {
	return func(
		ctx context.Context,
		req interface{},
		info *grpc.UnaryServerInfo,
		handler grpc.UnaryHandler) (interface{}, error) {
		method := path.Base(info.FullMethod)

		slog.InfoContext(ctx, "grpc request", "method", method)

		start := time.Now()

		resp, err := handler(ctx, req)

		duration := time.Since(start)

		if err != nil {
			st, _ := status.FromError(err)
			slog.ErrorContext(ctx, "grpc finished with error", "method", method, "code", st.Code(), "error", err, "duration", duration)
		} else {
			slog.InfoContext(ctx, "grpc finished successfully", "method", method, "duration", duration)
		}
		return resp, err
	}
}
