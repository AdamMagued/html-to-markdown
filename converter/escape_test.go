package converter

import (
	"testing"
)

func TestEscapeContent(t *testing.T) {
	conv := NewConverter()
	conv.Register.EscapedChar('>')

	input := []byte{'a', '>'}

	output := conv.escapeContent(input)
	if len(output) != 3 {
		t.Error("expected different length")
	}
	// Since '>' is a character used for quotes in markdown,
	// there should be a marker before it.
	if output[0] != 'a' || output[1] != placeholderByte || output[2] != '>' {
		t.Error("expected different characters")
	}
}

func TestUnEscapeContent(t *testing.T) {
	conv := NewConverter()
	conv.Register.UnEscaper(func(chars []byte, index int) int {
		if chars[index] != '>' {
			return -1
		}

		// A bit too simplistic for demonstration purposes.
		// Normally here would be content to check if the escaping is necessary...
		return 1
	}, PriorityStandard)

	input := []byte{placeholderByte, 'a', 'b'}
	output := conv.unEscapeContent(input)
	if len(output) != 2 {
		t.Error("expected different length")
	}
	// No escaping needed
	if output[0] != 'a' || output[1] != 'b' {
		t.Error("expected different characters")
	}

	input = []byte{placeholderByte, '>', 'a'}
	output = conv.unEscapeContent(input)
	if len(output) != 3 {
		t.Error("expected different length")
	}
	// Escaping needed
	if output[0] != '\\' || output[1] != '>' || output[2] != 'a' {
		t.Error("expected different characters")
	}
}

func TestEscapeContent_SanitizePlaceholderByte(t *testing.T) {
	conv := NewConverter()
	conv.Register.EscapedChar('*')

	testCases := []struct {
		name     string
		input    []byte
		expected []byte
	}{
		{
			name:     "standalone placeholder byte",
			input:    []byte{placeholderByte},
			expected: []byte(string('\ufffd')),
		},
		{
			name:     "null byte",
			input:    []byte{'\u0000'},
			expected: []byte(string('\ufffd')),
		},
		{
			name:     "placeholder byte before text",
			input:    []byte{placeholderByte, 'b', 'o', 'l', 'd'},
			expected: append([]byte(string('\ufffd')), 'b', 'o', 'l', 'd'),
		},
		{
			name:     "placeholder byte after text",
			input:    []byte{'t', 'e', 'x', 't', placeholderByte},
			expected: append([]byte("text"), []byte(string('\ufffd'))...),
		},
		{
			name:     "placeholder byte between text",
			input:    []byte{'a', placeholderByte, 'b'},
			expected: append(append([]byte("a"), []byte(string('\ufffd'))...), 'b'),
		},
		{
			name:     "placeholder byte adjacent to escaped char",
			input:    []byte{placeholderByte, '*'},
			expected: append([]byte(string('\ufffd')), placeholderByte, '*'),
		},
		{
			name:     "multiple placeholder bytes",
			input:    []byte{placeholderByte, placeholderByte, placeholderByte},
			expected: []byte(string("\ufffd\ufffd\ufffd")),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			output := conv.escapeContent(tc.input)
			if string(output) != string(tc.expected) {
				t.Errorf("expected %q, got %q", string(tc.expected), string(output))
			}
		})
	}
}

