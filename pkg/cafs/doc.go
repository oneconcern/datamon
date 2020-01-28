// Package cafs provides a content-addressable file system.
//
// All content is indexed according to a deduplication scheme,
// e.g. Blake hash.
//
// A cafs file system stores files as blobs identified by their hash.
// Blobs are organized in leaves of a fixed size.
//
// Each leaf is stored on the backend store using the hash as an object reference to the storage resource.
//
// A root key is computed from the hashes of all leaves.
//
// Each file is thus associated with its root key object on the backend, which stores the keys of all leaves.
//
// The default leaf size is 2MB.
// When used to store bundles, the leaf size is part of the bundle specification.
package cafs
