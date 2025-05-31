package assert

// Equal checks whether expected and actual are actually equal and fails the test
// if they are not.
func Equal[V comparable](t TestingFatalf, expected, actual V, msgAndArgs ...any) {
	t.Helper()

	if expected == actual {
		return
	}

	t.Fatalf("not equal: expected `%#v` but got `%#v`%s",
		expected, actual, fromMsgAndArgs(msgAndArgs...),
	)
}
