package google

import (
	"context"
	"time"

	"github.com/oneconcern/datamon/pkg/auth/status"
	"github.com/oneconcern/datamon/pkg/model"
	goauth "google.golang.org/api/oauth2/v2"
	goption "google.golang.org/api/option"
)

const timeout = 60 * time.Second

// New returns a new instance of google Auth
func New() Auth {
	return Auth{}
}

// Auth implements Authable for google credentials
type Auth struct {
}

// Principal queries google oauth2 with some local credentials to extract user
// information (aka principal).
//
// By default, credentials are taken from your default application_default_credentials.
// On linux, this is located at ~/.config/gcloud/application_default_credentials.json.
func (g Auth) Principal(credFile string) (model.Contributor, error) {

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	svc, err := goauth.NewService(ctx,
		goption.WithCredentialsFile(credFile),
		goption.WithScopes(goauth.UserinfoEmailScope, goauth.UserinfoProfileScope),
	)
	if err != nil {
		return model.Contributor{}, status.ErrAuthService.Wrap(err)
	}

	var u *goauth.Userinfoplus
	u, err = svc.Userinfo.Get().Do()
	if err != nil {
		return model.Contributor{}, status.ErrUserinfo.Wrap(err)
	}
	fullName := func() (name string) {
		if u.Name != "" {
			name = u.Name
			return
		}
		if u.GivenName != "" {
			name += u.GivenName + " "
		}
		if u.FamilyName != "" {
			name += u.FamilyName + " "
		}
		if name == "" {
			// fall back on email if no nominative attributes are set
			name = u.Email
		}
		return
	}

	if u.Email == "" {
		return model.Contributor{}, status.ErrEmailScope
	}
	// NOTE(frederic): at this moment, the profile scope is not required and we fall back
	// on email for the name if the full name is not available.

	return model.Contributor{
		Email: u.Email,
		Name:  fullName(),
	}, nil
}
