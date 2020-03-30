package core

import (
	"context"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"strings"

	"github.com/oneconcern/datamon/pkg/model"
	"github.com/oneconcern/datamon/pkg/storage"
	"gopkg.in/yaml.v2"
)

type testReadCloserWithErr struct {
}

func (testReadCloserWithErr) Read(_ []byte) (int, error) {
	return 0, errors.New("io error")
}
func (testReadCloserWithErr) Close() error {
	return nil
}

func fakeBundlePath(repo, bundleID string) string {
	return model.GetArchivePathToBundle(repo, bundleID)
}

func fakeRepoPath(repo string) string {
	return model.GetArchivePathToRepoDescriptor(repo)
}

func fakeLabelPath(repo, id string) string {
	return model.GetArchivePathToLabel(repo, id)
}

func extractBundleID(pth string) string {
	comp, err := model.GetArchivePathComponents(pth)
	if err != nil {
		panic(fmt.Sprintf("test config error: %v", err))
	}
	if comp.BundleID == "" {
		panic(fmt.Sprintf("test config error: did not find a bundle in this path: %s", pth))
	}
	return comp.BundleID
}

func extractRepo(pth string) string {
	comp, err := model.GetArchivePathComponents(pth)
	if err != nil {
		panic(fmt.Sprintf("test config error: %v", err))
	}
	if comp.Repo == "" {
		panic(fmt.Sprintf("test config error: did not find a repo in this path: %s", pth))
	}
	return comp.Repo
}

func extractIDFromLabel(pth string) string {
	comp, err := model.GetArchivePathComponents(pth)
	if err != nil {
		panic(fmt.Sprintf("test config error: %v", err))
	}
	if comp.LabelName == "" {
		panic(fmt.Sprintf("test config error: did not find a label in this path: %s", pth))
	}
	return comp.LabelName
}

func fakeBD(bundleID string) model.BundleDescriptor {
	return model.BundleDescriptor{
		ID:           bundleID,
		LeafSize:     16,
		Message:      "this is a message",
		Version:      4,
		Contributors: []model.Contributor{},
	}
}

func fakeRD(repo string) model.RepoDescriptor {
	return model.RepoDescriptor{
		Name:        repo,
		Description: fmt.Sprintf("test %s", repo),
		Contributor: model.Contributor{Email: "test@example.com"},
	}
}

func fakeLD(label string) model.LabelDescriptor {
	return model.LabelDescriptor{
		Name:      label,
		BundleID:  fmt.Sprintf("bundle-%s", label),
		Timestamp: testTime(),
		Contributors: []model.Contributor{
			{Email: "test1@example.com"},
			{Email: "test2@example.com"},
		},
	}
}

func fakeLV(gcsGeneration int64) storage.Version {
	return storage.NewVersionGcs(gcsGeneration)
}

func buildBundleYaml(id string) string {
	asYaml, _ := yaml.Marshal(fakeBD(id))
	return string(asYaml)
}

func buildRepoYaml(repo string) string {
	asYaml, _ := yaml.Marshal(fakeRD(repo))
	return string(asYaml)
}

func buildLabelYaml(id string) string {
	asYaml, _ := yaml.Marshal(fakeLD(id))
	return string(asYaml)
}

func garbleYaml(in string) string {
	return in + `
>>>> # this line intentionally invalid YAML
	`
}

func goodKeysPrefixFunc(repo string) func(context.Context, string, string, string, int) ([]string, string, error) {
	return func(_ context.Context, _ string, _ string, _ string, _ int) ([]string, string, error) {
		return []string{fakeBundlePath(repo, "myID1"), fakeBundlePath(repo, "myID2"), fakeBundlePath(repo, "myID3")}, "", nil
	}
}

func goodWindowKeysPrefixFunc(keysBatchFixture []string) func(context.Context, string, string, string, int) ([]string, string, error) {
	return func(_ context.Context, next string, _ string, _ string, count int) ([]string, string, error) {
		index := 0
		window := minInt(count, len(keysBatchFixture))

		switch next {
		case "":
			break
		default:
			for i, key := range keysBatchFixture {
				if key == next {
					index = i
					break
				}
			}
		}
		var following string
		if index+window < len(keysBatchFixture) {
			following = keysBatchFixture[index+window]
		}
		last := minInt(index+window, len(keysBatchFixture))
		return keysBatchFixture[index:last], following, nil
	}
}

func breakAfterFourBatches(keysBatchFixture []string) func(context.Context, string, string, string, int) ([]string, string, error) {
	return func(_ context.Context, next string, _ string, _ string, count int) ([]string, string, error) {
		// returns an error somewhere within the batch
		index := 0
		window := minInt(count, len(keysBatchFixture))

		switch next {
		case "":
			break
		default:
			for i, key := range keysBatchFixture {
				if key == next {
					index = i
					break
				}
			}
		}

		if index > 4*testBatchSize {
			return nil, "", errors.New("test key fetch error")
		}

		var following string
		if index+window < len(keysBatchFixture) {
			following = keysBatchFixture[index+window]
		}
		last := minInt(index+window, len(keysBatchFixture))
		return keysBatchFixture[index:last], following, nil
	}
}

func goodHasFunc(_ context.Context, _ string) (bool, error) {
	return true, nil
}

func goodKeysFunc(_ context.Context) ([]string, error) {
	return nil, nil
}

func goodGetFunc(_ context.Context, pth string) (io.ReadCloser, error) {
	id := extractBundleID(pth)
	return ioutil.NopCloser(strings.NewReader(buildBundleYaml(id))), nil
}

func goodGetRepoFunc(_ context.Context, pth string) (io.ReadCloser, error) {
	repo := extractRepo(pth)
	return ioutil.NopCloser(strings.NewReader(buildRepoYaml(repo))), nil
}

func goodGetLabelFunc(_ context.Context, pth string) (io.ReadCloser, error) {
	id := extractIDFromLabel(pth)
	return ioutil.NopCloser(strings.NewReader(buildLabelYaml(id))), nil
}

func breakAferFiveBundlesGetFunc(_ context.Context, pth string) (io.ReadCloser, error) {
	// returns an error somewhere within the batch
	id := extractBundleID(pth)
	index := 0
	for i, key := range keysBatchFixture {
		if strings.Contains(key, id) {
			index = i
			break
		}
	}
	if index > 5*testBatchSize {
		return nil, errors.New("test repo fetch error")
	}
	return ioutil.NopCloser(strings.NewReader(buildBundleYaml(id))), nil
}
