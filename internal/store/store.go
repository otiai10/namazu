package store

// Store defines the base interface for data store operations
type Store interface {
	// Close releases any resources held by the store
	Close() error
}
