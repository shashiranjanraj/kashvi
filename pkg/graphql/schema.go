package graphql

import (
	"github.com/graphql-go/graphql"
)

// NewSchema creates a new GraphQL schema from a provided RootQuery
func NewSchema(query *graphql.Object) (graphql.Schema, error) {
	return graphql.NewSchema(graphql.SchemaConfig{
		Query: query,
	})
}
