// Package loop provides utilities for loop representation and detection.
//
// Loop detection works by inspecting the SSA block comments to determine the
// state of loop, and tracks the transition between the states. Combining the
// state information and instructions encountered, the loop parameters (e.g.
// index variable, bounds, increment) are extracted if possible.
//
// A condition graph is built from the transition between short-circuited loop
// conditions so to reconstruct the loop conditions.
package loop
