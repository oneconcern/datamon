package core

import (
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"regexp"
	"sync"
	"testing"
	"time"

	"math/rand" // #nosec

	randfile "github.com/oneconcern/datamon/internal/rand"
	"github.com/oneconcern/datamon/pkg/cafs"
	context2 "github.com/oneconcern/datamon/pkg/context"
	"github.com/oneconcern/datamon/pkg/core/mocks"
	"github.com/oneconcern/datamon/pkg/model"
	"github.com/oneconcern/datamon/pkg/storage/localfs"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

const (
	commonLocation   = "common"
	modifiedLocation = "modified"
	conflictsDir     = ".conflicts"
)

var modifRex, contribRex, emailRex *regexp.Regexp

func init() {
	modifRex = regexp.MustCompile(`(\w+)\s+modified this file on`)
	contribRex = regexp.MustCompile(`service-user-(\w+)`)
	emailRex = regexp.MustCompile(`fred@(\w+)\.com`)
}

func testDiamondEnv() (mocks.TestEnv, func(testing.TB) func()) {
	tmp := stringOrDie(ioutil.TempDir("", "test-diamond-"))
	testRoot := stringOrDie(ioutil.TempDir(tmp, "core-data-"))

	// bundle stores
	sourceDir := filepath.Join(testRoot, "bundle", "source")
	// our download destination
	destinationDir := filepath.Join(testRoot, "bundle", "destination")
	// our source dataset to upload
	originalDir := filepath.Join(testRoot, "internal")

	// context
	blobDir := filepath.Join(sourceDir, "blob")
	metaDir := filepath.Join(sourceDir, "meta")
	vmetaDir := filepath.Join(sourceDir, "vmeta")
	wal := filepath.Join(sourceDir, "wal")
	readLog := filepath.Join(sourceDir, "readLog")

	for _, dir := range []string{
		sourceDir, destinationDir, originalDir,
		blobDir, metaDir, vmetaDir, wal, readLog,
	} {
		err := os.MkdirAll(dir, 0700)
		if err != nil {
			panic(fmt.Errorf("could not create test environment dir %s: %v", dir, err))
		}
	}

	return mocks.TestEnv{
			LeafSize:               cafs.DefaultLeafSize,
			Repo:                   "bundle-diamond-test-repo",
			TestRoot:               testRoot,
			SourceDir:              sourceDir,
			BlobDir:                blobDir,
			MetaDir:                metaDir,
			VmetaDir:               vmetaDir,
			Wal:                    wal,
			ReadLog:                readLog,
			DestinationDir:         destinationDir,
			ReBundleEntriesPerFile: 15,
			Original:               originalDir,
			DataDir:                "dir",
		}, func(t testing.TB) func() {
			return func() {
				t.Logf("unwinding diamond test environment")
				_ = os.RemoveAll(tmp)
			}
		}
}

// parameterization for sample fs generation
type genInputParam struct {
	pods                  []string
	meanNumFilesPerPod    int
	maxNumFilesPerPod     int
	fileNameLength        int
	minFileSizeLeafFactor float64
	maxFileSizeLeafFactor float64
}

func (p *genInputParam) lastPod() string {
	return p.pods[len(p.pods)-1]
}

func newGenInput(opts ...func(*genInputParam)) genInputParam {
	p := defaultGenInput()
	for _, apply := range opts {
		apply(&p)
	}
	return p
}

// TODO(fred): test(CI) - smaller sample size for CI

// defaultGenInput generates about 250 MB of random data
func defaultGenInput() genInputParam {
	return genInputParam{
		pods:                  []string{"pod1", "pod2", "pod3", "pod4", "pod5"},
		meanNumFilesPerPod:    10,
		maxNumFilesPerPod:     15,
		fileNameLength:        10,
		minFileSizeLeafFactor: 0.5,
		maxFileSizeLeafFactor: 5,
	}
}

// smallGenInput generates about 10 MB of random data
func smallGenInput(p *genInputParam) {
	p.pods = []string{"pod1", "pod2", "pod3"}
	p.meanNumFilesPerPod = 1000
	p.maxNumFilesPerPod = 1200
	p.fileNameLength = 20
	p.minFileSizeLeafFactor = 0.001
	p.maxFileSizeLeafFactor = 0.001
}

// makeDiamondInput produces a sample partitioned dataset to test diamonds
func makeDiamondInput(t testing.TB, dest string, leafSize uint32, p genInputParam) {
	t0 := time.Now()
	sampleSize := 0
	rand.Seed(time.Now().UnixNano()) // #nosec

	// constructs a randomized, partitioned set of files
	for _, splitDir := range p.pods {
		variability := rand.Intn((p.maxNumFilesPerPod-p.meanNumFilesPerPod)*2) - (p.maxNumFilesPerPod - p.meanNumFilesPerPod) // #nosec
		for i := 0; i < p.meanNumFilesPerPod+variability; i++ {
			name := randfile.LetterString(p.fileNameLength)
			size := int(float64(leafSize)*p.minFileSizeLeafFactor) + int(rand.Float64()*p.maxFileSizeLeafFactor*float64(leafSize)) // #nosec
			require.NoError(t, cafs.GenerateFile(filepath.Join(dest, splitDir, name), size, leafSize))
			sampleSize += size
		}
	}
	t.Logf("constructed sample data (%d bytes) in %v", sampleSize, time.Since(t0).Truncate(time.Second))
}

func TestDiamondUpload(t *testing.T) {
	ev, cleanup := testDiamondEnv()
	defer cleanup(t)()
	t.Logf("test location: %s", ev.TestRoot)
	ctx := mocks.FakeContext2(ev.MetaDir, ev.VmetaDir, ev.BlobDir)
	sample := newGenInput()
	makeDiamondInput(t, ev.Original, ev.LeafSize, sample)

	// create a repo
	require.NoError(t, CreateRepo(model.RepoDescriptor{Name: ev.Repo, Description: "test"}, ctx))

	// initialize a diamond
	diamond := NewDiamond(ev.Repo, ctx,
		DiamondLogger(mocks.TestLogger()),
		DiamondDescriptor(model.NewDiamondDescriptor(model.DiamondTag("coordinator"))),
	)
	_, err := CreateDiamond(ev.Repo, ctx, DiamondDescriptor(&diamond.DiamondDescriptor))
	require.NoError(t, err)

	diamondID := diamond.DiamondDescriptor.DiamondID

	// attempting an empty commit fails
	err = diamond.Commit()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "no split to commit")

	// create splits in parallel
	var wg sync.WaitGroup
	for _, toPin := range sample.pods {
		pod := toPin
		wg.Add(1)
		go func(wg *sync.WaitGroup) {
			defer wg.Done()
			testSplitAdd(t, diamondID, pod, ctx, ev)
		}(&wg)
	}
	mocks.TestLogger().Info("waiting for simulated pods")
	wg.Wait()

	verifySplitsAreDone(t, diamondID, ctx, sample, ev)

	// simulate an unfinished split: won't join the diamond
	_, err = CreateSplit(ev.Repo, diamondID, ctx)
	require.NoError(t, err)

	mocks.TestLogger().Info("commit started", zap.String("diamond_id", diamondID))
	require.NoError(t, diamond.Commit())

	bundleID := verifyDiamondAfterCommit(t, diamond, ctx, sample, ev, func(t testing.TB, meta model.DiamondDescriptor) {
		assert.False(t, meta.HasConflicts)
		assert.False(t, meta.HasCheckpoints)
		assert.Equal(t, model.EnableConflicts, meta.Mode)
		assert.Equal(t, "coordinator", meta.Tag)
	})

	downloadBundleAndCheck(t, bundleID, ctx, sample, ev)

	// attempting another commit fails
	err = diamond.Commit()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "cannot proceed with diamond commit")

	// adding a new split fails
	_, err = CreateSplit(ev.Repo, diamondID, ctx)
	require.Error(t, err)
}

func TestDiamondUploadWithConflicts(t *testing.T) {
	ev, cleanup := testDiamondEnv()
	defer cleanup(t)()
	t.Logf("test location: %s", ev.TestRoot)

	ctx := mocks.FakeContext2(ev.MetaDir, ev.VmetaDir, ev.BlobDir)
	sample := newGenInput(smallGenInput)
	makeDiamondInput(t, ev.Original, ev.LeafSize, sample)

	expectedNoConflicts, expectedConflicts, actualConflictsLocation := makeConflicts(t, ev)

	// create a repo
	require.NoError(t, CreateRepo(model.RepoDescriptor{Name: ev.Repo, Description: "test"}, ctx))

	// initialize a diamond
	diamond := NewDiamond(ev.Repo, ctx,
		DiamondLogger(mocks.TestLogger()),
		DiamondDescriptor(model.NewDiamondDescriptor(model.DiamondTag("coordinator"))),
	)
	_, err := CreateDiamond(ev.Repo, ctx, DiamondDescriptor(&diamond.DiamondDescriptor))
	require.NoError(t, err)

	diamondID := diamond.DiamondDescriptor.DiamondID

	// create splits (do it sequentially, in order to generate assertable conflicts)
	for _, toPin := range sample.pods {
		pod := toPin
		triggerConflicts(t, actualConflictsLocation, pod)
		testSplitAdd(t, diamondID, pod, ctx, ev, SplitKeyFilter(func(pth string) bool { // admit conlicts
			if dir := filepath.Base(filepath.Dir(pth)); dir == pod || dir == commonLocation {
				return true
			}
			if filepath.Base(filepath.Dir(filepath.Dir(pth))) == commonLocation {
				return true
			}
			return false
		}))
	}

	verifySplitsAreDone(t, diamondID, ctx, sample, ev)

	mocks.TestLogger().Info("commit started", zap.String("diamond_id", diamondID))

	// failure when conflicts are forbidden
	diamond.DiamondDescriptor.Mode = model.ForbidConflicts
	err = diamond.Commit()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "commit operation given up")

	// conflicts handled
	diamond.DiamondDescriptor.Mode = model.EnableConflicts
	require.NoError(t, diamond.Commit())

	bundleID := verifyDiamondAfterCommit(t, diamond, ctx, sample, ev, func(t testing.TB, meta model.DiamondDescriptor) {
		assert.True(t, meta.HasConflicts)
		assert.False(t, meta.HasCheckpoints)
		assert.Equal(t, model.EnableConflicts, meta.Mode)
		assert.Equal(t, "coordinator", meta.Tag)
	})

	downloadBundleAndCheck(t, bundleID, ctx, sample, ev)

	verifyConflicts(t, conflictsDir, commonLocation, modifiedLocation, expectedNoConflicts, expectedConflicts, sample, ev)

	// proceed with some update in dest, then upload as a new bundle
	// new bundle should not contain .conflicts (filtered)
	checkUpdateAndUpload(t, ctx, ev)
}

func checkUpdateAndUpload(t testing.TB, ctx context2.Stores, ev mocks.TestEnv) {
	done := false
	size := 100

	err := filepath.Walk(ev.DestinationDir, func(pth string, info os.FileInfo, erw error) error {
		// shuffle only one file
		if done || info.IsDir() || model.IsGeneratedFile(pth) || erw != nil {
			return nil
		}
		require.NoError(t, cafs.GenerateFile(pth, size, ev.LeafSize))
		done = true
		return nil
	})
	require.NoError(t, err)

	consumable := localfs.New(afero.NewBasePathFs(afero.NewOsFs(), ev.DestinationDir))
	newBundle := NewBundle(
		Repo(ev.Repo),
		ConsumableStore(consumable),
		ContextStores(ctx),
	)
	err = Upload(backgroundContexter(), newBundle)
	require.NoError(t, err)

	// download the new bundle to a new destination
	newDest := filepath.Join(ev.DestinationDir, "new")
	require.NoError(t, os.MkdirAll(newDest, 0777))
	consumable = localfs.New(afero.NewBasePathFs(afero.NewOsFs(), newDest))

	bundle := NewBundle(
		ContextStores(ctx),
		Repo(ev.Repo),
		BundleID(newBundle.BundleID),
		ConsumableStore(consumable),
		Logger(mocks.TestLogger()))

	require.NoError(t, Publish(backgroundContexter(), bundle))

	// assert downloaded content
	// * new bundle no more reports about conflicts
	_, err = os.Stat(filepath.Join(newDest, conflictsDir))
	assert.True(t, os.IsNotExist(err))
}

func TestDiamondUploadIgnoredConflicts(t *testing.T) {
	ev, cleanup := testDiamondEnv()
	defer cleanup(t)()
	t.Logf("test location: %s", ev.TestRoot)

	ctx := mocks.FakeContext2(ev.MetaDir, ev.VmetaDir, ev.BlobDir)
	sample := newGenInput(smallGenInput)
	makeDiamondInput(t, ev.Original, ev.LeafSize, sample)

	_, _, actualConflictsLocation := makeConflicts(t, ev)

	// create a repo
	require.NoError(t, CreateRepo(model.RepoDescriptor{Name: ev.Repo, Description: "test"}, ctx))

	// initialize a diamond
	diamond := NewDiamond(ev.Repo, ctx,
		DiamondLogger(mocks.TestLogger()),
		DiamondDescriptor(model.NewDiamondDescriptor(
			model.DiamondTag("coordinator"),
			model.DiamondMode(model.IgnoreConflicts))),
		DiamondMessage("diamond commit"),
	)
	_, err := CreateDiamond(ev.Repo, ctx, DiamondDescriptor(&diamond.DiamondDescriptor))
	require.NoError(t, err)

	diamondID := diamond.DiamondDescriptor.DiamondID

	// create splits (do it sequentially, in order to generate assertable conflicts)
	for _, toPin := range sample.pods {
		pod := toPin
		triggerConflicts(t, actualConflictsLocation, pod)
		testSplitAdd(t, diamondID, pod, ctx, ev, SplitKeyFilter(func(pth string) bool { // admit conlicts
			if dir := filepath.Base(filepath.Dir(pth)); dir == pod || dir == commonLocation {
				return true
			}
			if filepath.Base(filepath.Dir(filepath.Dir(pth))) == commonLocation {
				return true
			}
			return false
		}))
	}

	verifySplitsAreDone(t, diamondID, ctx, sample, ev)

	mocks.TestLogger().Info("commit started", zap.String("diamond_id", diamondID))

	// conflicts ignored
	require.NoError(t, diamond.Commit())

	bundleID := verifyDiamondAfterCommit(t, diamond, ctx, sample, ev, func(t testing.TB, meta model.DiamondDescriptor) {
		assert.False(t, meta.HasConflicts)
		assert.False(t, meta.HasCheckpoints)
		assert.Equal(t, model.IgnoreConflicts, meta.Mode)
		assert.Equal(t, "coordinator", meta.Tag)
	})

	downloadBundleAndCheck(t, bundleID, ctx, sample, ev, func(t testing.TB, b model.BundleDescriptor) {
		assert.Equal(t, "diamond commit", b.Message)
	})

	verifyIgnoredConflicts(t, conflictsDir, commonLocation, modifiedLocation, sample, ev)
}

func makeConflicts(t testing.TB, ev mocks.TestEnv) (int, int, string) {
	conflictsLocation := filepath.Join(ev.Original, commonLocation)
	expectedNoConflicts := injectConflicts(t, ev.Original, conflictsLocation, 0.01, "")
	t.Logf("generated identical files with no actual conflicts: %d", expectedNoConflicts)

	actualConflictsLocation := filepath.Join(ev.Original, commonLocation, modifiedLocation)
	expectedConflicts := injectConflicts(t, ev.Original, actualConflictsLocation, 0.01, fmt.Sprintf("initial version of this file at %v", time.Now()))
	t.Logf("generated identical files to trigger actual conflicts: %d", expectedConflicts)
	return expectedNoConflicts, expectedConflicts, actualConflictsLocation
}

func verifyConflicts(t testing.TB, specialDir, commonLocation, modifiedLocation string, expectedNoConflicts, expectedConflicts int, sample genInputParam, ev mocks.TestEnv) {
	// assert conflicts

	// - identical files do not trigger conflicts
	raw, err := ioutil.ReadDir(filepath.Join(ev.DestinationDir, commonLocation))
	require.NoError(t, err)

	nonconflicting := make([]os.FileInfo, 0, len(raw))
	for _, info := range raw {
		if !info.IsDir() {
			nonconflicting = append(nonconflicting, info)
		}
	}
	assert.Len(t, nonconflicting, expectedNoConflicts)

	// - modified files trigger conflicts
	conflictingSplits, err := ioutil.ReadDir(filepath.Join(ev.DestinationDir, specialDir))
	require.NoError(t, err)
	require.Len(t, conflictingSplits, len(sample.pods)-1)

	for _, conflicter := range conflictingSplits {
		// verify last split won
		assert.NotEqual(t, sample.lastPod(), conflicter.Name())
		require.True(t, conflicter.IsDir())
		conflicts := 0
		// versions reported as conflicts
		erw := filepath.Walk(filepath.Join(ev.DestinationDir, specialDir, conflicter.Name()),
			func(pth string, info os.FileInfo, erwf error) error {
				if info.IsDir() || erwf != nil {
					return nil
				}
				conflicts++
				data, erf := ioutil.ReadFile(pth)
				require.NoError(t, erf)

				matched := modifRex.FindSubmatch(data)
				require.Len(t, matched, 2)
				assert.Contains(t, sample.pods, string(matched[1]))
				assert.NotEqual(t, sample.lastPod(), string(matched[1]))

				// downloaded version: last split won
				data, erf = ioutil.ReadFile(filepath.Join(ev.DestinationDir, commonLocation, modifiedLocation, filepath.Base(pth)))
				require.NoError(t, erf)

				matched = modifRex.FindSubmatch(data)
				require.Len(t, matched, 2)
				assert.Equal(t, sample.lastPod(), string(matched[1]))
				return nil
			})
		require.NoError(t, erw)
		assert.Equal(t, expectedConflicts, conflicts)
	}
}

func verifyIgnoredConflicts(t testing.TB, specialDir, commonLocation, modifiedLocation string, sample genInputParam, ev mocks.TestEnv) {
	_, err := ioutil.ReadDir(filepath.Join(ev.DestinationDir, conflictsDir))
	require.Error(t, err)
	assert.True(t, os.IsNotExist(err))

	erw := filepath.Walk(filepath.Join(ev.DestinationDir, commonLocation, modifiedLocation),
		func(pth string, info os.FileInfo, erwf error) error {
			if info.IsDir() || erwf != nil {
				return nil
			}
			// downloaded version: last split won
			data, erf := ioutil.ReadFile(pth)
			require.NoError(t, erf)

			matched := modifRex.FindSubmatch(data)
			require.Len(t, matched, 2)
			assert.Equal(t, sample.lastPod(), string(matched[1]))
			return nil
		})
	require.NoError(t, erw)
}

func stringOrDie(arg string, err error) string {
	if err != nil {
		panic(err)
	}
	return arg
}

func testSplitAdd(t testing.TB, diamondID, pod string, ctx context2.Stores, ev mocks.TestEnv, extraOpts ...SplitOption) {
	// split add --path {ev.Original/pod}
	// set the consumable source directory for this simulated pod
	consumable := localfs.New(afero.NewBasePathFs(afero.NewOsFs(), ev.Original))

	splitOpts := []SplitOption{
		SplitDescriptor(model.NewSplitDescriptor(
			model.SplitTag(pod), // tag the logs
			model.SplitContributor(model.Contributor{Name: "service-user-" + pod, Email: "fred@" + pod + ".com"}),
		)),
		SplitConsumableStore(consumable),
		SplitLogger(mocks.TestLogger()),
		SplitKeyFilter(func(pth string) bool { // filtering on specific pod partition, rather than rebasing the split root
			return filepath.Base(filepath.Dir(pth)) == pod
		}),
	}
	splitOpts = append(splitOpts, extraOpts...)

	split := NewSplit(ev.Repo, diamondID, ctx, splitOpts...)
	_, err := CreateSplit(ev.Repo, diamondID, ctx, SplitDescriptor(&split.SplitDescriptor))
	require.NoError(t, err)

	// split add upload
	require.NoError(t, split.Upload())
}

func verifySplitsAreDone(t testing.TB, diamondID string, ctx context2.Stores, sample genInputParam, ev mocks.TestEnv) {
	// verify status of the splits: split list
	listed := make([]model.SplitDescriptor, 0, 5)
	err := ListSplitsApply(ev.Repo, diamondID, ctx,
		func(sd model.SplitDescriptor) error {
			listed = append(listed, sd)
			return nil
		})
	require.NoError(t, err)
	require.Len(t, listed, len(sample.pods))
	for _, split := range listed {
		require.Equal(t, model.SplitDone, split.State)
		assert.Len(t, split.Contributors, 1)
	}
}

func downloadBundleAndCheck(t testing.TB, bundleID string, ctx context2.Stores, sample genInputParam, ev mocks.TestEnv, addons ...func(testing.TB, model.BundleDescriptor)) {
	// bundle download

	// set the consumable destination directory for this bundle
	consumable := localfs.New(afero.NewBasePathFs(afero.NewOsFs(), ev.DestinationDir))

	bundle := NewBundle(
		ContextStores(ctx),
		Repo(ev.Repo),
		BundleID(bundleID),
		ConsumableStore(consumable),
		Logger(mocks.TestLogger()))

	require.NoError(t, Publish(backgroundContexter(), bundle))

	// assert downloaded content
	require.Truef(t, mocks.ValidateDataFiles(t, ev.Original, ev.DestinationDir), "downloaded bundle differ from original dataset")

	// assert contributors
	require.NoError(t, unpackBundleDescriptor(backgroundContexter(), bundle, false))
	assert.Len(t, bundle.BundleDescriptor.Contributors, len(sample.pods))
	for _, c := range bundle.BundleDescriptor.Contributors {
		// model.Contributor{Name: "service-user-" + pod, Email: "fred@" + pod + ".com"}),
		matches := emailRex.FindStringSubmatch(c.Email)
		require.Len(t, matches, 2)
		assert.Contains(t, sample.pods, matches[1])

		matches = contribRex.FindStringSubmatch(c.Name)
		require.Len(t, matches, 2)
		assert.Contains(t, sample.pods, matches[1])
	}

	for _, addon := range addons {
		addon(t, bundle.BundleDescriptor)
	}
}

func verifyDiamondAfterCommit(t testing.TB, diamond *Diamond, ctx context2.Stores, sample genInputParam, ev mocks.TestEnv, addons ...func(testing.TB, model.DiamondDescriptor)) string {
	bundleID := diamond.DiamondDescriptor.BundleID
	require.NotEmpty(t, bundleID)
	require.Equal(t, bundleID, diamond.BundleID)
	require.Equal(t, bundleID, diamond.BundleDescriptor.ID)

	// assert diamond metadata
	meta, err := GetDiamond(diamond.RepoID, diamond.DiamondDescriptor.DiamondID, ctx)
	require.NoError(t, err)
	assert.Equal(t, model.DiamondDone, meta.State)
	assert.Len(t, meta.Splits, len(sample.pods))
	assert.NotEmpty(t, meta.EndTime)

	for _, addon := range addons {
		addon(t, meta)
	}
	return bundleID
}

// injectConflicts randomly copies some files to a common location
func injectConflicts(t testing.TB, root, target string, rate float64, alterData string) int {
	conflicts := 0
	require.NoError(t, os.MkdirAll(target, 0755))
	require.NoError(t, filepath.Walk(root, func(pth string, info os.FileInfo, err error) error {
		if info.IsDir() || err != nil {
			return nil
		}
		//#nosec
		if rand.Float64() < rate {
			conflicts++

			// copy file to a commonly uploaded location
			source, err := os.Open(pth)
			require.NoError(t, err)

			destination, err := os.Create(filepath.Join(target, filepath.Base(pth)))
			require.NoError(t, err)

			if alterData != "" {
				_, erw := destination.WriteString(alterData)
				require.NoError(t, erw)
			}

			_, err = io.Copy(destination, source)
			require.NoError(t, err)
			_ = source.Close()
			_ = destination.Close()
		}
		return nil
	}))
	return conflicts
}

// triggerConflicts inserts a change to some files
func triggerConflicts(t testing.TB, target, signature string) int {
	tmpDir, err := ioutil.TempDir(filepath.Dir(target), "")
	require.NoError(t, err)

	injected := injectConflicts(t, target, tmpDir, 1.00, fmt.Sprintf("%s modified this file on %v", signature, time.Now()))
	_ = os.RemoveAll(target)
	_ = os.Rename(tmpDir, target)
	return injected
}

func TestBasenameFilter(t *testing.T) {

}
