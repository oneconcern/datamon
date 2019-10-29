// Package auth allows for authenticating datamon against some external identity provider
package auth

import "github.com/oneconcern/datamon/pkg/model"

// Authable knows how to retrieve a principal from credentials
type Authable interface {
	Principal(credFile string) (model.Contributor, error)
}
