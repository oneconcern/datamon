package graphapi

import (
	context "context"
	"errors"
	"strings"

	"github.com/go-openapi/swag"
	"github.com/oneconcern/trumpet/pkg/engine"
	"github.com/oneconcern/trumpet/pkg/store"
)

// NewResolvers creates a new resolver root implementation
func NewResolvers(engine *engine.Runtime) ResolverRoot {
	return &rootResolver{
		query:        &queryResolver{engine},
		mutations:    &mutationResolver{engine},
		repositories: &repositoryResolver{engine},
	}
}

type rootResolver struct {
	query        *queryResolver
	mutations    *mutationResolver
	repositories *repositoryResolver
}

func (r *rootResolver) Mutation() MutationResolver {
	return r.mutations
}
func (r *rootResolver) Query() QueryResolver {
	return r.query
}
func (r *rootResolver) Repository() RepositoryResolver {
	return r.repositories
}

type queryResolver struct {
	engine *engine.Runtime
}

func (q *queryResolver) Repositories(ctx context.Context) ([]Repository, error) {
	lst, err := q.engine.ListRepo()
	if err != nil {
		return nil, err
	}

	conv := make([]Repository, len(lst))
	for i, v := range lst {
		conv[i] = convertRepo(&v)
	}
	return conv, nil
}

func (q *queryResolver) Repository(ctx context.Context, name string) (*Repository, error) {
	repo, err := q.engine.GetRepo(name)
	if err != nil {
		return nil, err
	}

	res := convertRepo(repo)
	tags, err := loadTags(repo)
	if err != nil {
		return nil, err
	}

	branches, err := loadBranches(repo)
	if err != nil {
		return nil, err
	}

	res.Branches = branches
	res.Tags = tags
	return &res, nil
}

type mutationResolver struct {
	engine *engine.Runtime
}

func (m *mutationResolver) CreateRepository(ctx context.Context, repo RepositoryInput) (*Repository, error) {
	if strings.TrimSpace(repo.Name) == "" {
		return nil, errors.New("name is required")
	}

	rep, err := m.engine.CreateRepo(repo.Name, swag.StringValue(repo.Description))
	if err != nil {
		return nil, err
	}

	res := convertRepo(rep)
	return &res, nil
}

func (m *mutationResolver) CreateBundle(ctx context.Context, params BundleInput) (*Bundle, error) {
	return nil, nil
}

func (m *mutationResolver) DeleteRepository(ctx context.Context, id string) (*Repository, error) {
	repo, err := m.engine.GetRepo(id)
	if err != nil {
		return nil, err
	}

	return nil, m.engine.DeleteRepo(repo.Name)
}

func (m *mutationResolver) DeleteBranch(ctx context.Context, repository string, branch string) (*string, error) {
	repo, err := m.engine.GetRepo(repository)
	if err != nil {
		return nil, err
	}

	if err := repo.DeleteBranch(branch); err != nil {
		return nil, err
	}
	return &branch, nil
}

func (m *mutationResolver) DeleteTag(ctx context.Context, repository string, tag string) (*string, error) {
	repo, err := m.engine.GetRepo(repository)
	if err != nil {
		return nil, err
	}

	if err := repo.DeleteTag(tag); err != nil {
		return nil, err
	}
	return &tag, nil
}

type repositoryResolver struct {
	engine *engine.Runtime
}

func (r *repositoryResolver) Tags(ctx context.Context, obj *Repository) ([]BundleRef, error) {
	if len(obj.Tags) > 0 { // when eagerly loaded, use those otherwise fetch
		return obj.Tags, nil
	}

	repo, err := r.engine.GetRepo(obj.Name)
	if err != nil {
		return nil, err
	}

	return loadTags(repo)
}

func (r *repositoryResolver) Branches(ctx context.Context, obj *Repository) ([]BundleRef, error) {
	if len(obj.Branches) > 0 { // when eagerly loaded, use those otherwise fetch
		return obj.Branches, nil
	}

	repo, err := r.engine.GetRepo(obj.Name)
	if err != nil {
		return nil, err
	}

	return loadBranches(repo)
}

func (r *repositoryResolver) Bundle(ctx context.Context, obj *Repository, id string) (*Bundle, error) {
	repo, err := r.engine.GetRepo(obj.Name)
	if err != nil {
		return nil, err
	}

	b, err := repo.GetBundle(id)
	if err != nil {
		return nil, err
	}

	return &Bundle{
		ID:        b.ID,
		Message:   &b.Message,
		Timestamp: b.Timestamp,
		Changes:   convertChangeSet(&b.Changes),
	}, nil
}

func (r *repositoryResolver) Snapshot(ctx context.Context, obj *Repository, id string) (*Snapshot, error) {
	repo, err := r.engine.GetRepo(obj.Name)
	if err != nil {
		return nil, err
	}

	sn, err := repo.Checkout("", id)
	if err != nil {
		return nil, err
	}

	res := convertSnapshot(sn)
	return &res, nil
}

func convertRepo(r *engine.Repo) Repository {
	return Repository{
		Name:          r.Name,
		Description:   swag.String(r.Description),
		DefaultBranch: r.CurrentBranch,
	}
}

func loadTags(r *engine.Repo) ([]BundleRef, error) {
	tags, err := r.ListTags()
	if err != nil {
		return nil, err
	}
	return bundlesForCommits(r, NameTypeTag, tags)
}

func loadBranches(r *engine.Repo) ([]BundleRef, error) {
	branches, err := r.ListBranches()
	if err != nil {
		return nil, err
	}
	return bundlesForCommits(r, NameTypeBranch, branches)
}

func convertChangeSet(cs *store.ChangeSet) (ncs ChangeSet) {
	ncs.Added = make([]VersionedObject, len(cs.Added))
	for i, e := range cs.Added {
		ncs.Added[i] = convertEntry(&e)
	}

	ncs.Deleted = make([]VersionedObject, len(cs.Deleted))
	for i, e := range cs.Deleted {
		ncs.Deleted[i] = convertEntry(&e)
	}

	return
}

func convertEntry(et *store.Entry) (o VersionedObject) {
	o.Path = et.Path
	o.ID = et.Hash
	o.Mode = swag.String(et.Mode.String())
	o.Mtime = et.Mtime
	return
}

func convertSnapshot(sn *store.Snapshot) Snapshot {
	objs := make([]VersionedObject, len(sn.Entries))
	for i, e := range sn.Entries {
		objs[i] = convertEntry(&e)
	}

	return Snapshot{
		Bundle:          swag.String(sn.NewCommit),
		ID:              sn.ID,
		Objects:         objs,
		Parents:         sn.Parents[:],
		PreviousCommits: sn.PreviousCommits[:],
		Timestamp:       sn.Timestamp,
	}
}

func bundlesForCommits(repo *engine.Repo, tpe NameType, names []string) ([]BundleRef, error) {
	result := make([]BundleRef, len(names))
	for i, name := range names {
		b, err := repo.GetBundle(name)
		if err != nil {
			return nil, err
		}
		result[i] = BundleRef{
			ID: b.ID,
			Name: &Named{
				Type:  tpe,
				Value: name,
			},
		}
	}
	return result, nil
}
