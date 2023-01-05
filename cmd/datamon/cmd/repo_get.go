package cmd

import (
	"bytes"
	"context"
	"sync/atomic"
	"text/template"
	"time"

	"github.com/docker/go-units"
	context2 "github.com/oneconcern/datamon/pkg/context"
	"github.com/oneconcern/datamon/pkg/core"
	status "github.com/oneconcern/datamon/pkg/core/status"
	"github.com/oneconcern/datamon/pkg/errors"
	"github.com/oneconcern/datamon/pkg/model"
	"github.com/spf13/cobra"
	"golang.org/x/sys/unix"
)

// GetRepoCommand retrieves the description of a repo
var GetRepoCommand = &cobra.Command{
	Use:   "get",
	Short: "Get repo info by name",
	Long: `Performs a direct lookup of repos by name.
Prints corresponding repo information if the name exists,
exits with ENOENT status otherwise.`,
	Run: func(cmd *cobra.Command, args []string) {
		var err error

		defer func(t0 time.Time) {
			cliUsage(t0, "repo get", err)
		}(time.Now())

		ctx := context.Background()
		datamonFlagsPtr := &datamonFlags
		optionInputs := newCliOptionInputs(config, datamonFlagsPtr)
		remoteStores, err := optionInputs.datamonContext(ctx, ReadOnlyContext())
		if err != nil {
			wrapFatalln("create remote stores", err)
			return
		}
		repoDescriptor, err := core.GetRepoDescriptorByRepoName(
			remoteStores, datamonFlags.repo.RepoName)
		if err != nil {
			if errors.Is(err, status.ErrNotFound) {
				wrapFatalWithCodef(int(unix.ENOENT), "didn't find repo %q", datamonFlags.repo.RepoName)
				return
			}
			wrapFatalln("error downloading repo information", err)
			return
		}

		var buf bytes.Buffer
		err = repoDescriptorTemplate(datamonFlags).Execute(&buf, repoDescriptor)
		if err != nil {
			wrapFatalln("executing template", err)
			return
		}
		log.Println(buf.String())

		if datamonFlags.repo.withSize {
			log.Println("All bundles with their total size:")
			// retrieve all files for all bundles and report about the size of each bundle
			var grandTotal uint64
			err = core.ListBundlesApply(datamonFlags.repo.RepoName, remoteStores,
				retrieveFileSizes(
					datamonFlags.repo.RepoName,
					remoteStores,
					&datamonFlags,
					optionInputs,
					&grandTotal,
				),
				core.ConcurrentList(datamonFlags.core.ConcurrencyFactor),
				core.BatchSize(datamonFlags.core.BatchSize),
				core.WithMetrics(datamonFlags.root.metrics.IsEnabled()),
			)
			if err != nil {
				wrapFatalln("concurrent list bundles", err)
				return
			}

			log.Printf("\nGrand total for repo %q: %d (%s)",
				datamonFlags.repo.RepoName,
				grandTotal,
				units.HumanSize(float64(grandTotal)),
			)
		}
	},
	PreRun: func(cmd *cobra.Command, args []string) {
		if err := newCliOptionInputs(config, &datamonFlags).populateRemoteConfig(); err != nil {
			wrapFatalln("populate remote config", err)
		}
	},
}

func init() {
	requireFlags(GetRepoCommand,
		addRepoNameOptionFlag(GetRepoCommand),
	)
	addSkipAuthFlag(GetRepoCommand)
	addRepoSizeFlag(GetRepoCommand)

	repoCmd.AddCommand(GetRepoCommand)
}

func retrieveFileSizes(repo string, stores context2.Stores, datamonFlags *flagsT, optionInputs *cliOptionInputs, grandTotal *uint64) func(model.BundleDescriptor) error {
	return func(b model.BundleDescriptor) error {
		ctx := context.Background()
		bundleOpts, err := optionInputs.bundleOpts(ctx, ReadOnlyContext())
		if err != nil {
			wrapFatalln("failed to initialize bundle options", err)
		}

		logger, err := optionInputs.getLogger()
		if err != nil {
			return err
		}
		bundleOpts = append(bundleOpts, core.Repo(repo))
		bundleOpts = append(bundleOpts, core.BundleID(b.ID))
		bundleOpts = append(bundleOpts, core.BundleWithMetrics(datamonFlags.root.metrics.IsEnabled()))
		bundleOpts = append(bundleOpts, core.Logger(logger))
		bundle := core.NewBundle(
			bundleOpts...,
		)

		err = core.PopulateFiles(context.Background(), bundle)
		if err != nil {
			return err
		}

		var bundleSize uint64
		for _, e := range bundle.BundleEntries {
			bundleSize += e.Size
		}
		atomic.AddUint64(grandTotal, bundleSize)

		var buf bytes.Buffer
		if err := bundleSizeTemplate(*datamonFlags).Execute(&buf, struct {
			model.BundleDescriptor
			Size      uint64
			HumanSize string
		}{
			BundleDescriptor: b,
			Size:             bundleSize,
			HumanSize:        units.HumanSize(float64(bundleSize)),
		}); err != nil {
			return err
		}
		log.Println(buf.String())

		return nil
	}
}

var bundleSizeTemplate func(flagsT) *template.Template

func init() {
	bundleSizeTemplate = func(opts flagsT) *template.Template {
		if opts.core.Template != "" {
			t, err := template.New("bundle size").Parse(datamonFlags.core.Template)
			if err != nil {
				wrapFatalln("invalid template", err)
			}
			return t
		}
		const bundleLineTemplateString = `{{.ID}} , {{.Message}}, {{.Timestamp}}, {{.Size}} ({{.HumanSize}})`
		return template.Must(template.New("list line").Parse(bundleLineTemplateString))
	}
}

/*
	LeafSize               uint32        `json:"leafSize" yaml:"leafSize"`                               // Bundles blobs are independently generated
	ID                     string        `json:"id" yaml:"id"`                                           // Unique ID for the bundle.
	Message                string        `json:"message" yaml:"message"`                                 // Message for the commit/bundle
	Parents                []string      `json:"parents,omitempty" yaml:"parents,omitempty"`             // Bundles with parent child relation
	Timestamp              time.Time     `json:"timestamp,omitempty" yaml:"timestamp,omitempty"`         // Local wall clock time
	Contributors           []Contributor `json:"contributors" yaml:"contributors"`                       // Contributor for the bundle
	BundleEntriesFileCount uint64        `json:"count" yaml:"count"`                                     // Number of file index files in this bundle
	Version                uint64        `json:"version,omitempty" yaml:"version,omitempty"`             // Version for the metadata model used for this bundle
	Deduplication          string        `json:"deduplication,omitempty" yaml:"deduplication,omitempty"` // Deduplication scheme used
	RunStage               string        `json:"runstage,omitempty" yaml:"runstage,omitempty"`           // Path to the run stage
*/
