// Package resource provides Laravel-style API Resource transformers.
//
// Define a Resource to control exactly what JSON shape your API returns:
//
//	type UserResource struct{ resource.Base }
//	func (r *UserResource) ToArray(v interface{}) resource.Map {
//	    u := v.(models.User)
//	    return resource.Map{
//	        "id":    u.ID,
//	        "name":  u.Name,
//	        "email": u.Email,
//	        "links": resource.Map{"self": "/api/users/" + fmt.Sprint(u.ID)},
//	    }
//	}
//
// Respond:
//
//	resource.New(&UserResource{}, user).Respond(w)
//	resource.Collection(&UserResource{}, users).Respond(w)
package resource

import (
	"encoding/json"
	"net/http"

	"github.com/shashiranjanraj/kashvi/pkg/orm"
)

// Map is a convenient alias for the output of ToArray.
type Map = map[string]interface{}

// Transformer defines the single method a Resource must implement.
type Transformer interface {
	// ToArray converts one model instance into a Map.
	ToArray(v interface{}) Map
}

// Base can be embedded in any Resource to satisfy future extension points.
type Base struct{}

// ------------------- Single resource -------------------

// Resource wraps a single model with its transformer.
type Resource struct {
	transformer Transformer
	data        interface{}
	meta        Map
}

// New creates a Resource for a single model instance.
func New(t Transformer, data interface{}) *Resource {
	return &Resource{transformer: t, data: data}
}

// WithMeta attaches additional metadata to the response envelope.
func (r *Resource) WithMeta(meta Map) *Resource {
	r.meta = meta
	return r
}

// MarshalJSON implements json.Marshaler so Resource can be nested.
func (r *Resource) MarshalJSON() ([]byte, error) {
	return json.Marshal(r.transformer.ToArray(r.data))
}

// Respond writes the resource as JSON with status 200.
func (r *Resource) Respond(w http.ResponseWriter) {
	out := Map{"data": r.transformer.ToArray(r.data)}
	if r.meta != nil {
		out["meta"] = r.meta
	}
	writeJSON(w, http.StatusOK, out)
}

// ------------------- Collection resource -------------------

// Collection wraps a slice of models with a transformer.
type Collection struct {
	transformer Transformer
	items       interface{}
	pagination  *orm.Pagination
	meta        Map
}

// CollectionOf creates a Collection from a slice (passed as interface{}).
// items should be a []SomeModel.
func CollectionOf(t Transformer, items interface{}) *Collection {
	return &Collection{transformer: t, items: items}
}

// WithPagination attaches pagination metadata.
func (c *Collection) WithPagination(p orm.Pagination) *Collection {
	c.pagination = &p
	return c
}

// WithMeta attaches extra metadata.
func (c *Collection) WithMeta(meta Map) *Collection {
	c.meta = meta
	return c
}

// Respond writes the collection as JSON with status 200.
func (c *Collection) Respond(w http.ResponseWriter) {
	// Use reflection-free iteration via json round-trip.
	raw, _ := json.Marshal(c.items)
	var rawSlice []json.RawMessage
	_ = json.Unmarshal(raw, &rawSlice)

	var result []interface{}
	for _, item := range rawSlice {
		var v interface{}
		_ = json.Unmarshal(item, &v)
		result = append(result, c.transformer.ToArray(v))
	}

	out := Map{"data": result}
	if c.pagination != nil {
		out["pagination"] = c.pagination
	}
	if c.meta != nil {
		out["meta"] = c.meta
	}
	writeJSON(w, http.StatusOK, out)
}

// ------------------- Helpers -------------------

func writeJSON(w http.ResponseWriter, status int, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v) //nolint:errcheck
}
