package assert

// NilErr checks that `val` is nil. Causes a fatal error otherwise.
func NilErr(t TestingFatalf, val error, msgAndArgs ...any) {
	t.Helper()

	if val == nil {
		return
	}

	t.Fatalf("expected nil but got `%#v`%s", val, fromMsgAndArgs(msgAndArgs...))
}

// NotNilErr checks that `val` is not nil. Causes a fatal error otherwise.
func NotNilErr(t TestingFatalf, val error, msgAndArgs ...any) {
	t.Helper()

	if val != nil {
		return
	}

	t.Fatalf("unexpected nil%s", fromMsgAndArgs(msgAndArgs...))
}
