package types

import (
	"bytes"
	"compress/gzip"
	"database/sql/driver"
	"encoding/json"
	"errors"

	"io/ioutil"
)

type GzippedText []byte

func (g GzippedText) Value() (driver.Value, error) {
	b := make([]byte, 0, len(g))
	buf := bytes.NewBuffer(b)
	w := gzip.NewWriter(buf)
	w.Write(g)
	w.Close()
	return buf.Bytes(), nil

}
func (g *GzippedText) Scan(src interface{}) error {
	var source []byte
	switch src := src.(type) {
	case string:
		source = []byte(src)
	case []byte:
		source = src
	default:
		return errors.New("Incompatible type for GzippedText")
	}
	reader, err := gzip.NewReader(bytes.NewReader(source))
	if err != nil {
		return err
	}
	defer reader.Close()
	b, err := ioutil.ReadAll(reader)
	if err != nil {
		return err
	}
	*g = GzippedText(b)
	return nil
}

type JSONText json.RawMessage

var emptyJSON = JSONText("{}")

func (j JSONText) MarshalJSON() ([]byte, error) {
	if len(j) == 0 {
		return emptyJSON, nil
	}
	return j, nil
}
func (j *JSONText) UnmarshalJSON(data []byte) error {
	if j == nil {
		return errors.New("JSONText: UnmarshalJSON on nil pointer")
	}
	*j = append((*j)[0:0], data...)
	return nil
}
func (j JSONText) Value() (driver.Value, error) {
	var m json.RawMessage
	var err = j.Unmarshal(&m)
	if err != nil {
		return []byte{}, err
	}
	return []byte(j), nil
}
func (j *JSONText) Scan(src interface{}) error {
	var source []byte
	switch t := src.(type) {
	case string:
		source = []byte(t)
	case []byte:
		if len(t) == 0 {
			source = emptyJSON
		} else {
			source = t
		}
	case nil:
		*j = emptyJSON
	default:
		return errors.New("Incompatible type for JSONText")
	}
	*j = append((*j)[0:0], source...)
	return nil
}
func (j *JSONText) Unmarshal(v interface{}) error {
	if len(*j) == 0 {
		*j = emptyJSON
	}
	return json.Unmarshal([]byte(*j), v)
}
func (j JSONText) String() string {
	return string(j)
}

type NullJSONText struct {
	JSONText
	Valid bool // Valid is true if JSONText is not NULL
}

func (n *NullJSONText) Scan(value interface{}) error {
	if value == nil {
		n.JSONText, n.Valid = emptyJSON, false
		return nil
	}
	n.Valid = true
	return n.JSONText.Scan(value)
}
func (n NullJSONText) Value() (driver.Value, error) {
	if !n.Valid {
		return nil, nil
	}
	return n.JSONText.Value()
}

type BitBool bool

func (b BitBool) Value() (driver.Value, error) {
	if b {
		return []byte{1}, nil
	}
	return []byte{0}, nil
}
func (b *BitBool) Scan(src interface{}) error {
	v, ok := src.([]byte)
	if !ok {
		return errors.New("bad []byte type assertion")
	}
	*b = v[0] == 1
	return nil
}
