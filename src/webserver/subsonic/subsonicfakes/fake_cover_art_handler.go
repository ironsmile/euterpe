// Code generated by counterfeiter. DO NOT EDIT.
package subsonicfakes

import (
	"net/http"
	"sync"

	"github.com/ironsmile/euterpe/src/webserver/subsonic"
)

type FakeCoverArtHandler struct {
	FindStub        func(http.ResponseWriter, *http.Request, int64) error
	findMutex       sync.RWMutex
	findArgsForCall []struct {
		arg1 http.ResponseWriter
		arg2 *http.Request
		arg3 int64
	}
	findReturns struct {
		result1 error
	}
	findReturnsOnCall map[int]struct {
		result1 error
	}
	invocations      map[string][][]interface{}
	invocationsMutex sync.RWMutex
}

func (fake *FakeCoverArtHandler) Find(arg1 http.ResponseWriter, arg2 *http.Request, arg3 int64) error {
	fake.findMutex.Lock()
	ret, specificReturn := fake.findReturnsOnCall[len(fake.findArgsForCall)]
	fake.findArgsForCall = append(fake.findArgsForCall, struct {
		arg1 http.ResponseWriter
		arg2 *http.Request
		arg3 int64
	}{arg1, arg2, arg3})
	stub := fake.FindStub
	fakeReturns := fake.findReturns
	fake.recordInvocation("Find", []interface{}{arg1, arg2, arg3})
	fake.findMutex.Unlock()
	if stub != nil {
		return stub(arg1, arg2, arg3)
	}
	if specificReturn {
		return ret.result1
	}
	return fakeReturns.result1
}

func (fake *FakeCoverArtHandler) FindCallCount() int {
	fake.findMutex.RLock()
	defer fake.findMutex.RUnlock()
	return len(fake.findArgsForCall)
}

func (fake *FakeCoverArtHandler) FindCalls(stub func(http.ResponseWriter, *http.Request, int64) error) {
	fake.findMutex.Lock()
	defer fake.findMutex.Unlock()
	fake.FindStub = stub
}

func (fake *FakeCoverArtHandler) FindArgsForCall(i int) (http.ResponseWriter, *http.Request, int64) {
	fake.findMutex.RLock()
	defer fake.findMutex.RUnlock()
	argsForCall := fake.findArgsForCall[i]
	return argsForCall.arg1, argsForCall.arg2, argsForCall.arg3
}

func (fake *FakeCoverArtHandler) FindReturns(result1 error) {
	fake.findMutex.Lock()
	defer fake.findMutex.Unlock()
	fake.FindStub = nil
	fake.findReturns = struct {
		result1 error
	}{result1}
}

func (fake *FakeCoverArtHandler) FindReturnsOnCall(i int, result1 error) {
	fake.findMutex.Lock()
	defer fake.findMutex.Unlock()
	fake.FindStub = nil
	if fake.findReturnsOnCall == nil {
		fake.findReturnsOnCall = make(map[int]struct {
			result1 error
		})
	}
	fake.findReturnsOnCall[i] = struct {
		result1 error
	}{result1}
}

func (fake *FakeCoverArtHandler) Invocations() map[string][][]interface{} {
	fake.invocationsMutex.RLock()
	defer fake.invocationsMutex.RUnlock()
	fake.findMutex.RLock()
	defer fake.findMutex.RUnlock()
	copiedInvocations := map[string][][]interface{}{}
	for key, value := range fake.invocations {
		copiedInvocations[key] = value
	}
	return copiedInvocations
}

func (fake *FakeCoverArtHandler) recordInvocation(key string, args []interface{}) {
	fake.invocationsMutex.Lock()
	defer fake.invocationsMutex.Unlock()
	if fake.invocations == nil {
		fake.invocations = map[string][][]interface{}{}
	}
	if fake.invocations[key] == nil {
		fake.invocations[key] = [][]interface{}{}
	}
	fake.invocations[key] = append(fake.invocations[key], args)
}

var _ subsonic.CoverArtHandler = new(FakeCoverArtHandler)
