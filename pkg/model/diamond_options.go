package model

// DiamondDescriptorOption defines an option to build a DiamondDescriptor
type DiamondDescriptorOption func(*DiamondDescriptor)

// SplitDescriptorOption defines an option to build a SplitDescriptor
type SplitDescriptorOption func(*SplitDescriptor)

// DiamondID sets the DiamondID of a DiamondDescriptor
func DiamondID(id string) DiamondDescriptorOption {
	return func(d *DiamondDescriptor) {
		if id != "" {
			d.DiamondID = id
		}
	}
}

// DiamondMode sets the conflict resolution mode for a diamond
func DiamondMode(mode ConflictMode) DiamondDescriptorOption {
	return func(d *DiamondDescriptor) {
		d.Mode = mode
	}
}

// DiamondTag sets an informative tag on the diamond
func DiamondTag(tag string) DiamondDescriptorOption {
	return func(d *DiamondDescriptor) {
		d.Tag = tag
	}
}

// DiamondClone clones from a DiamondDescriptor
func DiamondClone(m DiamondDescriptor) DiamondDescriptorOption {
	return func(d *DiamondDescriptor) {
		*d = m
	}
}

// SplitID sets the splitID of a SplitDescriptor
func SplitID(id string) SplitDescriptorOption {
	return func(s *SplitDescriptor) {
		if id != "" {
			s.SplitID = id
		}
	}
}

// SplitContributors defines the list of contributors for a SplitDescriptor
func SplitContributors(c []Contributor) SplitDescriptorOption {
	return func(s *SplitDescriptor) {
		s.Contributors = c
	}
}

// SplitContributor defines a single contributor for a SplitDescriptor
func SplitContributor(c Contributor) SplitDescriptorOption {
	return SplitContributors([]Contributor{c})
}

// SplitTag sets an informative tag on the split
func SplitTag(tag string) SplitDescriptorOption {
	return func(s *SplitDescriptor) {
		s.Tag = tag
	}
}

// SplitClone clones from a SplitDescriptor
func SplitClone(m SplitDescriptor) SplitDescriptorOption {
	return func(s *SplitDescriptor) {
		*s = m
	}
}
