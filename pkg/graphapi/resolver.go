package graphapi

import (
	context "context"
	"errors"
	"strconv"
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
	lst, err := q.engine.ListRepo(ctx)
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
	repo, err := q.engine.GetRepo(ctx, name)
	if err != nil {
		return nil, err
	}

	res := convertRepo(repo)
	tags, err := loadTags(ctx, repo)
	if err != nil {
		return nil, err
	}

	branches, err := loadBranches(ctx, repo)
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

	rep, err := m.engine.CreateRepo(ctx, repo.Name, swag.StringValue(repo.Description))
	if err != nil {
		return nil, err
	}

	res := convertRepo(rep)
	return &res, nil
}

func (m *mutationResolver) CreateBundle(ctx context.Context, params BundleInput) (*Bundle, error) {
	repo, err := m.engine.GetRepo(ctx, params.Repository)
	if err != nil {
		return nil, err
	}

	changes, err := readInputChangeSet(params.Changes)
	if err != nil {
		return nil, err
	}

	nb, err := repo.CommitFromChangeSet(ctx, params.Message, swag.StringValue(params.Branch), changes)
	if err != nil {
		return nil, err
	}

	sb, err := repo.GetBundle(ctx, nb.ID)
	if err != nil {
		return nil, err
	}

	res := convertBundle(sb)
	return &res, nil
}

func readInputChangeSet(c *ChangeSetInput) (result store.ChangeSet, err error) {
	result.Added = make([]store.Entry, len(c.Added))
	for i, o := range c.Added {
		e, err := readInputObject(&o)
		if err != nil {
			return store.ChangeSet{}, err
		}
		result.Added[i] = e
	}
	result.Deleted = make([]store.Entry, len(c.Deleted))
	for i, o := range c.Deleted {
		e, err := readInputObject(&o)
		if err != nil {
			return store.ChangeSet{}, err
		}
		result.Deleted[i] = e
	}
	return
}

func readInputObject(o *ObjectInput) (result store.Entry, err error) {
	m, err := readMode(swag.StringValue(o.Mode))
	if err != nil {
		return result, err
	}
	result.Hash = o.ID
	result.Path = o.Path
	result.Mode = m
	result.Mtime = o.Mtime
	return
}

func readMode(m string) (store.FileMode, error) {
	if m == "" {
		m = "0600"
	}
	res, err := strconv.ParseUint(m, 8, 32)
	if err != nil {
		return 0, err
	}
	return store.FileMode(uint32(res)), nil
}

func (m *mutationResolver) DeleteRepository(ctx context.Context, id string) (*Repository, error) {
	repo, err := m.engine.GetRepo(ctx, id)
	if err != nil {
		return nil, err
	}

	return nil, m.engine.DeleteRepo(ctx, repo.Name)
}

func (m *mutationResolver) DeleteBranch(ctx context.Context, repository string, branch string) (*string, error) {
	repo, err := m.engine.GetRepo(ctx, repository)
	if err != nil {
		return nil, err
	}

	if err := repo.DeleteBranch(ctx, branch); err != nil {
		return nil, err
	}
	return &branch, nil
}

func (m *mutationResolver) DeleteTag(ctx context.Context, repository string, tag string) (*string, error) {
	repo, err := m.engine.GetRepo(ctx, repository)
	if err != nil {
		return nil, err
	}

	if err := repo.DeleteTag(ctx, tag); err != nil {
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

	repo, err := r.engine.GetRepo(ctx, obj.Name)
	if err != nil {
		return nil, err
	}

	return loadTags(ctx, repo)
}

func (r *repositoryResolver) Branches(ctx context.Context, obj *Repository) ([]BundleRef, error) {
	if len(obj.Branches) > 0 { // when eagerly loaded, use those otherwise fetch
		return obj.Branches, nil
	}

	repo, err := r.engine.GetRepo(ctx, obj.Name)
	if err != nil {
		return nil, err
	}

	return loadBranches(ctx, repo)
}

func (r *repositoryResolver) Bundle(ctx context.Context, obj *Repository, id string) (*Bundle, error) {
	repo, err := r.engine.GetRepo(ctx, obj.Name)
	if err != nil {
		return nil, err
	}

	b, err := repo.GetBundle(ctx, id)
	if err != nil {
		return nil, err
	}

	res := convertBundle(b)
	return &res, nil
}

func (r *repositoryResolver) Snapshot(ctx context.Context, obj *Repository, id string) (*Snapshot, error) {
	repo, err := r.engine.GetRepo(ctx, obj.Name)
	if err != nil {
		return nil, err
	}

	sn, err := repo.Checkout(ctx, "", id)
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

func loadTags(ctx context.Context, r *engine.Repo) ([]BundleRef, error) {
	tags, err := r.ListTags(ctx)
	if err != nil {
		return nil, err
	}
	return bundlesForCommits(ctx, r, NameTypeTag, tags)
}

func loadBranches(ctx context.Context, r *engine.Repo) ([]BundleRef, error) {
	branches, err := r.ListBranches(ctx)
	if err != nil {
		return nil, err
	}
	return bundlesForCommits(ctx, r, NameTypeBranch, branches)
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

func convertBundle(b *store.Bundle) Bundle {

	return Bundle{
		ID:        b.ID,
		Message:   &b.Message,
		Timestamp: b.Timestamp,
		Changes:   convertChangeSet(&b.Changes),
	}
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

func bundlesForCommits(ctx context.Context, repo *engine.Repo, tpe NameType, names []string) ([]BundleRef, error) {
	result := make([]BundleRef, len(names))
	for i, name := range names {
		b, err := repo.GetBundle(ctx, name)
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
