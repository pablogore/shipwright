// Package config contiene la configuración de la aplicación Shipwright.
package config

import "errors"

// ErrNoPort when a port number for the application is not set
var ErrNoPort = Error{errors.New("PORT is not defined")}

// Error is the config package's error type if an environment variable is missing
type Error struct {
	Err error
}

// Error ...
func (e Error) Error() string { return "Environment variable: " + e.Err.Error() }

// Unwrap returns the underlying error for go error catching logic
func (e Error) Unwrap() error { return e.Err }
