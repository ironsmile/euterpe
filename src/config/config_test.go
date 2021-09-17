package config_test

import (
	"bytes"
	"encoding/json"
	"testing"
	"time"

	"github.com/ironsmile/euterpe/src/config"
)

// TestScanSectionUnmarshalJSON makes sure that decoding the JSON for the "library_scan"
// configuration key is working as expected.
func TestScanSectionUnmarshalJSON(t *testing.T) {
	jsonBuff := bytes.NewBufferString(`
		{
			"disable": false,
			"files_per_operation": 100,
			"sleep_after_operation": "15ms",
			"initial_wait_duration": "100ms"
		}
	`)

	var ss config.ScanSection
	dec := json.NewDecoder(jsonBuff)
	err := dec.Decode(&ss)
	if err != nil {
		t.Fatalf("decoding ScanSection JSON failed: %s", err)
	}

	expected := config.ScanSection{
		Disable:           false,
		FilesPerOperation: 100,
		SleepPerOperation: 15 * time.Millisecond,
		InitialWait:       100 * time.Millisecond,
	}

	if ss != expected {
		t.Errorf("expected `%+v` but got `%+v`", expected, ss)
	}
}
