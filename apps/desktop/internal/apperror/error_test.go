package apperror

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"
	"testing"
)

func TestPublicErrorContract(t *testing.T) {
	data, err := os.ReadFile("../../../../examples/contracts/app-errors.json")
	if err != nil {
		t.Fatal(err)
	}
	var cases []Payload
	if err = json.Unmarshal(data, &cases); err != nil {
		t.Fatal(err)
	}
	for _, want := range cases {
		t.Run(string(want.Code), func(t *testing.T) {
			original := New(want.Code, want.Message)
			var got Payload
			if err := json.Unmarshal(Marshal(original), &got); err != nil {
				t.Fatal(err)
			}
			if got != want {
				t.Fatalf("got %+v, want %+v", got, want)
			}
			var fields map[string]json.RawMessage
			if err := json.Unmarshal(Marshal(original), &fields); err != nil {
				t.Fatal(err)
			}
			if len(fields) != 2 || fields["code"] == nil || fields["message"] == nil {
				t.Fatalf("unexpected public fields: %s", Marshal(original))
			}
			if original.Error() != want.Message {
				t.Fatal("outer Wails message differs from payload")
			}
		})
	}
}

func TestInternalDetailsNeverCrossPublicBoundary(t *testing.T) {
	public := New(Unauthenticated, "Войдите снова.")
	wrapped := fmt.Errorf("SQL/token secret detail: %w", public)
	if safe := Public(wrapped); safe != public || strings.Contains(safe.Error(), "secret") || !errors.Is(safe, public) {
		t.Fatal("wrapper was not removed")
	}
	var typedNil *Error
	for _, err := range []error{errors.New("password=private /path/to/database"), &Error{}, typedNil} {
		if got := Describe(err); got.Code != InternalError || strings.Contains(got.Message, "private") {
			t.Fatal(got)
		}
		if strings.Contains(string(Marshal(err)), "private") {
			t.Fatal("raw details in JSON")
		}
	}
	if Public(nil) != nil || string(Marshal(nil)) != "null" {
		t.Fatal("success turned into an error")
	}
	if got := Describe(New(Code("UNKNOWN"), "secret")); got.Code != InternalError || strings.Contains(got.Message, "secret") {
		t.Fatal(got)
	}
}
