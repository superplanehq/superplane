package grpc

import (
	"context"
	"net/http"
	"strings"

	"github.com/google/uuid"
	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	log "github.com/sirupsen/logrus"
	"github.com/superplanehq/superplane/pkg/logging"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
)

// RecordGatewayRequest names the API operation without recording its body.
func RecordGatewayRequest(ctx context.Context) {
	fields := log.Fields{}
	if method, ok := runtime.RPCMethod(ctx); ok {
		fields["action"] = method
	}
	if pattern, ok := runtime.HTTPPathPattern(ctx); ok {
		fields["object"] = pattern
	}
	logging.AddRequestLogFields(ctx, fields)
}

// AuditGatewayResponse records identifiers returned by successful operations,
// including creates whose object IDs are not present in the request URL.
func AuditGatewayResponse(ctx context.Context, _ http.ResponseWriter, response proto.Message) error {
	RecordGatewayRequest(ctx)
	if response == nil || logging.RequestLogFields(ctx) == nil {
		return nil
	}

	ids := map[string]string{}
	message := response.ProtoReflect()
	appendAuditIdentifiers(ids, "", message)
	message.Range(func(field protoreflect.FieldDescriptor, value protoreflect.Value) bool {
		if field.IsList() || field.IsMap() || field.Kind() != protoreflect.MessageKind {
			return true
		}
		resource := value.Message()
		prefix := string(field.Name()) + "."
		appendAuditIdentifiers(ids, prefix, resource)
		metadata := resource.Descriptor().Fields().ByName("metadata")
		if metadata == nil || metadata.IsList() || metadata.IsMap() || metadata.Kind() != protoreflect.MessageKind || !resource.Has(metadata) {
			return true
		}
		meta := resource.Get(metadata).Message()
		appendAuditIdentifiers(ids, prefix, meta)
		if meta.Descriptor().Fields().ByName("id") != nil {
			return true
		}
		// Roles and groups use metadata.name as their identifier.
		name := meta.Descriptor().Fields().ByName("name")
		if name != nil && !name.IsList() && !name.IsMap() && name.Kind() == protoreflect.StringKind && meta.Has(name) {
			ids[prefix+"name"] = meta.Get(name).String()
		}
		return true
	})
	if len(ids) > 0 {
		logging.AddRequestLogFields(ctx, log.Fields{"object_ids": ids})
	}
	return nil
}

func appendAuditIdentifiers(ids map[string]string, prefix string, message protoreflect.Message) {
	message.Range(func(field protoreflect.FieldDescriptor, value protoreflect.Value) bool {
		name := string(field.Name())
		if field.IsList() || field.IsMap() || field.Kind() != protoreflect.StringKind || (name != "id" && !strings.HasSuffix(name, "_id")) {
			return true
		}
		// Accept UUIDs only: arbitrary fields named id can also contain tokens.
		if id, err := uuid.Parse(value.String()); err == nil && id != uuid.Nil {
			ids[prefix+name] = id.String()
		}
		return true
	})
}
