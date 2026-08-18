package grpc

import (
	"net/http"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/encoding/protojson"
)

func writeGatewayHTTPError(r *http.Request, w http.ResponseWriter, err error) {
	sanitized := SanitizeError(r.Context(), err)
	reportGatewayError(r, err, sanitized)
	s := status.Convert(sanitized)

	marshaler := &runtime.JSONPb{
		MarshalOptions: protojson.MarshalOptions{
			EmitUnpopulated: true,
		},
	}

	body, marshalErr := marshaler.Marshal(s.Proto())
	if marshalErr != nil {
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", marshaler.ContentType(s.Proto()))
	w.WriteHeader(runtime.HTTPStatusFromCode(s.Code()))
	_, _ = w.Write(body)
}
