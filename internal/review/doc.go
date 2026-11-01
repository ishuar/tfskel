// Package review owns the `tfskel review plan` flow: parse → analyze →
// filter → format → optional AI narrative → exit-code derivation.
//
// The module's central invariant: once the filter is applied, the analysis is
// the single source of truth for everything downstream — the formatted report
// and the AI payload both see exactly the filtered resource set. The AI step
// is additive: any AI failure warns via the logger but never surfaces as an
// error and never affects the exit code.
package review
