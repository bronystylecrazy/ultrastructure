// Package di provides a small declarative layer on top of Fx for building apps.
//
// The package models apps as nodes (Provide, Invoke, Module, etc.) that can be
// composed, planned, and built into Fx options. It also includes helpers for:
//   - conditional wiring (If/When/Switch)
//   - replacements/defaults
//   - configuration loading (Viper)
//   - diagnostics and lifecycle hooks
//   - auto-grouping of providers
//
// See the examples directory for usage patterns.
package di
