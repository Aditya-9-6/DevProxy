package pool

// PinToCore is deprecated. Direct OS thread pinning is discouraged in Go's scheduler.
// Use context-aware sharding instead.
func PinToCore(coreID int) {}

// Unpin is deprecated.
func Unpin() {}
