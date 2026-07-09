package authz

import (
	"context"
	_ "embed"
	"fmt"

	"github.com/open-policy-agent/opa/v1/rego"
)

//go:embed publication.rego
var publicationSource string

// Publication is the compiled publication policy: the allowlist of fields
// each resource may expose on the anonymous public tier.
type Publication struct {
	fields     map[string]map[string]struct{}
	lineFields map[string]struct{}
}

// NewPublication compiles the policy and materializes the field sets once at
// startup — the public handlers consult plain sets on the hot path.
func NewPublication(ctx context.Context) (*Publication, error) {
	query, err := rego.New(
		rego.Query("data.opentreasury.publication"),
		rego.Module("publication.rego", publicationSource),
	).PrepareForEval(ctx)
	if err != nil {
		return nil, fmt.Errorf("compiling publication policy: %w", err)
	}

	results, err := query.Eval(ctx)
	if err != nil || len(results) == 0 || len(results[0].Expressions) == 0 {
		return nil, fmt.Errorf("evaluating publication policy: %w", err)
	}

	document, ok := results[0].Expressions[0].Value.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("publication policy has unexpected shape")
	}

	publication := &Publication{fields: map[string]map[string]struct{}{}, lineFields: map[string]struct{}{}}

	rawFields, ok := document["fields"].(map[string]any)
	if !ok {
		return nil, fmt.Errorf("publication policy missing fields map")
	}
	for resource, raw := range rawFields {
		names, ok := raw.([]any)
		if !ok {
			return nil, fmt.Errorf("publication fields for %s have unexpected shape", resource)
		}
		set := map[string]struct{}{}
		for _, name := range names {
			set[fmt.Sprint(name)] = struct{}{}
		}
		publication.fields[resource] = set
	}

	rawLineFields, ok := document["line_fields"].([]any)
	if !ok {
		return nil, fmt.Errorf("publication policy missing line_fields")
	}
	for _, name := range rawLineFields {
		publication.lineFields[fmt.Sprint(name)] = struct{}{}
	}

	return publication, nil
}

// Redact returns a copy of row containing only fields the policy publishes
// for the resource. Nested journal lines are masked with the line allowlist.
func (p *Publication) Redact(resource string, row map[string]any) map[string]any {
	allowed, ok := p.fields[resource]
	if !ok {
		return map[string]any{} // unknown resource publishes nothing
	}

	masked := make(map[string]any, len(allowed))
	for field, value := range row {
		if _, ok := allowed[field]; !ok {
			continue
		}
		if field == "lines" {
			masked[field] = p.redactLines(value)
			continue
		}
		masked[field] = value
	}

	return masked
}

func (p *Publication) redactLines(value any) any {
	lines, ok := value.([]map[string]any)
	if !ok {
		return []map[string]any{}
	}

	maskedLines := make([]map[string]any, 0, len(lines))
	for _, line := range lines {
		maskedLine := map[string]any{}
		for field, fieldValue := range line {
			if _, ok := p.lineFields[field]; ok {
				maskedLine[field] = fieldValue
			}
		}
		maskedLines = append(maskedLines, maskedLine)
	}

	return maskedLines
}

// PublishesField reports whether the policy publishes a field — used by tests
// to assert classification decisions.
func (p *Publication) PublishesField(resource, field string) bool {
	_, ok := p.fields[resource][field]
	return ok
}
