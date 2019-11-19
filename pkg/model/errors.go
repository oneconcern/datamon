package model

// ConsumableStorePathMetadataErr defines errors related to consumable store metadata
type ConsumableStorePathMetadataErr struct {
	msg string
}

func (e ConsumableStorePathMetadataErr) Error() string {
	return e.msg
}
