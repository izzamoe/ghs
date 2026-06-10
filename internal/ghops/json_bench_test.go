package ghops

import (
	"encoding/json"
	"testing"
)

const userJSON = `{"id":275592473,"login":"zamyb","name":"IZZAMUDDIN","email":"work@example.com"}`

func BenchmarkUnmarshalUser_StringToBytes(b *testing.B) {
	// old approach: string → []byte(output) → unmarshal
	output := userJSON
	for b.Loop() {
		var user User
		_ = json.Unmarshal([]byte(output), &user)
	}
}

func BenchmarkUnmarshalUser_DirectBytes(b *testing.B) {
	// new approach: OutputBytes returns []byte → unmarshal directly
	output := []byte(userJSON)
	for b.Loop() {
		var user User
		_ = json.Unmarshal(output, &user)
	}
}

func BenchmarkParseAuthAccounts_DirectBytes(b *testing.B) {
	data := []byte(`{"hosts":{"github.com":[{"state":"success","active":true,"host":"github.com","login":"zamyb"}]}}`)
	for b.Loop() {
		_, _ = ParseAuthAccounts("github.com", data)
	}
}
