package assert_test

import (
	"fmt"
	"strings"
	"testing"

	"github.com/ironsmile/euterpe/src/assert"
	"github.com/ironsmile/euterpe/src/assert/assertfakes"
)

// TestEqual makes sure that the Equal function works for various types of arguments.
func TestEqual(t *testing.T) {
	fakeT := &assertfakes.FakeTestingErrf{}
	actual := int64(5)
	assert.Equal(fakeT, 5, actual)
	if fakeT.ErrorfCallCount() != 0 {
		t.Errorf("expected Errorf not to be called for int64 and const expression")
	}
	if fakeT.HelperCallCount() != 1 {
		t.Errorf("expected Helper() to be called on the testing type")
	}

	assert.Equal(fakeT, 10, actual)
	if fakeT.ErrorfCallCount() != 1 {
		t.Errorf("expected Errorf to be called for different int64 values")
	}

	fakeT = &assertfakes.FakeTestingErrf{}
	var (
		actualStr   string = "test val"
		expectedStr string = "test val"
	)
	assert.Equal(fakeT, expectedStr, actualStr)
	if fakeT.ErrorfCallCount() != 0 {
		t.Errorf("expected Errorf not to be called for two string values")
	}

	const (
		formatting   = `test formatting: %d`
		formattedVal = 123
	)
	fakeT = &assertfakes.FakeTestingErrf{}
	assert.Equal(fakeT, 10, 12, formatting, formattedVal)
	if fakeT.ErrorfCallCount() != 1 {
		t.Errorf("expected Errorf to be called for two different integers")
	}

	expectedMessage := fmt.Sprintf(formatting, formattedVal)
	errorFormat, args := fakeT.ErrorfArgsForCall(0)

	loggedMessage := fmt.Sprintf(errorFormat, args...)
	if !strings.Contains(loggedMessage, expectedMessage) {
		t.Errorf("message `%s` was not part of format: `%s`", loggedMessage, errorFormat)
	}

	if len(args) < 2 {
		t.Errorf("expected at least the two values tested to be used in error")
	}

	usedExpected, ok := args[0].(int)
	if !ok {
		t.Errorf("expected first format value to be int but it was %T", args[0])
	}
	if usedExpected != 10 {
		t.Errorf("expected first format value to be %d but it was %d", 10, usedExpected)
	}

	usedActual, ok := args[1].(int)
	if !ok {
		t.Errorf("expected second format value to be int but it was %T", args[1])
	}
	if usedActual != 12 {
		t.Errorf("expected second format value to be %d but it was %d", 12, usedActual)
	}
}

// TestEqualPanicsOnWrongArgs makes sure that Equal panics when the first argument
// after expected and actual is not a string.
func TestEqualPanicsOnWrongArgs(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Errorf("expected the test to panic because of wrong arguments")
		}
	}()

	fakeT := &assertfakes.FakeTestingErrf{}
	assert.Equal(fakeT, 5, 12, 123, "baba")
}
