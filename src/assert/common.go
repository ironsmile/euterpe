package assert

import "fmt"

//go:generate go run github.com/maxbrunsfeld/counterfeiter/v6 -generate

//counterfeiter:generate . TestingFatalf

// TestingFatalf is an which supports reporting fatal errors in testing types such as
// testing.T, testing.TB and similar.
type TestingFatalf interface {
	Fatalf(format string, args ...any)
	Helper()
}

func fromMsgAndArgs(msgAndArgs ...any) string {
	if len(msgAndArgs) == 0 {
		return ""
	}

	fmtStr, ok := msgAndArgs[0].(string)
	if !ok {
		panic("The first argument in msgAndArgs must be a string format value.")
	}

	return fmt.Sprintf(" ("+fmtStr+")", msgAndArgs[1:]...)
}
