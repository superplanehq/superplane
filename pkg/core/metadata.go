package core

/*
 * Metadata is arbitrary data that can be stored for nodes and executions.
 * MetadataReader allows components to read metadata from a node or execution.
 */
type MetadataReader interface {
	Get() any
}

/*
 * MetadataWriter allows components to write metadata to a node or execution.
 * It inherits from MetadataReader to allow components to read metadata as well.
 */
type MetadataWriter interface {
	MetadataReader
	Set(any) error
}
