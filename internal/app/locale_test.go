package app

import "testing"

func TestLocaleSelectionDoesNotAcceptUnknownOrModifyOnFailure(t *testing.T) {
	s := &Service{}
	if s.Locale() != "zh-CN" {
		t.Fatal("unexpected default")
	}
	if err := s.SetLocale("en-US"); err != nil || s.Locale() != "en-US" {
		t.Fatal("language did not change", err)
	}
	if err := s.SetLocale("invalid"); err == nil || s.Locale() != "en-US" {
		t.Fatal("invalid locale changed state")
	}
	if err := s.SetLocale("zh-CN"); err != nil || s.Locale() != "zh-CN" {
		t.Fatal("cannot switch back", err)
	}
}
