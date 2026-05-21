package embedded

import "testing"

func TestNumberLines(t *testing.T) {
	cases := []struct {
		name string
		in   string
		want string
	}{
		{"empty", "", ""},
		{"single line no newline", "hello", "1| hello\n"},
		{"single line trailing newline", "hello\n", "1| hello\n"},
		{"two lines", "a\nb\n", "1| a\n2| b\n"},
		{
			"width adapts past 9 lines",
			"1\n2\n3\n4\n5\n6\n7\n8\n9\n10\n",
			" 1| 1\n 2| 2\n 3| 3\n 4| 4\n 5| 5\n 6| 6\n 7| 7\n 8| 8\n 9| 9\n10| 10\n",
		},
		{"blank interior line preserved", "a\n\nb\n", "1| a\n2| \n3| b\n"},
	}
	for _, c := range cases {
		if got := numberLines(c.in); got != c.want {
			t.Errorf("%s:\n got %q\nwant %q", c.name, got, c.want)
		}
	}
}
