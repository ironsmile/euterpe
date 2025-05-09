package assert_test

import (
	"io"
	"testing"

	"github.com/ironsmile/euterpe/src/assert"
	"github.com/ironsmile/euterpe/src/assert/assertfakes"
)

// TestNilErr makes sure that ErrNil works as expected.
func TestNilErr(t *testing.T) {
	var nilErr error

	fakeTf := &assertfakes.FakeTestingFatalf{}
	assert.NilErr(fakeTf, nilErr)
	if fakeTf.FatalfCallCount() != 0 {
		t.Fatalf("unexpected Fatalf() call for nil error")
	}
	if fakeTf.HelperCallCount() != 1 {
		t.Fatalf("testing.T.Helper() not called")
	}

	assert.NilErr(fakeTf, io.EOF)
	if fakeTf.FatalfCallCount() != 1 {
		t.Fatalf("expected Fatalf() to be called but it was not")
	}
}

// TestNotNilErr makes sure that ErrNotNil works as expected.
func TestNotNilErr(t *testing.T) {

	fakeTf := &assertfakes.FakeTestingFatalf{}
	assert.NotNilErr(fakeTf, io.EOF)
	if fakeTf.FatalfCallCount() != 0 {
		t.Fatalf("unexpected Fatalf() call for nil error")
	}
	if fakeTf.HelperCallCount() != 1 {
		t.Fatalf("testing.T.Helper() not called")
	}

	var nilErr error
	assert.NotNilErr(fakeTf, nilErr)
	if fakeTf.FatalfCallCount() != 1 {
		t.Fatalf("expected Fatalf() to be called but it was not")
	}
}
