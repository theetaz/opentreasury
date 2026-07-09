package treasury

import "context"

// AuditMetadata identifies who caused a domain change and through which
// request. It travels on the context so every write path can stamp its
// audit event without widening repository signatures.
type AuditMetadata struct {
	Actor     string
	RequestID string
}

type auditMetadataKey struct{}

func ContextWithAuditMetadata(ctx context.Context, metadata AuditMetadata) context.Context {
	return context.WithValue(ctx, auditMetadataKey{}, metadata)
}

func auditMetadataFromContext(ctx context.Context) AuditMetadata {
	metadata, ok := ctx.Value(auditMetadataKey{}).(AuditMetadata)
	if !ok || metadata.Actor == "" {
		metadata.Actor = "system"
	}
	return metadata
}
