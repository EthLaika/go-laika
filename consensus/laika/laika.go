// Package laika implements the laika proof-of-capacity algorithm
package laika

import "errors"

// Various error messages to mark blocks invalid. These should be private to
// prevent engine specific errors from being referenced in the remainder of the
// codebase, inherently breaking if the engine is swapped out. Please put common
// error types into the consensus package.
var (
	// errInvalidPoC is thrown if the verification of the PoC fails
	errInvalidPoC = errors.New("invalid poc")
)
