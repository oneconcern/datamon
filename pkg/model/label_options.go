package model

// LabelDescriptorOption is a functor to build label descriptors
type LabelDescriptorOption func(descriptor *LabelDescriptor)

// LabelContributors sets a list of contributors for the label
func LabelContributors(c []Contributor) LabelDescriptorOption {
	return func(ld *LabelDescriptor) {
		ld.Contributors = c
	}
}

// LabelContributor sets a single contributor for the label
func LabelContributor(c Contributor) LabelDescriptorOption {
	return LabelContributors([]Contributor{c})
}

// LabelName sets a name for the label
func LabelName(name string) LabelDescriptorOption {
	return func(ld *LabelDescriptor) {
		ld.Name = name
	}
}
