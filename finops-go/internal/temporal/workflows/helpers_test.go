package workflows_test

import "github.com/stretchr/testify/mock"

// Matchers for activity mocks -- match any context and any input.
var (
	testAnyCtx   = mock.Anything
	testAnyInput = mock.Anything
)
