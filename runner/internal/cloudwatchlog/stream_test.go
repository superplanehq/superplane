package cloudwatchlog

import "testing"

func TestNormalizeTerminalOutput(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
	}{
		{name: "no cr", in: "hello\nworld", want: "hello\nworld"},
		{name: "git progress", in: "Counting objects:  1%\rCounting objects:  2%\rCounting objects:  3%\n", want: "Counting objects:  3%\n"},
		{name: "incomplete line", in: "Receiving objects: 10%\rReceiving objects: 11%", want: "Receiving objects: 11%"},
		{name: "crlf", in: "line\r\nnext", want: "line\nnext"},
		{name: "multiple lines", in: "a\rbc\nd\r ef\n", want: "bc\n ef\n"},
		{
			name: "apt progress",
			in:   "\x1b[33mReading package lists... 0%\x1b[0m\r\x1b[33mReading package lists... 50%\x1b[0m\r\x1b[33mReading package lists... Done\x1b[0m\n",
			want: "Reading package lists... Done\n",
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := string(normalizeTerminalOutput([]byte(tc.in)))
			if got != tc.want {
				t.Fatalf("got %q, want %q", got, tc.want)
			}
		})
	}
}

func TestCollapseCarriageReturns(t *testing.T) {
	t.Run("delegates via normalize for cr-only", func(t *testing.T) {
		in := "Counting objects:  1%\rCounting objects:  3%\n"
		if got, want := string(collapseCarriageReturns([]byte(in))), "Counting objects:  3%\n"; got != want {
			t.Fatalf("got %q, want %q", got, want)
		}
	})
}

func TestBytesToEvents_stripsTrailingCR(t *testing.T) {
	var lastTS int64
	ev := bytesToEvents([]byte("line\r\n"), &lastTS)
	if len(ev) != 1 || ev[0].Message == nil || *ev[0].Message != "line" {
		t.Fatalf("events = %#v", ev)
	}
}
