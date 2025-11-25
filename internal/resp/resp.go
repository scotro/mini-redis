// Package resp implements the Redis Serialization Protocol (RESP) parser.
package resp

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"strconv"
)

// RESP type markers
const (
	TypeSimpleString = '+'
	TypeError        = '-'
	TypeInteger      = ':'
	TypeBulkString   = '$'
	TypeArray        = '*'
)

// Common errors
var (
	ErrInvalidType     = errors.New("invalid RESP type")
	ErrInvalidFormat   = errors.New("invalid RESP format")
	ErrInvalidLength   = errors.New("invalid length")
	ErrUnexpectedEOF   = errors.New("unexpected end of input")
	ErrInvalidInteger  = errors.New("invalid integer format")
)

// Value represents a RESP value
type Value struct {
	Type  byte     // '+', '-', ':', '$', '*'
	Str   string   // For simple strings, errors, and bulk strings
	Num   int      // For integers
	Array []Value  // For arrays
	Null  bool     // For null bulk strings and null arrays
}

// Parse reads and parses a RESP value from the reader.
func Parse(reader *bufio.Reader) (Value, error) {
	typeByte, err := reader.ReadByte()
	if err != nil {
		if err == io.EOF {
			return Value{}, ErrUnexpectedEOF
		}
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
		return Value{}, fmt.Errorf("%w: %c", ErrInvalidType, typeByte)
	}
}

// parseSimpleString parses a simple string (+OK\r\n)
func parseSimpleString(reader *bufio.Reader) (Value, error) {
	line, err := readLine(reader)
	if err != nil {
		return Value{}, err
	}
	return Value{Type: TypeSimpleString, Str: line}, nil
}

// parseError parses an error (-ERR message\r\n)
func parseError(reader *bufio.Reader) (Value, error) {
	line, err := readLine(reader)
	if err != nil {
		return Value{}, err
	}
	return Value{Type: TypeError, Str: line}, nil
}

// parseInteger parses an integer (:1000\r\n)
func parseInteger(reader *bufio.Reader) (Value, error) {
	line, err := readLine(reader)
	if err != nil {
		return Value{}, err
	}

	num, err := strconv.Atoi(line)
	if err != nil {
		return Value{}, ErrInvalidInteger
	}

	return Value{Type: TypeInteger, Num: num}, nil
}

// parseBulkString parses a bulk string ($5\r\nhello\r\n)
func parseBulkString(reader *bufio.Reader) (Value, error) {
	line, err := readLine(reader)
	if err != nil {
		return Value{}, err
	}

	length, err := strconv.Atoi(line)
	if err != nil {
		return Value{}, ErrInvalidLength
	}

	// Handle null bulk string ($-1\r\n)
	if length == -1 {
		return Value{Type: TypeBulkString, Null: true}, nil
	}

	if length < 0 {
		return Value{}, ErrInvalidLength
	}

	// Read the string content plus \r\n
	buf := make([]byte, length+2)
	_, err = io.ReadFull(reader, buf)
	if err != nil {
		if err == io.EOF || err == io.ErrUnexpectedEOF {
			return Value{}, ErrUnexpectedEOF
		}
		return Value{}, err
	}

	// Verify trailing \r\n
	if buf[length] != '\r' || buf[length+1] != '\n' {
		return Value{}, ErrInvalidFormat
	}

	return Value{Type: TypeBulkString, Str: string(buf[:length])}, nil
}

// parseArray parses an array (*2\r\n$3\r\nGET\r\n$3\r\nkey\r\n)
func parseArray(reader *bufio.Reader) (Value, error) {
	line, err := readLine(reader)
	if err != nil {
		return Value{}, err
	}

	count, err := strconv.Atoi(line)
	if err != nil {
		return Value{}, ErrInvalidLength
	}

	// Handle null array (*-1\r\n)
	if count == -1 {
		return Value{Type: TypeArray, Null: true}, nil
	}

	if count < 0 {
		return Value{}, ErrInvalidLength
	}

	array := make([]Value, count)
	for i := 0; i < count; i++ {
		val, err := Parse(reader)
		if err != nil {
			return Value{}, err
		}
		array[i] = val
	}

	return Value{Type: TypeArray, Array: array}, nil
}

// readLine reads until \r\n and returns the line without the delimiter.
func readLine(reader *bufio.Reader) (string, error) {
	line, err := reader.ReadString('\n')
	if err != nil {
		if err == io.EOF {
			return "", ErrUnexpectedEOF
		}
		return "", err
	}

	// Line must end with \r\n
	if len(line) < 2 || line[len(line)-2] != '\r' {
		return "", ErrInvalidFormat
	}

	return line[:len(line)-2], nil
}

// Serialize converts the Value back to RESP format.
func (v Value) Serialize() []byte {
	switch v.Type {
	case TypeSimpleString:
		return []byte(fmt.Sprintf("+%s\r\n", v.Str))
	case TypeError:
		return []byte(fmt.Sprintf("-%s\r\n", v.Str))
	case TypeInteger:
		return []byte(fmt.Sprintf(":%d\r\n", v.Num))
	case TypeBulkString:
		if v.Null {
			return []byte("$-1\r\n")
		}
		return []byte(fmt.Sprintf("$%d\r\n%s\r\n", len(v.Str), v.Str))
	case TypeArray:
		if v.Null {
			return []byte("*-1\r\n")
		}
		result := []byte(fmt.Sprintf("*%d\r\n", len(v.Array)))
		for _, elem := range v.Array {
			result = append(result, elem.Serialize()...)
		}
		return result
	default:
		return nil
	}
}
