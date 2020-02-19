package core

import (
	"context"
	"sort"

	"github.com/oneconcern/datamon/pkg/core/status"
	"github.com/oneconcern/datamon/pkg/model"
	"github.com/oneconcern/datamon/pkg/storage"
)

const (
	typicalContextsNum = 16 // default number of allocated memory slots for contexts in a config
)

// ListContexts provides the list of available contexts in a remote configuration store, sorted.
//
// TODO(fred): make context a first class citizen in core
func ListContexts(config storage.Store) ([]string, error) {
	contexts := make([]string, 0, typicalContextsNum)
	iterator := func(next string) ([]string, string, error) {
		return config.KeysPrefix(context.Background(), next, model.GetArchivePathPrefixToContexts(), "", typicalContextsNum)
	}

	var (
		keys []string
		next string
		err  error
	)
	for {
		keys, next, err = iterator(next)
		if err != nil {
			return nil, status.ErrConfigContext.Wrap(err)
		}
		for _, k := range keys {
			c, erp := model.GetArchivePathComponents(k)
			if erp != nil {
				return nil, status.ErrConfigContext.Wrap(err)
			}
			contexts = append(contexts, c.Context)
		}
		if next == "" {
			break
		}
	}
	sort.Strings(contexts)
	return contexts, nil
}
