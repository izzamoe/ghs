package ghops

import "testing"

var sinkEmail string

func BenchmarkSelectPrimaryEmail(b *testing.B) {
	emails := []Email{
		{Email: "unverified@example.com", Primary: true, Verified: false},
		{Email: "secondary@example.com", Primary: false, Verified: true},
		{Email: "primary@example.com", Primary: true, Verified: true},
	}
	for b.Loop() {
		sinkEmail = SelectPrimaryEmail(emails)
	}
}

func BenchmarkNoreplyEmail(b *testing.B) {
	user := User{ID: 275592473, Login: "zamyb"}
	for b.Loop() {
		sinkEmail = NoreplyEmail(user)
	}
}
