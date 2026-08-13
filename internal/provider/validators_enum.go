package provider

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
)

func oneOfStrings(name string, allowed ...string) validator.String {
	set := make(map[string]struct{}, len(allowed))
	for _, a := range allowed {
		set[strings.ToLower(a)] = struct{}{}
	}
	return stringOneOfValidator{name: name, allowed: allowed, set: set}
}

type stringOneOfValidator struct {
	name    string
	allowed []string
	set     map[string]struct{}
}

func (v stringOneOfValidator) Description(_ context.Context) string {
	return fmt.Sprintf("%s must be one of: %s", v.name, strings.Join(v.allowed, ", "))
}

func (v stringOneOfValidator) MarkdownDescription(ctx context.Context) string {
	return v.Description(ctx)
}

func (v stringOneOfValidator) ValidateString(_ context.Context, req validator.StringRequest, resp *validator.StringResponse) {
	if req.ConfigValue.IsNull() || req.ConfigValue.IsUnknown() {
		return
	}
	s := strings.ToLower(strings.TrimSpace(req.ConfigValue.ValueString()))
	if _, ok := v.set[s]; ok {
		return
	}
	resp.Diagnostics.AddAttributeError(
		req.Path,
		fmt.Sprintf("Invalid %s", v.name),
		fmt.Sprintf("%s must be one of: %s; got %q", v.name, strings.Join(v.allowed, ", "), req.ConfigValue.ValueString()),
	)
}

func oneOfInt64(name string, allowed ...int64) validator.Int64 {
	set := make(map[int64]struct{}, len(allowed))
	for _, a := range allowed {
		set[a] = struct{}{}
	}
	return int64OneOfValidator{name: name, allowed: allowed, set: set}
}

type int64OneOfValidator struct {
	name    string
	allowed []int64
	set     map[int64]struct{}
}

func (v int64OneOfValidator) Description(_ context.Context) string {
	parts := make([]string, len(v.allowed))
	for i, a := range v.allowed {
		parts[i] = strconv.FormatInt(a, 10)
	}
	return fmt.Sprintf("%s must be one of: %s", v.name, strings.Join(parts, ", "))
}

func (v int64OneOfValidator) MarkdownDescription(ctx context.Context) string {
	return v.Description(ctx)
}

func (v int64OneOfValidator) ValidateInt64(_ context.Context, req validator.Int64Request, resp *validator.Int64Response) {
	if req.ConfigValue.IsNull() || req.ConfigValue.IsUnknown() {
		return
	}
	n := req.ConfigValue.ValueInt64()
	if _, ok := v.set[n]; ok {
		return
	}
	parts := make([]string, len(v.allowed))
	for i, a := range v.allowed {
		parts[i] = strconv.FormatInt(a, 10)
	}
	resp.Diagnostics.AddAttributeError(
		req.Path,
		fmt.Sprintf("Invalid %s", v.name),
		fmt.Sprintf("%s must be one of: %s; got %d", v.name, strings.Join(parts, ", "), n),
	)
}
