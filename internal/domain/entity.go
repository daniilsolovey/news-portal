package domain

// Entity represents a domain model
// Replace this with your actual domain entity
type Entity struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}

// Domain layer is used for converting data from delivery to necessary structs
// for further working with usecase and repository layers

// FromRequest converts request data to domain entity
// Replace this with your actual conversion logic
func FromRequest(data map[string]interface{}) (*Entity, error) {
	// TODO: implement conversion logic
	return &Entity{}, nil
}

// ToResponse converts domain entity to response format
// Replace this with your actual conversion logic
func (e *Entity) ToResponse() map[string]interface{} {
	return map[string]interface{}{
		"id":   e.ID,
		"name": e.Name,
	}
}

