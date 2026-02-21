// Package bind decodes and validates an HTTP request body into a struct.
package bind

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"

	"github.com/shashiranjanraj/kashvi/config"
	"github.com/shashiranjanraj/kashvi/pkg/validate"
)

// maxBodyBytes returns the configured request body size limit (default 4 MB).
func maxBodyBytes() int64 {
	n, err := strconv.ParseInt(config.Get("MAX_BODY_BYTES", "4194304"), 10, 64)
	if err != nil || n <= 0 {
		return 4 << 20 // 4 MB
	}
	return n
}

// JSON decodes r.Body as JSON into dest and runs validation.
// The body is capped at MAX_BODY_BYTES (default 4 MB) to prevent memory exhaustion.
// Returns (errs, nil) when there are validation failures.
// Returns (nil, err) when the body is malformed JSON or too large.
func JSON(r *http.Request, dest interface{}) (errs map[string]string, err error) {
	r.Body = http.MaxBytesReader(nil, r.Body, maxBodyBytes())

	dec := json.NewDecoder(r.Body)
	if err = dec.Decode(dest); err != nil {
		var maxErr *http.MaxBytesError
		if errors.As(err, &maxErr) {
			return nil, fmt.Errorf("request body too large (max %d bytes)", maxErr.Limit)
		}
		return nil, fmt.Errorf("invalid JSON: %w", err)
	}

	errs = validate.Struct(dest)
	if validate.HasErrors(errs) {
		return errs, nil
	}

	return nil, nil
}
