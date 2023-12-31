// Copyright (C) 2023 Takayuki Sato. All Rights Reserved.
// This program is free software under MIT License.
// See the file LICENSE in this distribution for more details.

package v0_5_0

import (
	"bytes"
	"encoding/json"
	"io"
	"reflect"
	"strconv"
	"strings"
)

// MarshalJSON returns a byte array of JSON string which expresses the content
// of this map.
func (om Map[K, V]) MarshalJSON() ([]byte, error) {
	var buf bytes.Buffer
	buf.WriteString("{")

	ent := om.Front()
	if ent != nil {
		err := addJsonKey(&buf, ent.Key())
		if err != nil {
			return nil, err
		}
		buf.Write([]byte(":"))
		err = addJsonValue(&buf, ent.Value())
		if err != nil {
			return nil, err
		}

		for ent = ent.Next(); ent != nil; ent = ent.Next() {
			buf.WriteString(",")
			err = addJsonKey(&buf, ent.Key())
			if err != nil {
				return nil, err
			}
			buf.WriteString(":")
			err = addJsonValue(&buf, ent.Value())
			if err != nil {
				return nil, err
			}
		}
	}

	buf.WriteString("}")
	return buf.Bytes(), nil
}

// UnsupportedTypeError is an error type which is returned by Marshal when
// attempting to encode an unsupported key type.
type UnsupportedKeyTypeError struct {
	Type reflect.Type
}

func (err UnsupportedKeyTypeError) Error() string {
	if err.Type == nil {
		return "json: unsupported key type: any"
	} else {
		return "json: unsupported key type: " + err.Type.String()
	}
}

// SyntaxError is an error stype which is returned by Unmarshal when an input
// json does not start with "{" or end with "}", or there are value type
// mismatches.
type SyntaxError struct {
	Offset int64
	msg    string
}

func (err SyntaxError) Error() string {
	return err.msg + " (offset:" + strconv.FormatInt(err.Offset, 10) + ")"
}

func addJsonKey(buf *bytes.Buffer, key any) error {
	quote := false
	switch key.(type) {
	default:
		return UnsupportedKeyTypeError{Type: reflect.TypeOf(key)}
	case string:
	case *string:
		if key == (*string)(nil) {
			quote = true
		}
	case bool, int, int8, int16, int32, int64, uint, uint8, uint16,
		uint32, uint64, float32, float64,
		*bool, *int, *int8, *int16, *int32, *int64, *uint, *uint8, *uint16,
		*uint32, *uint64, *float32, *float64:
		quote = true
	}
	bs, err := json.Marshal(key)
	if err != nil {
		return err
	}
	if quote {
		buf.WriteString(`"`)
		buf.Write(bs)
		buf.WriteString(`"`)
	} else {
		buf.Write(bs)
	}
	return nil
}

func addJsonValue[V any](buf *bytes.Buffer, val V) error {
	bs, err := json.Marshal(val)
	if err != nil {
		return err
	}
	buf.Write(bs)
	return nil
}

// UnmarshalJSON sets the content of this map from a JSON data.
func (om *Map[K, V]) UnmarshalJSON(data []byte) error {
	dec := json.NewDecoder(strings.NewReader(string(data)))

	// Open bracket
	tok, err := dec.Token()
	if err == io.EOF {
		return nil
	}
	if err != nil {
		return err
	}
	ok := false
	switch tok.(type) {
	case json.Delim:
		if tok.(json.Delim).String() == "{" {
			ok = true
		}
	}
	if !ok {
		return SyntaxError{
			Offset: 0,
			msg:    "The input JSON does not start with '{'",
		}
	}

	depth := 0
	for {
		tok, err := dec.Token()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}
		switch tok.(type) {
		case json.Delim:
			switch tok.(json.Delim).String() {
			case "{":
				return SyntaxError{
					Offset: dec.InputOffset(),
					msg:    "Invalid character '" + tok.(json.Delim).String() + "'",
				}
			case "}":
				depth--
			}
			continue
		}

		if depth == 0 {
			var key K
			switch any(key).(type) {
			case string:
				key = any(tok).(K)
			case *string:
				if tok == "null" {
					key = *new(K)
				} else {
					str := tok.(string)
					key = any(&str).(K)
				}
			case bool, int, int8, int16, int32, int64, uint, uint8,
				uint16, uint32, uint64, float32, float64:
				err = json.Unmarshal([]byte(tok.(string)), &key)
				if err != nil {
					return err
				}
			case *bool, *int, *int8, *int16, *int32, *int64, *uint, *uint8,
				*uint16, *uint32, *uint64, *float32, *float64:
				tt := reflect.TypeOf(key).Elem()
				key = reflect.New(tt).Interface().(K)
				err = json.Unmarshal([]byte(tok.(string)), key)
				if err != nil {
					return err
				}
			default:
				return &UnsupportedKeyTypeError{Type: reflect.TypeOf(key)}
			}
			var val V
			dec.Decode(&val)
			om.Store(key, val)
		}
	}

	if depth >= 0 {
		return SyntaxError{
			Offset: dec.InputOffset(),
			msg:    "The input JSON does not end with '}'",
		}
	}
	return nil
}
