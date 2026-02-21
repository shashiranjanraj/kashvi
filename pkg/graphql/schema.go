package graphql

import (
	"github.com/graphql-go/graphql"
	"github.com/shashiranjanraj/kashvi/app/models"
	"github.com/shashiranjanraj/kashvi/pkg/orm"
)

// OrderType represents a GraphQL order object.
var OrderType = graphql.NewObject(graphql.ObjectConfig{
	Name: "Order",
	Fields: graphql.Fields{
		"id":      &graphql.Field{Type: graphql.Int},
		"user_id": &graphql.Field{Type: graphql.Int},
		"total":   &graphql.Field{Type: graphql.Float},
		"status":  &graphql.Field{Type: graphql.String},
	},
})

// UserType represents a GraphQL user object with nested orders.
var UserType = graphql.NewObject(graphql.ObjectConfig{
	Name: "User",
	Fields: graphql.Fields{
		"id":    &graphql.Field{Type: graphql.Int},
		"name":  &graphql.Field{Type: graphql.String},
		"email": &graphql.Field{Type: graphql.String},
		"orders": &graphql.Field{
			Type: graphql.NewList(OrderType),
			Resolve: func(p graphql.ResolveParams) (interface{}, error) {
				user, ok := p.Source.(models.User)
				if !ok {
					return nil, nil
				}

				var orders []models.Order
				if err := orm.DB().
					Model(&models.Order{}).
					Where("user_id = ?", user.ID).
					Get(&orders); err != nil {
					return nil, err
				}

				return orders, nil
			},
		},
	},
})

// RootQuery is the top-level GraphQL query object.
var RootQuery = graphql.NewObject(graphql.ObjectConfig{
	Name: "Query",
	Fields: graphql.Fields{
		"users": &graphql.Field{
			Type: graphql.NewList(UserType),
			Resolve: func(p graphql.ResolveParams) (interface{}, error) {
				var users []models.User
				err := orm.DB().Model(&models.User{}).Get(&users)
				return users, err
			},
		},
		"user": &graphql.Field{
			Type: UserType,
			Args: graphql.FieldConfigArgument{
				"id": &graphql.ArgumentConfig{Type: graphql.NewNonNull(graphql.Int)},
			},
			Resolve: func(p graphql.ResolveParams) (interface{}, error) {
				id := p.Args["id"].(int)
				var user models.User
				user.ID = uint(id)
				err := orm.DB().Model(&models.User{}).Where("id = ?", id).First(&user)
				return user, err
			},
		},
	},
})

// Schema is the built graphql-go schema.
var Schema, _ = graphql.NewSchema(graphql.SchemaConfig{
	Query: RootQuery,
})
