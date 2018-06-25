package store

// ACL represents the configuration of access control lists
//
// A permission can be granted to a group or a user
type ACL struct {
	Read  []string `json:"read,omitempty" yaml:"read,omitempty"`
	Write []string `json:"write,omitempty" yaml:"write,omitempty"`
	Admin []string `json:"admin,omitempty" yaml:"admin,omitempty"`
}

// Repo represents a repository in the trumpet
type Repo struct {
	Name        string            `json:"name" yaml:"name"`
	Description string            `json:"description,omitempty" yaml:"description,omitempty"`
	TagsRef     map[string]string `json:"tags,omitempty" yaml:"tags,omitempty"`
	BranchRef   map[string]string `json:"branches,omitempty" yaml:"branches,omitempty"`

	ACL *ACL `json:"acl,omitempty" yaml:"acl,omitempty"`

	refTags     map[string][]string
	refBranches map[string][]string
}

// Bundle represents a commit which is a file tree with the changes to the repository.
type Bundle struct {
	ID      string   `json:"id" yaml:"id"`
	Parents []string `json:"parents" yaml:"parents"`
}
