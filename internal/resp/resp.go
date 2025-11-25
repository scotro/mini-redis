// Package resp implements the Redis Serialization Protocol (RESP).
// This is a mock implementation for server development.
// Will be replaced by the real implementation from the resp agent.
package resp

import (
	"bufio"
	"errors"
	"fmt"
	"strconv"
	"strings"
)

// RESP type bytes
const (
	TypeSimpleString = '+'
	TypeError        = '-'
	TypeInteger      = ':'
	TypeBulkString   = '$'
	TypeArray        = '*'
)

// Value represents a RESP value.
type Value struct {
	Type  byte
	Str   string
	Num   int
	Array []Value
}

// Common errors
var (
	ErrInvalidSyntax = errors.New("invalid RESP syntax")
	ErrUnexpectedEOF = errors.New("unexpected EOF")
)

// Parse reads a RESP value from the reader.
func Parse(reader *bufio.Reader) (Value, error) {
	typeByte, err := reader.ReadByte()
	if err != nil {
		return Value{}, err
	}

	switch typeByte {
	case TypeSimpleString:
		return parseSimpleString(reader)
	case TypeError:
		return parseError(reader)
	case TypeInteger:
		return parseInteger(reader)
	case TypeBulkString:
		return parseBulkString(reader)
	case TypeArray:
		return parseArray(reader)
	default:
		return Value{}, fmt.Errorf("%w: unknown type byte '%c'", ErrInvalidSyntax, typeByte)
	}
}

func parseSimpleString(reader *bufio.Reader) (Value, error) {
	line, err := readLine(reader)
	if err != nil {
		return Value{}, err
	}
	return Value{Type: TypeSimpleString, Str: line}, nil
}

func parseError(reader *bufio.Reader) (Value, error) {
	line, err := readLine(reader)
	if err != nil {
		return Value{}, err
	}
	return Value{Type: TypeError, Str: line}, nil
}

func parseInteger(reader *bufio.Reader) (Value, error) {
	line, err := readLine(reader)
	if err != nil {
		return Value{}, err
	}
	num, err := strconv.Atoi(line)
	if err != nil {
		return Value{}, fmt.Errorf("%w: invalid integer '%s'", ErrInvalidSyntax, line)
	}
	return Value{Type: TypeInteger, Num: num}, nil
}

func parseBulkString(reader *bufio.Reader) (Value, error) {
	line, err := readLine(reader)
	if err != nil {
		return Value{}, err
	}
	length, err := strconv.Atoi(line)
	if err != nil {
		return Value{}, fmt.Errorf("%w: invalid bulk string length '%s'", ErrInvalidSyntax, line)
	}

	if length == -1 {
		return Value{Type: TypeBulkString, Str: "", Num: -1}, nil // null bulk string
	}

	data := make([]byte, length+2) // +2 for CRLF
	_, err = reader.Read(data)
	if err != nil {
		return Value{}, err
	}

	return Value{Type: TypeBulkString, Str: string(data[:length])}, nil
}

func parseArray(reader *bufio.Reader) (Value, error) {
	line, err := readLine(reader)
	if err != nil {
		return Value{}, err
	}
	count, err := strconv.Atoi(line)
	if err != nil {
		return Value{}, fmt.Errorf("%w: invalid array count '%s'", ErrInvalidSyntax, line)
	}

	if count == -1 {
		return Value{Type: TypeArray, Array: nil}, nil // null array
	}

	arr := make([]Value, count)
	for i := 0; i < count; i++ {
		val, err := Parse(reader)
		if err != nil {
			return Value{}, err
		}
		arr[i] = val
	}

	return Value{Type: TypeArray, Array: arr}, nil
}

func readLine(reader *bufio.Reader) (string, error) {
	line, err := reader.ReadString('\n')
	if err != nil {
		return "", err
	}
	return strings.TrimSuffix(line, "\r\n"), nil
}

// Serialize converts a Value to its RESP wire format.
func (v Value) Serialize() []byte {
	switch v.Type {
	case TypeSimpleString:
		return []byte(fmt.Sprintf("+%s\r\n", v.Str))
	case TypeError:
		return []byte(fmt.Sprintf("-%s\r\n", v.Str))
	case TypeInteger:
		return []byte(fmt.Sprintf(":%d\r\n", v.Num))
	case TypeBulkString:
		if v.Str == "" && v.Num == -1 {
			return []byte("$-1\r\n") // null bulk string
		}
		return []byte(fmt.Sprintf("$%d\r\n%s\r\n", len(v.Str), v.Str))
	case TypeArray:
		if v.Array == nil {
			return []byte("*-1\r\n") // null array
		}
		result := []byte(fmt.Sprintf("*%d\r\n", len(v.Array)))
		for _, elem := range v.Array {
			result = append(result, elem.Serialize()...)
		}
		return result
	default:
		return []byte("-ERR unknown type\r\n")
	}
}

// Helper constructors for common responses
func SimpleString(s string) Value {
	return Value{Type: TypeSimpleString, Str: s}
}

func Error(s string) Value {
	return Value{Type: TypeError, Str: s}
}

func Integer(n int) Value {
	return Value{Type: TypeInteger, Num: n}
}

func BulkString(s string) Value {
	return Value{Type: TypeBulkString, Str: s}
}

func NullBulkString() Value {
	return Value{Type: TypeBulkString, Str: "", Num: -1}
}

func Array(values ...Value) Value {
	return Value{Type: TypeArray, Array: values}
}

// IsNull returns true if this is a null bulk string or null array.
func (v Value) IsNull() bool {
	if v.Type == TypeBulkString && v.Str == "" && v.Num == -1 {
		return true
	}
	if v.Type == TypeArray && v.Array == nil {
		return true
	}
	return false
}
