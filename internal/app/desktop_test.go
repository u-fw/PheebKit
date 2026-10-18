package app

import (
	"reflect"
	"testing"
)

func TestDesktopExposesReviewedIPCOnly(t *testing.T) {
	typ := reflect.TypeOf(&Desktop{})
	for _, name := range []string{"Attach", "Close", "CanClose", "AssetHandler"} {
		if _, ok := typ.MethodByName(name); ok {
			t.Fatalf("internal method exposed: %s", name)
		}
	}
	if typ.NumMethod() != 58 {
		t.Fatalf("review changes to the desktop API: got %d methods", typ.NumMethod())
	}
	for _, name := range []string{"Start", "Probe", "ChooseMediaPaths", "SaveProject", "ConfirmQuit", "GetDiagnosticPreview"} {
		if _, ok := typ.MethodByName(name); !ok {
			t.Fatalf("missing application API: %s", name)
		}
	}
}
