package auth

import (
	"testing"
	"time"
)

func TestPasswordHashAndLocalLimits(t *testing.T) {
	salt := []byte("synthetic-salt01")
	first := derive("Synthetic-password-123", salt)
	second := derive("Synthetic-password-123", salt)
	if string(first) != string(second) || string(first) == string(derive("different-password", salt)) {
		t.Fatal("password derivation mismatch")
	}
	service := New(nil)
	if !service.acquire() || !service.acquire() || service.acquire() {
		t.Fatal("password concurrency is not bounded")
	}
	<-service.slots
	<-service.slots
	for range 27 {
		if !service.acquire() {
			t.Fatal("budget exhausted early")
		}
		<-service.slots
	}
	if service.acquire() {
		t.Fatal("rate limit missing")
	}
	service.window = time.Now().Add(-time.Minute - time.Second)
	if !service.acquire() {
		t.Fatal("budget did not reset")
	}
	<-service.slots
}
