package identify

import "testing"

func TestPickEmail(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name   string
		fields map[string]string
		want   string
	}{
		{
			name:   "labelled email field wins",
			fields: map[string]string{"name": "Tom", "email": "tom@example.com"},
			want:   "tom@example.com",
		},
		{
			name:   "labelled with separator",
			fields: map[string]string{"your-email": "you@x.com"},
			want:   "you@x.com",
		},
		{
			name:   "case-insensitive label match",
			fields: map[string]string{"Email": "ttt@y.com"},
			want:   "ttt@y.com",
		},
		{
			name:   "fallback to email-shaped value when no labelled field",
			fields: map[string]string{"contact": "find me at hi@example.com"},
			want:   "", // value contains spaces - fails strict shape regex
		},
		{
			name:   "fallback picks bare email value",
			fields: map[string]string{"contact": "hi@example.com"},
			want:   "hi@example.com",
		},
		{
			name:   "empty values ignored",
			fields: map[string]string{"email": "", "name": "Tom"},
			want:   "",
		},
		{
			name:   "label match must also be email-shaped (defensive)",
			fields: map[string]string{"email": "not an email", "comments": "i@y.co"},
			want:   "i@y.co", // label fails shape, fallback picks value
		},
		{
			name:   "no email anywhere",
			fields: map[string]string{"name": "Tom", "phone": "+46123"},
			want:   "",
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := PickEmail(c.fields)
			if got != c.want {
				t.Errorf("PickEmail()=%q, want %q", got, c.want)
			}
		})
	}
}

func TestEmailValueRE(t *testing.T) {
	t.Parallel()
	cases := []struct {
		raw  string
		want bool
	}{
		{"a@b.c", true},
		{"name+tag@example.co.uk", true},
		{"", false},
		{"a@", false},
		{"a@b", false},
		{"a@b.", false},
		{"a b@c.com", false},
		{"@example.com", false},
		{"foo@bar@baz.com", false},
	}
	for _, c := range cases {
		got := emailValueRE.MatchString(c.raw)
		if got != c.want {
			t.Errorf("emailValueRE(%q)=%v, want %v", c.raw, got, c.want)
		}
	}
}
