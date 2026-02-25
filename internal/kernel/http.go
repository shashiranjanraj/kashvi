// Package kernel provides the framework's default HTTP kernel.
//
// This package is INTERNAL to the Kashvi framework.
// External users should NOT import this package directly.
//
// For project-level HTTP configuration, use pkg/app:
//
//	app.New().Routes(func(r *router.Router) { ... }).Run()
package kernel

// NOTE: This file is intentionally kept minimal.
// The actual kernel logic has moved to pkg/app/kernel.go
// so it can accept user-provided route registrations without
// importing project-specific code.
//
// This file is retained as a placeholder to avoid breaking
// any internal references that might import this package.
