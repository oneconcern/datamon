package trumpet

// func NewRepo(name, description, branch string) *Repo {
// 	return &Repo{
// 		Name:          name,
// 		Description:   description,
// 		CurrentBranch: branch,
// 		stage:         NewStage(""),
// 	}
// }

// Repo is the object that manages repositories
type Repo struct {
	Name          string
	Description   string
	CurrentBranch string

	baseDir string
	stage   *Stage
}

// Stage to record pending changes into
func (r *Repo) Stage() *Stage {
	return r.stage
}
