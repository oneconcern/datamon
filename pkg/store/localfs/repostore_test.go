package localfs

import (
	"context"
	"io/ioutil"
	"os"
	"testing"

	"github.com/oneconcern/datamon/pkg/store"
	"github.com/stretchr/testify/require"
)

func TestCreateRepo(t *testing.T) {
	td, err := ioutil.TempDir("", "tpt-tst")
	require.NoError(t, err)
	defer os.RemoveAll(td)

	st := NewRepos(td)
	require.NoError(t, st.Initialize())
	defer st.Close()

	ctx := context.Background()

	err = st.Create(ctx, &store.Repo{
		Name:        "test-repo",
		Description: "the repository for tests",
	})
	require.NoError(t, err)

	repo, err := st.Get(ctx, "test-repo")
	require.NoError(t, err)
	require.Equal(t, "the repository for tests", repo.Description)

	err = st.Create(ctx, &store.Repo{
		Name:        "test-repo",
		Description: "another description",
	})
	require.Error(t, err)

	repo, err = st.Get(ctx, "test-repo")
	require.NoError(t, err)
	require.Equal(t, "the repository for tests", repo.Description)
}

func TestDeleteRepo(t *testing.T) {
	td, err := ioutil.TempDir("", "tpt-tst")
	require.NoError(t, err)
	defer os.RemoveAll(td)

	st := NewRepos(td)
	require.NoError(t, st.Initialize())
	defer st.Close()

	ctx := context.Background()

	err = st.Create(ctx, &store.Repo{
		Name:        "test-repo",
		Description: "the repository for tests",
	})
	require.NoError(t, err)

	err = st.Create(ctx, &store.Repo{
		Name:        "test-repo-2",
		Description: "the 2nd repository for tests",
	})
	require.NoError(t, err)

	err = st.Create(ctx, &store.Repo{
		Name:        "test-repo-3",
		Description: "the 3rd repository for tests",
	})
	require.NoError(t, err)

	err = st.Delete(ctx, "test-repo-2")
	require.NoError(t, err)

	_, err = st.Get(ctx, "test-repo-2")
	require.EqualError(t, store.RepoNotFound, err.Error())

	names, err := st.List(ctx)
	require.NoError(t, err)
	require.Len(t, names, 2)
}
