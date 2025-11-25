package resp

import (
	"bufio"
	"bytes"
	"strings"
	"testing"
)

func TestParseSimpleString(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    Value
		wantErr bool
	}{
		{
			name:  "simple OK",
			input: "+OK\r\n",
			want:  Value{Type: TypeSimpleString, Str: "OK"},
		},
		{
			name:  "simple PONG",
			input: "+PONG\r\n",
			want:  Value{Type: TypeSimpleString, Str: "PONG"},
		},
		{
			name:  "empty string",
			input: "+\r\n",
			want:  Value{Type: TypeSimpleString, Str: ""},
		},
		{
			name:  "string with spaces",
			input: "+hello world\r\n",
			want:  Value{Type: TypeSimpleString, Str: "hello world"},
		},
		{
			name:    "missing CRLF",
			input:   "+OK",
			wantErr: true,
		},
		{
			name:    "wrong line ending",
			input:   "+OK\n",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reader := bufio.NewReader(strings.NewReader(tt.input))
			got, err := Parse(reader)
			if (err != nil) != tt.wantErr {
				t.Errorf("Parse() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr {
				if got.Type != tt.want.Type {
					t.Errorf("Parse() Type = %v, want %v", got.Type, tt.want.Type)
				}
				if got.Str != tt.want.Str {
					t.Errorf("Parse() Str = %v, want %v", got.Str, tt.want.Str)
				}
			}
		})
	}
}

func TestParseError(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    Value
		wantErr bool
	}{
		{
			name:  "simple error",
			input: "-ERR unknown command\r\n",
			want:  Value{Type: TypeError, Str: "ERR unknown command"},
		},
		{
			name:  "WRONGTYPE error",
			input: "-WRONGTYPE Operation against a key holding the wrong kind of value\r\n",
			want:  Value{Type: TypeError, Str: "WRONGTYPE Operation against a key holding the wrong kind of value"},
		},
		{
			name:  "empty error",
			input: "-\r\n",
			want:  Value{Type: TypeError, Str: ""},
		},
		{
			name:    "missing CRLF",
			input:   "-ERR",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reader := bufio.NewReader(strings.NewReader(tt.input))
			got, err := Parse(reader)
			if (err != nil) != tt.wantErr {
				t.Errorf("Parse() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr {
				if got.Type != tt.want.Type {
					t.Errorf("Parse() Type = %v, want %v", got.Type, tt.want.Type)
				}
				if got.Str != tt.want.Str {
					t.Errorf("Parse() Str = %v, want %v", got.Str, tt.want.Str)
				}
			}
		})
	}
}

func TestParseInteger(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    Value
		wantErr bool
	}{
		{
			name:  "positive integer",
			input: ":1000\r\n",
			want:  Value{Type: TypeInteger, Num: 1000},
		},
		{
			name:  "zero",
			input: ":0\r\n",
			want:  Value{Type: TypeInteger, Num: 0},
		},
		{
			name:  "negative integer",
			input: ":-1\r\n",
			want:  Value{Type: TypeInteger, Num: -1},
		},
		{
			name:  "large positive",
			input: ":999999999\r\n",
			want:  Value{Type: TypeInteger, Num: 999999999},
		},
		{
			name:    "invalid integer",
			input:   ":abc\r\n",
			wantErr: true,
		},
		{
			name:    "missing CRLF",
			input:   ":100",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reader := bufio.NewReader(strings.NewReader(tt.input))
			got, err := Parse(reader)
			if (err != nil) != tt.wantErr {
				t.Errorf("Parse() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr {
				if got.Type != tt.want.Type {
					t.Errorf("Parse() Type = %v, want %v", got.Type, tt.want.Type)
				}
				if got.Num != tt.want.Num {
					t.Errorf("Parse() Num = %v, want %v", got.Num, tt.want.Num)
				}
			}
		})
	}
}

func TestParseBulkString(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    Value
		wantErr bool
	}{
		{
			name:  "simple bulk string",
			input: "$5\r\nhello\r\n",
			want:  Value{Type: TypeBulkString, Str: "hello"},
		},
		{
			name:  "empty bulk string",
			input: "$0\r\n\r\n",
			want:  Value{Type: TypeBulkString, Str: ""},
		},
		{
			name:  "null bulk string",
			input: "$-1\r\n",
			want:  Value{Type: TypeBulkString, Null: true},
		},
		{
			name:  "bulk string with spaces",
			input: "$11\r\nhello world\r\n",
			want:  Value{Type: TypeBulkString, Str: "hello world"},
		},
		{
			name:  "bulk string with binary data",
			input: "$6\r\nfoo\r\nb\r\n",
			want:  Value{Type: TypeBulkString, Str: "foo\r\nb"},
		},
		{
			name:    "invalid length",
			input:   "$abc\r\nhello\r\n",
			wantErr: true,
		},
		{
			name:    "negative length (not -1)",
			input:   "$-2\r\n",
			wantErr: true,
		},
		{
			name:    "truncated content",
			input:   "$10\r\nhello\r\n",
			wantErr: true,
		},
		{
			name:    "missing trailing CRLF",
			input:   "$5\r\nhello",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reader := bufio.NewReader(strings.NewReader(tt.input))
			got, err := Parse(reader)
			if (err != nil) != tt.wantErr {
				t.Errorf("Parse() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr {
				if got.Type != tt.want.Type {
					t.Errorf("Parse() Type = %v, want %v", got.Type, tt.want.Type)
				}
				if got.Str != tt.want.Str {
					t.Errorf("Parse() Str = %v, want %v", got.Str, tt.want.Str)
				}
				if got.Null != tt.want.Null {
					t.Errorf("Parse() Null = %v, want %v", got.Null, tt.want.Null)
				}
			}
		})
	}
}

func TestParseArray(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    Value
		wantErr bool
	}{
		{
			name:  "simple array",
			input: "*2\r\n$3\r\nGET\r\n$3\r\nkey\r\n",
			want: Value{
				Type: TypeArray,
				Array: []Value{
					{Type: TypeBulkString, Str: "GET"},
					{Type: TypeBulkString, Str: "key"},
				},
			},
		},
		{
			name:  "empty array",
			input: "*0\r\n",
			want:  Value{Type: TypeArray, Array: []Value{}},
		},
		{
			name:  "null array",
			input: "*-1\r\n",
			want:  Value{Type: TypeArray, Null: true},
		},
		{
			name:  "mixed type array",
			input: "*3\r\n+OK\r\n:100\r\n$5\r\nhello\r\n",
			want: Value{
				Type: TypeArray,
				Array: []Value{
					{Type: TypeSimpleString, Str: "OK"},
					{Type: TypeInteger, Num: 100},
					{Type: TypeBulkString, Str: "hello"},
				},
			},
		},
		{
			name:  "nested array",
			input: "*2\r\n*2\r\n$3\r\nfoo\r\n$3\r\nbar\r\n*1\r\n:42\r\n",
			want: Value{
				Type: TypeArray,
				Array: []Value{
					{
						Type: TypeArray,
						Array: []Value{
							{Type: TypeBulkString, Str: "foo"},
							{Type: TypeBulkString, Str: "bar"},
						},
					},
					{
						Type: TypeArray,
						Array: []Value{
							{Type: TypeInteger, Num: 42},
						},
					},
				},
			},
		},
		{
			name:  "array with null element",
			input: "*3\r\n$3\r\nfoo\r\n$-1\r\n$3\r\nbar\r\n",
			want: Value{
				Type: TypeArray,
				Array: []Value{
					{Type: TypeBulkString, Str: "foo"},
					{Type: TypeBulkString, Null: true},
					{Type: TypeBulkString, Str: "bar"},
				},
			},
		},
		{
			name:    "invalid count",
			input:   "*abc\r\n",
			wantErr: true,
		},
		{
			name:    "negative count (not -1)",
			input:   "*-2\r\n",
			wantErr: true,
		},
		{
			name:    "truncated array",
			input:   "*3\r\n$3\r\nfoo\r\n",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reader := bufio.NewReader(strings.NewReader(tt.input))
			got, err := Parse(reader)
			if (err != nil) != tt.wantErr {
				t.Errorf("Parse() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr {
				if got.Type != tt.want.Type {
					t.Errorf("Parse() Type = %v, want %v", got.Type, tt.want.Type)
				}
				if got.Null != tt.want.Null {
					t.Errorf("Parse() Null = %v, want %v", got.Null, tt.want.Null)
				}
				if !compareArrays(got.Array, tt.want.Array) {
					t.Errorf("Parse() Array = %v, want %v", got.Array, tt.want.Array)
				}
			}
		})
	}
}

func TestParseInvalidType(t *testing.T) {
	input := "?invalid\r\n"
	reader := bufio.NewReader(strings.NewReader(input))
	_, err := Parse(reader)
	if err == nil {
		t.Error("Parse() expected error for invalid type")
	}
}

func TestParseEmptyInput(t *testing.T) {
	reader := bufio.NewReader(strings.NewReader(""))
	_, err := Parse(reader)
	if err == nil {
		t.Error("Parse() expected error for empty input")
	}
}

func TestSerializeSimpleString(t *testing.T) {
	tests := []struct {
		name  string
		value Value
		want  string
	}{
		{
			name:  "simple OK",
			value: Value{Type: TypeSimpleString, Str: "OK"},
			want:  "+OK\r\n",
		},
		{
			name:  "empty string",
			value: Value{Type: TypeSimpleString, Str: ""},
			want:  "+\r\n",
		},
		{
			name:  "string with spaces",
			value: Value{Type: TypeSimpleString, Str: "hello world"},
			want:  "+hello world\r\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.value.Serialize()
			if string(got) != tt.want {
				t.Errorf("Serialize() = %q, want %q", string(got), tt.want)
			}
		})
	}
}

func TestSerializeError(t *testing.T) {
	tests := []struct {
		name  string
		value Value
		want  string
	}{
		{
			name:  "simple error",
			value: Value{Type: TypeError, Str: "ERR unknown command"},
			want:  "-ERR unknown command\r\n",
		},
		{
			name:  "empty error",
			value: Value{Type: TypeError, Str: ""},
			want:  "-\r\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.value.Serialize()
			if string(got) != tt.want {
				t.Errorf("Serialize() = %q, want %q", string(got), tt.want)
			}
		})
	}
}

func TestSerializeInteger(t *testing.T) {
	tests := []struct {
		name  string
		value Value
		want  string
	}{
		{
			name:  "positive integer",
			value: Value{Type: TypeInteger, Num: 1000},
			want:  ":1000\r\n",
		},
		{
			name:  "zero",
			value: Value{Type: TypeInteger, Num: 0},
			want:  ":0\r\n",
		},
		{
			name:  "negative integer",
			value: Value{Type: TypeInteger, Num: -1},
			want:  ":-1\r\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.value.Serialize()
			if string(got) != tt.want {
				t.Errorf("Serialize() = %q, want %q", string(got), tt.want)
			}
		})
	}
}

func TestSerializeBulkString(t *testing.T) {
	tests := []struct {
		name  string
		value Value
		want  string
	}{
		{
			name:  "simple bulk string",
			value: Value{Type: TypeBulkString, Str: "hello"},
			want:  "$5\r\nhello\r\n",
		},
		{
			name:  "empty bulk string",
			value: Value{Type: TypeBulkString, Str: ""},
			want:  "$0\r\n\r\n",
		},
		{
			name:  "null bulk string",
			value: Value{Type: TypeBulkString, Null: true},
			want:  "$-1\r\n",
		},
		{
			name:  "bulk string with CRLF",
			value: Value{Type: TypeBulkString, Str: "foo\r\nb"},
			want:  "$6\r\nfoo\r\nb\r\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.value.Serialize()
			if string(got) != tt.want {
				t.Errorf("Serialize() = %q, want %q", string(got), tt.want)
			}
		})
	}
}

func TestSerializeArray(t *testing.T) {
	tests := []struct {
		name  string
		value Value
		want  string
	}{
		{
			name: "simple array",
			value: Value{
				Type: TypeArray,
				Array: []Value{
					{Type: TypeBulkString, Str: "GET"},
					{Type: TypeBulkString, Str: "key"},
				},
			},
			want: "*2\r\n$3\r\nGET\r\n$3\r\nkey\r\n",
		},
		{
			name:  "empty array",
			value: Value{Type: TypeArray, Array: []Value{}},
			want:  "*0\r\n",
		},
		{
			name:  "null array",
			value: Value{Type: TypeArray, Null: true},
			want:  "*-1\r\n",
		},
		{
			name: "mixed type array",
			value: Value{
				Type: TypeArray,
				Array: []Value{
					{Type: TypeSimpleString, Str: "OK"},
					{Type: TypeInteger, Num: 100},
					{Type: TypeBulkString, Str: "hello"},
				},
			},
			want: "*3\r\n+OK\r\n:100\r\n$5\r\nhello\r\n",
		},
		{
			name: "nested array",
			value: Value{
				Type: TypeArray,
				Array: []Value{
					{
						Type: TypeArray,
						Array: []Value{
							{Type: TypeBulkString, Str: "foo"},
						},
					},
				},
			},
			want: "*1\r\n*1\r\n$3\r\nfoo\r\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.value.Serialize()
			if string(got) != tt.want {
				t.Errorf("Serialize() = %q, want %q", string(got), tt.want)
			}
		})
	}
}

func TestRoundTrip(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{"simple string", "+OK\r\n"},
		{"error", "-ERR unknown command\r\n"},
		{"integer", ":1000\r\n"},
		{"negative integer", ":-1\r\n"},
		{"bulk string", "$5\r\nhello\r\n"},
		{"null bulk string", "$-1\r\n"},
		{"empty bulk string", "$0\r\n\r\n"},
		{"simple array", "*2\r\n$3\r\nGET\r\n$3\r\nkey\r\n"},
		{"empty array", "*0\r\n"},
		{"null array", "*-1\r\n"},
		{"mixed array", "*3\r\n+OK\r\n:100\r\n$5\r\nhello\r\n"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reader := bufio.NewReader(strings.NewReader(tt.input))
			parsed, err := Parse(reader)
			if err != nil {
				t.Fatalf("Parse() error = %v", err)
			}

			serialized := parsed.Serialize()
			if !bytes.Equal(serialized, []byte(tt.input)) {
				t.Errorf("RoundTrip() got %q, want %q", string(serialized), tt.input)
			}
		})
	}
}

// compareArrays recursively compares two slices of Value
func compareArrays(a, b []Value) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if !compareValues(a[i], b[i]) {
			return false
		}
	}
	return true
}

// compareValues recursively compares two Values
func compareValues(a, b Value) bool {
	if a.Type != b.Type || a.Str != b.Str || a.Num != b.Num || a.Null != b.Null {
		return false
	}
	return compareArrays(a.Array, b.Array)
}
