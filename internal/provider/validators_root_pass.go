package provider

import (
	"context"
	"fmt"
	"regexp"
	"unicode"

	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
)

var rootPassSpecial = regexp.MustCompile(`[%\-_+]`)

type rootPassValidator struct{}

func rootPassRules() validator.String {
	return rootPassValidator{}
}

func (v rootPassValidator) Description(_ context.Context) string {
	return "password must be 8-30 chars with upper, lower, digit, and one of % - _ +"
}

func (v rootPassValidator) MarkdownDescription(ctx context.Context) string {
	return v.Description(ctx)
}

func (v rootPassValidator) ValidateString(_ context.Context, req validator.StringRequest, resp *validator.StringResponse) {
	if req.ConfigValue.IsNull() || req.ConfigValue.IsUnknown() {
		return
	}
	pass := req.ConfigValue.ValueString()
	if err := validateRootPass(pass); err != nil {
		resp.Diagnostics.AddAttributeError(req.Path, "Invalid root_pass", err.Error())
	}
}

func validateRootPass(pass string) error {
	if n := len(pass); n < 8 || n > 30 {
		return fmt.Errorf("must be 8-30 characters (got %d)", n)
	}
	if pass[0] == '%' || pass[0] == '-' || pass[0] == '_' || pass[0] == '+' {
		return fmt.Errorf("must not start with a special character")
	}
	var upper, lower, digit bool
	for _, r := range pass {
		switch {
		case unicode.IsUpper(r):
			upper = true
		case unicode.IsLower(r):
			lower = true
		case unicode.IsDigit(r):
			digit = true
		}
	}
	if !upper || !lower || !digit || !rootPassSpecial.MatchString(pass) {
		return fmt.Errorf("must contain an upper case letter, a lower case letter, a digit, and one of: %% - _ +")
	}
	for _, bad := range []string{"@", "#"} {
		for i := 0; i < len(pass); i++ {
			if string(pass[i]) == bad {
				return fmt.Errorf("must not contain %q", bad)
			}
		}
	}
	return nil
}
