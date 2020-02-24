# Add upload scheduler to webservice

This is a proposal to facet `bundle upload` via the Datamon webservice (i.e. the `web` command).

Additionally, so that the web service -- currently a frontend for `repo list`, `bundle list`, and `bundle list files`, all operations that typically complete in per-HTTP-request goroutines in about the expected timeframe for page loads in a 1.0-ish webservice -- can continue to be useful for a possible deploy to Kubernetes, faceting `bundle upload` necessarily includes implementing some at least rudimentary scheduling functionality:  The webservice will need to keep track of several ongoing uploads, each triggered by an HTTP-request, but with fire-and-forget (i.e. event) semantics rather than those of request-response (i.e. RPC).

### data sources

This proposal calls for an additional, optional flag to the `web` command, `--upload-source`, say, which can be present in arbitrary arities (to allow arbitrary numbers of upload sources -- is this possible with pflags?).  When this flag isn't present, the webservice will continue to function as before.  When a non-empty list of directories is passed in via this flag, an additional page will be available in the web ui that, at first, provides a simple file browser rooted at each of these directories.

From within any navigable directory of the filebrowser-as-a-webservice, it will additionally be possible to click an "upload" button to schedule a `bundle upload` of the tree rooted at that directory.  The "upload" button will be attached to a form faceting a few additional `bundle upload` parameters:

* `--concurrency-factor`
* `--message`
* `--repo` (dropdown)
* `--label`

### scheduler

Round-robin scheduling with a heartbeat
[as described here](https://github.com/ransomw/tubing/blob/master/src/core.clj)
is expected to be as complicated as a first pass of the scheduler internals will be.  That is to say, every effort will be made to decomplect scheduler design at the possible expense of performance.

In addition to the filebrowser views described in the data sources section, the web UI will provide a flat list of all uploads according to source directory and destination description (repo, message, label), including simple status (complete, error, ongoing).

## nice-to-haves

### downloadable logs

The web UI will provide a flat list of all uploads, including logging  the output from `pkg/core` and beyond for download.

### build filelists

currently, bundle structure is expected to be described entirely by filesystem layout.
