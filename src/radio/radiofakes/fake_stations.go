// Code generated by counterfeiter. DO NOT EDIT.
package radiofakes

import (
	"context"
	"sync"

	"github.com/ironsmile/euterpe/src/radio"
)

type FakeStations struct {
	CreateStub        func(context.Context, radio.Station) (int64, error)
	createMutex       sync.RWMutex
	createArgsForCall []struct {
		arg1 context.Context
		arg2 radio.Station
	}
	createReturns struct {
		result1 int64
		result2 error
	}
	createReturnsOnCall map[int]struct {
		result1 int64
		result2 error
	}
	DeleteStub        func(context.Context, int64) error
	deleteMutex       sync.RWMutex
	deleteArgsForCall []struct {
		arg1 context.Context
		arg2 int64
	}
	deleteReturns struct {
		result1 error
	}
	deleteReturnsOnCall map[int]struct {
		result1 error
	}
	GetAllStub        func(context.Context) ([]radio.Station, error)
	getAllMutex       sync.RWMutex
	getAllArgsForCall []struct {
		arg1 context.Context
	}
	getAllReturns struct {
		result1 []radio.Station
		result2 error
	}
	getAllReturnsOnCall map[int]struct {
		result1 []radio.Station
		result2 error
	}
	ReplaceStub        func(context.Context, radio.Station) error
	replaceMutex       sync.RWMutex
	replaceArgsForCall []struct {
		arg1 context.Context
		arg2 radio.Station
	}
	replaceReturns struct {
		result1 error
	}
	replaceReturnsOnCall map[int]struct {
		result1 error
	}
	invocations      map[string][][]interface{}
	invocationsMutex sync.RWMutex
}

func (fake *FakeStations) Create(arg1 context.Context, arg2 radio.Station) (int64, error) {
	fake.createMutex.Lock()
	ret, specificReturn := fake.createReturnsOnCall[len(fake.createArgsForCall)]
	fake.createArgsForCall = append(fake.createArgsForCall, struct {
		arg1 context.Context
		arg2 radio.Station
	}{arg1, arg2})
	stub := fake.CreateStub
	fakeReturns := fake.createReturns
	fake.recordInvocation("Create", []interface{}{arg1, arg2})
	fake.createMutex.Unlock()
	if stub != nil {
		return stub(arg1, arg2)
	}
	if specificReturn {
		return ret.result1, ret.result2
	}
	return fakeReturns.result1, fakeReturns.result2
}

func (fake *FakeStations) CreateCallCount() int {
	fake.createMutex.RLock()
	defer fake.createMutex.RUnlock()
	return len(fake.createArgsForCall)
}

func (fake *FakeStations) CreateCalls(stub func(context.Context, radio.Station) (int64, error)) {
	fake.createMutex.Lock()
	defer fake.createMutex.Unlock()
	fake.CreateStub = stub
}

func (fake *FakeStations) CreateArgsForCall(i int) (context.Context, radio.Station) {
	fake.createMutex.RLock()
	defer fake.createMutex.RUnlock()
	argsForCall := fake.createArgsForCall[i]
	return argsForCall.arg1, argsForCall.arg2
}

func (fake *FakeStations) CreateReturns(result1 int64, result2 error) {
	fake.createMutex.Lock()
	defer fake.createMutex.Unlock()
	fake.CreateStub = nil
	fake.createReturns = struct {
		result1 int64
		result2 error
	}{result1, result2}
}

func (fake *FakeStations) CreateReturnsOnCall(i int, result1 int64, result2 error) {
	fake.createMutex.Lock()
	defer fake.createMutex.Unlock()
	fake.CreateStub = nil
	if fake.createReturnsOnCall == nil {
		fake.createReturnsOnCall = make(map[int]struct {
			result1 int64
			result2 error
		})
	}
	fake.createReturnsOnCall[i] = struct {
		result1 int64
		result2 error
	}{result1, result2}
}

func (fake *FakeStations) Delete(arg1 context.Context, arg2 int64) error {
	fake.deleteMutex.Lock()
	ret, specificReturn := fake.deleteReturnsOnCall[len(fake.deleteArgsForCall)]
	fake.deleteArgsForCall = append(fake.deleteArgsForCall, struct {
		arg1 context.Context
		arg2 int64
	}{arg1, arg2})
	stub := fake.DeleteStub
	fakeReturns := fake.deleteReturns
	fake.recordInvocation("Delete", []interface{}{arg1, arg2})
	fake.deleteMutex.Unlock()
	if stub != nil {
		return stub(arg1, arg2)
	}
	if specificReturn {
		return ret.result1
	}
	return fakeReturns.result1
}

func (fake *FakeStations) DeleteCallCount() int {
	fake.deleteMutex.RLock()
	defer fake.deleteMutex.RUnlock()
	return len(fake.deleteArgsForCall)
}

func (fake *FakeStations) DeleteCalls(stub func(context.Context, int64) error) {
	fake.deleteMutex.Lock()
	defer fake.deleteMutex.Unlock()
	fake.DeleteStub = stub
}

func (fake *FakeStations) DeleteArgsForCall(i int) (context.Context, int64) {
	fake.deleteMutex.RLock()
	defer fake.deleteMutex.RUnlock()
	argsForCall := fake.deleteArgsForCall[i]
	return argsForCall.arg1, argsForCall.arg2
}

func (fake *FakeStations) DeleteReturns(result1 error) {
	fake.deleteMutex.Lock()
	defer fake.deleteMutex.Unlock()
	fake.DeleteStub = nil
	fake.deleteReturns = struct {
		result1 error
	}{result1}
}

func (fake *FakeStations) DeleteReturnsOnCall(i int, result1 error) {
	fake.deleteMutex.Lock()
	defer fake.deleteMutex.Unlock()
	fake.DeleteStub = nil
	if fake.deleteReturnsOnCall == nil {
		fake.deleteReturnsOnCall = make(map[int]struct {
			result1 error
		})
	}
	fake.deleteReturnsOnCall[i] = struct {
		result1 error
	}{result1}
}

func (fake *FakeStations) GetAll(arg1 context.Context) ([]radio.Station, error) {
	fake.getAllMutex.Lock()
	ret, specificReturn := fake.getAllReturnsOnCall[len(fake.getAllArgsForCall)]
	fake.getAllArgsForCall = append(fake.getAllArgsForCall, struct {
		arg1 context.Context
	}{arg1})
	stub := fake.GetAllStub
	fakeReturns := fake.getAllReturns
	fake.recordInvocation("GetAll", []interface{}{arg1})
	fake.getAllMutex.Unlock()
	if stub != nil {
		return stub(arg1)
	}
	if specificReturn {
		return ret.result1, ret.result2
	}
	return fakeReturns.result1, fakeReturns.result2
}

func (fake *FakeStations) GetAllCallCount() int {
	fake.getAllMutex.RLock()
	defer fake.getAllMutex.RUnlock()
	return len(fake.getAllArgsForCall)
}

func (fake *FakeStations) GetAllCalls(stub func(context.Context) ([]radio.Station, error)) {
	fake.getAllMutex.Lock()
	defer fake.getAllMutex.Unlock()
	fake.GetAllStub = stub
}

func (fake *FakeStations) GetAllArgsForCall(i int) context.Context {
	fake.getAllMutex.RLock()
	defer fake.getAllMutex.RUnlock()
	argsForCall := fake.getAllArgsForCall[i]
	return argsForCall.arg1
}

func (fake *FakeStations) GetAllReturns(result1 []radio.Station, result2 error) {
	fake.getAllMutex.Lock()
	defer fake.getAllMutex.Unlock()
	fake.GetAllStub = nil
	fake.getAllReturns = struct {
		result1 []radio.Station
		result2 error
	}{result1, result2}
}

func (fake *FakeStations) GetAllReturnsOnCall(i int, result1 []radio.Station, result2 error) {
	fake.getAllMutex.Lock()
	defer fake.getAllMutex.Unlock()
	fake.GetAllStub = nil
	if fake.getAllReturnsOnCall == nil {
		fake.getAllReturnsOnCall = make(map[int]struct {
			result1 []radio.Station
			result2 error
		})
	}
	fake.getAllReturnsOnCall[i] = struct {
		result1 []radio.Station
		result2 error
	}{result1, result2}
}

func (fake *FakeStations) Replace(arg1 context.Context, arg2 radio.Station) error {
	fake.replaceMutex.Lock()
	ret, specificReturn := fake.replaceReturnsOnCall[len(fake.replaceArgsForCall)]
	fake.replaceArgsForCall = append(fake.replaceArgsForCall, struct {
		arg1 context.Context
		arg2 radio.Station
	}{arg1, arg2})
	stub := fake.ReplaceStub
	fakeReturns := fake.replaceReturns
	fake.recordInvocation("Replace", []interface{}{arg1, arg2})
	fake.replaceMutex.Unlock()
	if stub != nil {
		return stub(arg1, arg2)
	}
	if specificReturn {
		return ret.result1
	}
	return fakeReturns.result1
}

func (fake *FakeStations) ReplaceCallCount() int {
	fake.replaceMutex.RLock()
	defer fake.replaceMutex.RUnlock()
	return len(fake.replaceArgsForCall)
}

func (fake *FakeStations) ReplaceCalls(stub func(context.Context, radio.Station) error) {
	fake.replaceMutex.Lock()
	defer fake.replaceMutex.Unlock()
	fake.ReplaceStub = stub
}

func (fake *FakeStations) ReplaceArgsForCall(i int) (context.Context, radio.Station) {
	fake.replaceMutex.RLock()
	defer fake.replaceMutex.RUnlock()
	argsForCall := fake.replaceArgsForCall[i]
	return argsForCall.arg1, argsForCall.arg2
}

func (fake *FakeStations) ReplaceReturns(result1 error) {
	fake.replaceMutex.Lock()
	defer fake.replaceMutex.Unlock()
	fake.ReplaceStub = nil
	fake.replaceReturns = struct {
		result1 error
	}{result1}
}

func (fake *FakeStations) ReplaceReturnsOnCall(i int, result1 error) {
	fake.replaceMutex.Lock()
	defer fake.replaceMutex.Unlock()
	fake.ReplaceStub = nil
	if fake.replaceReturnsOnCall == nil {
		fake.replaceReturnsOnCall = make(map[int]struct {
			result1 error
		})
	}
	fake.replaceReturnsOnCall[i] = struct {
		result1 error
	}{result1}
}

func (fake *FakeStations) Invocations() map[string][][]interface{} {
	fake.invocationsMutex.RLock()
	defer fake.invocationsMutex.RUnlock()
	fake.createMutex.RLock()
	defer fake.createMutex.RUnlock()
	fake.deleteMutex.RLock()
	defer fake.deleteMutex.RUnlock()
	fake.getAllMutex.RLock()
	defer fake.getAllMutex.RUnlock()
	fake.replaceMutex.RLock()
	defer fake.replaceMutex.RUnlock()
	copiedInvocations := map[string][][]interface{}{}
	for key, value := range fake.invocations {
		copiedInvocations[key] = value
	}
	return copiedInvocations
}

func (fake *FakeStations) recordInvocation(key string, args []interface{}) {
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

var _ radio.Stations = new(FakeStations)
