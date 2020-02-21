package model

// BundleDescriptorOption is a functor to build a bundle descriptor with some options
type BundleDescriptorOption func(descriptor *BundleDescriptor)

// Message defines the message of the bundle descriptor
func Message(m string) BundleDescriptorOption {
	return func(b *BundleDescriptor) {
		b.Message = m
	}
}

// BundleContributors defines the list of contributors for a bundle descriptor
func BundleContributors(c []Contributor) BundleDescriptorOption {
	return func(b *BundleDescriptor) {
		b.Contributors = c
	}
}

// BundleContributor defines a single contributor for a bundle descriptor
func BundleContributor(c Contributor) BundleDescriptorOption {
	return BundleContributors([]Contributor{c})
}

// Parents defines the parents for a bundle descriptor
func Parents(p []string) BundleDescriptorOption {
	return func(b *BundleDescriptor) {
		b.Parents = p
	}
}

// Deduplication defines the deduplication scheme for a bundle descriptor
func Deduplication(d string) BundleDescriptorOption {
	return func(b *BundleDescriptor) {
		b.Deduplication = d
	}
}
