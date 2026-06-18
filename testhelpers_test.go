package postmark

// boolPtr is a test helper that returns a pointer to a bool value.
// It is defined here (rather than in a specific test file) so that it is
// available to all test files in the package without duplication.
func boolPtr(b bool) *bool { return &b }
