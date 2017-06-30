// Package build is a helper package for building SSA IR in the parent
// directory.
//
// Usage
//
// There are two ways of building SSA IR from source code:
//
// Build from a list of source files
//
// This is the normal usage, where a number of files are supplied (usually as
// command line arguments), and the builder tool considers all of the files part
// of the same package (i.e. in the same directory).
//
// Build from a Reader
//
// This is mostly used for testing or demo, where the input source code is read
// from a given io.Reader and written to a temporary file, which will be removed
// straight after the build is complete.
//
package build
