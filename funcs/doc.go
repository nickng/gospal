// Package funcs is a wrapper for functions to create a uniform representation
// for function-like constructs in Go.
//
// We define representation of Definition (i.e. callee perspective) and Instance
// (i.e. caller perspective) for:
//
//  - Builtin function
//  - Ordinary function
//  - Pointer to function (function in a variable)
//  - Closures (functions that carries variable with it)
//
// Definitions created by this package can be used as store.Value.
package funcs
