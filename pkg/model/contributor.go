package model

import "fmt"

// Contributor who created the object
type Contributor struct {
	Name  string `json:"name" yaml:"name"`
	Email string `json:"email" yaml:"email"`
	_     struct{}
}

func (c *Contributor) String() string {
	if c.Email == "" {
		return c.Name
	}
	if c.Name == "" {
		return c.Email
	}
	return fmt.Sprintf("%s <%s>", c.Name, c.Email)
}
