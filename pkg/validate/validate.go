// Package validate provides Laravel-inspired struct-tag validation for Kashvi.
//
// Supported rules (comma-separated in the `validate` tag):
//
//	required            field must not be zero/empty
//	nullable            if empty, skip all remaining rules for this field
//	email               valid email address
//	url                 valid URL (http/https)
//	uuid                valid UUID (v4)
//	ip                  valid IPv4 or IPv6 address
//	json                valid JSON string
//	boolean             "true","false","1","0" (or actual bool)
//	date                parseable date (many common layouts tried)
//	alpha               letters only
//	alpha_num           letters and digits only
//	alpha_dash          letters, digits, hyphens, underscores
//	numeric             any number
//	integer             whole number
//	min=N               string: min char length | number: min value
//	max=N               string: max char length | number: max value
//	size=N              string: exact length
//	gt=N                number > N
//	gte=N               number >= N
//	lt=N                number < N
//	lte=N               number <= N
//	between=min,max     number or string length between min and max (inclusive)
//	digits=N            exactly N decimal digits
//	in=a,b,c            value must be one of the listed items
//	not_in=a,b,c        value must NOT be one of the listed items
//	regex=pattern       value must match the regex (avoid commas in pattern)
//	confirmed           value must equal a sibling field named <field>_confirmation
//	before=date         value (as date) must be before given date
//	after=date          value (as date) must be after given date
//
// Example:
//
//	type Input struct {
//	    Name  string  `json:"name"  validate:"required,alpha_dash,min=2,max=100"`
//	    Email string  `json:"email" validate:"required,email"`
//	    Age   int     `json:"age"   validate:"required,gte=18,lte=120"`
//	    Role  string  `json:"role"  validate:"required,in=admin,user,moderator"`
//	    Site  string  `json:"site"  validate:"nullable,url"`
//	}
package validate

import (
	"encoding/json"
	"fmt"
	"net"
	"net/url"
	"reflect"
	"regexp"
	"strconv"
	"strings"
	"time"
	"unicode"
)

// ─── Public API ───────────────────────────────────────────────────────────────

// Struct validates all exported fields of v that carry a `validate` tag.
// Returns a map of fieldName → error message; empty map means no errors.
func Struct(v interface{}) map[string]string {
	errs := make(map[string]string)
	rv := reflect.ValueOf(v)
	if rv.Kind() == reflect.Ptr {
		rv = rv.Elem()
	}
	if rv.Kind() != reflect.Struct {
		return errs
	}
	rt := rv.Type()

	for i := 0; i < rt.NumField(); i++ {
		field := rt.Field(i)
		value := rv.Field(i)

		tag := field.Tag.Get("validate")
		if tag == "" {
			continue
		}

		name := jsonFieldName(field)
		rules := splitRules(tag)

		// If `nullable` is present and field is empty — skip all rules.
		if hasRule(rules, "nullable") && isEmpty(value) {
			continue
		}

		for _, rule := range rules {
			if rule == "nullable" {
				continue
			}
			if msg := applyRule(rule, name, value, rv); msg != "" {
				errs[name] = msg
				break // first failing rule per field
			}
		}
	}

	return errs
}

// HasErrors returns true when the errs map is non-empty.
func HasErrors(errs map[string]string) bool { return len(errs) > 0 }

// ─── Core dispatcher ──────────────────────────────────────────────────────────

func applyRule(rule, field string, v reflect.Value, parent reflect.Value) string {
	raw := fmt.Sprintf("%v", v.Interface())
	key, param, _ := strings.Cut(rule, "=")

	switch key {
	// ── Presence ──────────────────────────────────────────────────────
	case "required":
		if isEmpty(v) {
			return fmt.Sprintf("The %s field is required.", field)
		}

	// ── Format ────────────────────────────────────────────────────────
	case "email":
		if !emailRE.MatchString(raw) {
			return fmt.Sprintf("The %s must be a valid email address.", field)
		}
	case "url":
		u, err := url.ParseRequestURI(raw)
		if err != nil || (u.Scheme != "http" && u.Scheme != "https") {
			return fmt.Sprintf("The %s must be a valid URL.", field)
		}
	case "uuid":
		if !uuidRE.MatchString(raw) {
			return fmt.Sprintf("The %s must be a valid UUID.", field)
		}
	case "ip":
		if net.ParseIP(raw) == nil {
			return fmt.Sprintf("The %s must be a valid IP address.", field)
		}
	case "json":
		if !json.Valid([]byte(raw)) {
			return fmt.Sprintf("The %s must be a valid JSON string.", field)
		}
	case "boolean":
		lower := strings.ToLower(raw)
		if v.Kind() != reflect.Bool && lower != "true" && lower != "false" && lower != "1" && lower != "0" {
			return fmt.Sprintf("The %s field must be true or false.", field)
		}
	case "date":
		if _, err := parseDate(raw); err != nil {
			return fmt.Sprintf("The %s is not a valid date.", field)
		}

	// ── Character class ───────────────────────────────────────────────
	case "alpha":
		for _, c := range raw {
			if !unicode.IsLetter(c) {
				return fmt.Sprintf("The %s field must contain only letters.", field)
			}
		}
	case "alpha_num":
		for _, c := range raw {
			if !unicode.IsLetter(c) && !unicode.IsDigit(c) {
				return fmt.Sprintf("The %s field must contain only letters and numbers.", field)
			}
		}
	case "alpha_dash":
		for _, c := range raw {
			if !unicode.IsLetter(c) && !unicode.IsDigit(c) && c != '-' && c != '_' {
				return fmt.Sprintf("The %s field may only contain letters, numbers, dashes, and underscores.", field)
			}
		}
	case "numeric":
		if _, err := strconv.ParseFloat(raw, 64); err != nil {
			return fmt.Sprintf("The %s field must be a number.", field)
		}
	case "integer":
		if _, err := strconv.ParseInt(raw, 10, 64); err != nil {
			return fmt.Sprintf("The %s field must be an integer.", field)
		}

	// ── Size / range ──────────────────────────────────────────────────
	case "min":
		n := mustParseFloat(param)
		if isNumericKind(v) {
			if toFloat(v) < n {
				return fmt.Sprintf("The %s must be at least %s.", field, param)
			}
		} else {
			if float64(len([]rune(raw))) < n {
				return fmt.Sprintf("The %s must be at least %s characters.", field, param)
			}
		}
	case "max":
		n := mustParseFloat(param)
		if isNumericKind(v) {
			if toFloat(v) > n {
				return fmt.Sprintf("The %s must not be greater than %s.", field, param)
			}
		} else {
			if float64(len([]rune(raw))) > n {
				return fmt.Sprintf("The %s must not exceed %s characters.", field, param)
			}
		}
	case "size":
		n := mustParseFloat(param)
		if float64(len([]rune(raw))) != n {
			return fmt.Sprintf("The %s must be exactly %s characters.", field, param)
		}
	case "gt":
		n := mustParseFloat(param)
		if toFloat(v) <= n {
			return fmt.Sprintf("The %s must be greater than %s.", field, param)
		}
	case "gte":
		n := mustParseFloat(param)
		if toFloat(v) < n {
			return fmt.Sprintf("The %s must be greater than or equal to %s.", field, param)
		}
	case "lt":
		n := mustParseFloat(param)
		if toFloat(v) >= n {
			return fmt.Sprintf("The %s must be less than %s.", field, param)
		}
	case "lte":
		n := mustParseFloat(param)
		if toFloat(v) > n {
			return fmt.Sprintf("The %s must be less than or equal to %s.", field, param)
		}
	case "between":
		parts := strings.SplitN(param, ",", 2)
		if len(parts) == 2 {
			lo, hi := mustParseFloat(parts[0]), mustParseFloat(parts[1])
			if isNumericKind(v) {
				f := toFloat(v)
				if f < lo || f > hi {
					return fmt.Sprintf("The %s must be between %s and %s.", field, parts[0], parts[1])
				}
			} else {
				l := float64(len([]rune(raw)))
				if l < lo || l > hi {
					return fmt.Sprintf("The %s must be between %s and %s characters.", field, parts[0], parts[1])
				}
			}
		}
	case "digits":
		n := mustParseFloat(param)
		if !digitsOnlyRE.MatchString(raw) || float64(len(raw)) != n {
			return fmt.Sprintf("The %s must be %s digits.", field, param)
		}

	// ── Inclusion / exclusion ─────────────────────────────────────────
	case "in":
		allowed := strings.Split(param, ",")
		for _, a := range allowed {
			if raw == strings.TrimSpace(a) {
				return ""
			}
		}
		return fmt.Sprintf("The selected %s is invalid.", field)
	case "not_in":
		forbidden := strings.Split(param, ",")
		for _, f := range forbidden {
			if raw == strings.TrimSpace(f) {
				return fmt.Sprintf("The selected %s is invalid.", field)
			}
		}

	// ── Pattern ───────────────────────────────────────────────────────
	case "regex":
		re, err := regexp.Compile(param)
		if err != nil {
			return fmt.Sprintf("The %s has an invalid validation pattern.", field)
		}
		if !re.MatchString(raw) {
			return fmt.Sprintf("The %s format is invalid.", field)
		}

	// ── Cross-field ───────────────────────────────────────────────────
	case "confirmed":
		// Looks for a sibling field whose json tag is <field>_confirmation.
		confirmVal := findSiblingByJSONSuffix(parent, field, "_confirmation")
		if confirmVal == nil || fmt.Sprintf("%v", confirmVal.Interface()) != raw {
			return fmt.Sprintf("The %s confirmation does not match.", field)
		}

	// ── Date comparison ───────────────────────────────────────────────
	case "before":
		t1, err1 := parseDate(raw)
		t2, err2 := parseDate(param)
		if err1 != nil || err2 != nil || !t1.Before(t2) {
			return fmt.Sprintf("The %s must be a date before %s.", field, param)
		}
	case "after":
		t1, err1 := parseDate(raw)
		t2, err2 := parseDate(param)
		if err1 != nil || err2 != nil || !t1.After(t2) {
			return fmt.Sprintf("The %s must be a date after %s.", field, param)
		}
	}

	return ""
}

// ─── Helpers ─────────────────────────────────────────────────────────────────

var (
	emailRE      = regexp.MustCompile(`^[a-zA-Z0-9._%+\-]+@[a-zA-Z0-9.\-]+\.[a-zA-Z]{2,}$`)
	uuidRE       = regexp.MustCompile(`(?i)^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$`)
	digitsOnlyRE = regexp.MustCompile(`^\d+$`)
)

var dateLayouts = []string{
	time.RFC3339, "2006-01-02", "02/01/2006", "01/02/2006",
	"2006-01-02 15:04:05", "January 2, 2006", "Jan 2, 2006",
}

func parseDate(s string) (time.Time, error) {
	for _, layout := range dateLayouts {
		if t, err := time.Parse(layout, s); err == nil {
			return t, nil
		}
	}
	return time.Time{}, fmt.Errorf("cannot parse %q as date", s)
}

func isEmpty(v reflect.Value) bool {
	switch v.Kind() {
	case reflect.String:
		return strings.TrimSpace(v.String()) == ""
	case reflect.Slice, reflect.Map, reflect.Array:
		return v.Len() == 0
	case reflect.Ptr, reflect.Interface:
		return v.IsNil()
	case reflect.Bool:
		return false // false is a valid boolean value, not empty
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return v.Int() == 0
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return v.Uint() == 0
	case reflect.Float32, reflect.Float64:
		return v.Float() == 0
	}
	return false
}

func isNumericKind(v reflect.Value) bool {
	switch v.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64,
		reflect.Float32, reflect.Float64:
		return true
	}
	return false
}

func toFloat(v reflect.Value) float64 {
	switch v.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return float64(v.Int())
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return float64(v.Uint())
	case reflect.Float32, reflect.Float64:
		return v.Float()
	}
	f, _ := strconv.ParseFloat(fmt.Sprintf("%v", v.Interface()), 64)
	return f
}

func mustParseFloat(s string) float64 {
	f, _ := strconv.ParseFloat(strings.TrimSpace(s), 64)
	return f
}

func jsonFieldName(f reflect.StructField) string {
	name := f.Tag.Get("json")
	if name == "" || name == "-" {
		return strings.ToLower(f.Name)
	}
	if idx := strings.Index(name, ","); idx != -1 {
		name = name[:idx]
	}
	return name
}

// splitRules splits the validate tag by comma while keeping multi-value
// rule parameters (in=, not_in=, between=) intact.
// e.g. "required,in=admin,user,mod,max=100" → ["required","in=admin,user,mod","max=100"]
func splitRules(tag string) []string {
	var rules []string
	var current strings.Builder
	inParam := false // true when we are inside a multi-value param (in=, not_in=, between=)

	multiValuePrefixes := []string{"in=", "not_in=", "between="}

	for i := 0; i < len(tag); i++ {
		ch := tag[i]
		if ch == ',' {
			if inParam {
				// Check whether the next token starts a new rule keyword.
				// A new rule either has no '=' or has '=' after the first word.
				rest := tag[i+1:]
				if looksLikeNewRule(rest) {
					// This comma ends the current param rule.
					rules = append(rules, current.String())
					current.Reset()
					inParam = false
				} else {
					// Comma is part of the param value (e.g. in=a,b,c).
					current.WriteByte(ch)
				}
			} else {
				rules = append(rules, current.String())
				current.Reset()
			}
		} else {
			current.WriteByte(ch)
			// Check if we just completed a multi-value prefix.
			if !inParam {
				for _, pfx := range multiValuePrefixes {
					if strings.HasSuffix(current.String(), pfx) {
						inParam = true
						break
					}
				}
			}
		}
	}
	if current.Len() > 0 {
		rules = append(rules, current.String())
	}
	return rules
}

// looksLikeNewRule returns true when the string starts with a known rule keyword
// (i.e. the next token after a comma is a new rule, not a continuation of a param).
func looksLikeNewRule(s string) bool {
	known := []string{
		"required", "nullable", "email", "url", "uuid", "ip", "json",
		"boolean", "date", "alpha", "alpha_num", "alpha_dash", "numeric",
		"integer", "confirmed", "regex=", "min=", "max=", "size=",
		"gt=", "gte=", "lt=", "lte=", "digits=", "before=", "after=",
		"in=", "not_in=", "between=",
	}
	for _, k := range known {
		if strings.HasPrefix(s, k) {
			return true
		}
	}
	return false
}

func hasRule(rules []string, target string) bool {
	for _, r := range rules {
		if strings.TrimSpace(r) == target {
			return true
		}
	}
	return false
}

// findSiblingByJSONSuffix looks for a field in parent whose json name
// ends with the given suffix (e.g. "_confirmation").
// Used by 'confirmed': the field being validated IS the _confirmation field;
// we strip the suffix to find the original field to compare against.
func findSiblingByJSONSuffix(parent reflect.Value, confirmationField, suffix string) *reflect.Value {
	// confirmationField is e.g. "password_confirmation"
	// we want to find "password"
	base := strings.TrimSuffix(confirmationField, suffix)
	if base == confirmationField {
		// suffix not present — compare against a field named base+suffix instead
		// (fallback: look for <field>_confirmation of <field>)
		base = confirmationField + suffix
	}
	rt := parent.Type()
	for i := 0; i < rt.NumField(); i++ {
		if jsonFieldName(rt.Field(i)) == base {
			v := parent.Field(i)
			return &v
		}
	}
	return nil
}
