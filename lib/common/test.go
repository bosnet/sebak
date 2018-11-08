// Provide test utilities for the common package
package common

// Initialize a new config object for unittests
func NewTestConfig() Config {
	return NewConfig([]byte("sebak-unittest"))
}
