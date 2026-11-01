// Package strutil holds the small string transformations shared by the
// generators and the CLI: region abbreviation (eu-central-1 becomes euc1),
// app directory name normalization (spaces to hyphens), and trimmed
// non-empty input validation with a typed EmptyFieldError.
//
// It stays dependency-free and free of tfskel domain types. Anything that
// needs config or templates belongs in the package that owns that concept.
package strutil
