package models

import (
	"bytes"
	"context"
	"database/sql/driver"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"gorm.io/gorm/schema"
)

// JSONValue stores arbitrary JSON while preserving JSON number tokens as json.Number
// when decoding (DB scan, API ingress, and in-memory UnmarshalJSON). Writes marshal
// the stored value as-is, so tokens survive round-trips once ingress avoids float64.
type JSONValue struct {
	data any
}

func NewJSONValue(data any) JSONValue {
	return JSONValue{data: data}
}

func (j JSONValue) Data() any {
	return j.data
}

func (j JSONValue) Value() (driver.Value, error) {
	return json.Marshal(j.data)
}

func (j *JSONValue) Scan(value any) error {
	if value == nil {
		j.data = nil
		return nil
	}

	var data []byte
	switch v := value.(type) {
	case []byte:
		data = v
	case string:
		data = []byte(v)
	default:
		return fmt.Errorf("failed to unmarshal JSONB value: %v", value)
	}

	return UnmarshalJSONValue(data, &j.data)
}

func (j JSONValue) MarshalJSON() ([]byte, error) {
	return json.Marshal(j.data)
}

func (j *JSONValue) UnmarshalJSON(data []byte) error {
	return UnmarshalJSONValue(data, &j.data)
}

func (JSONValue) GormDataType() string {
	return "json"
}

func (JSONValue) GormDBDataType(db *gorm.DB, _ *schema.Field) string {
	switch db.Dialector.Name() {
	case "sqlite":
		return "JSON"
	case "mysql":
		return "JSON"
	case "postgres":
		return "JSONB"
	default:
		return ""
	}
}

func (j JSONValue) GormValue(_ context.Context, db *gorm.DB) clause.Expr {
	data, _ := j.MarshalJSON()

	switch db.Dialector.Name() {
	case "mysql":
		if v, ok := db.Dialector.(*mysql.Dialector); ok && !strings.Contains(v.ServerVersion, "MariaDB") {
			return gorm.Expr("CAST(? AS JSON)", string(data))
		}
	}

	return gorm.Expr("?", string(data))
}

// UnmarshalJSONValue decodes JSON into value using json.Number for numeric tokens
// and rejects payloads with multiple top-level values.
func UnmarshalJSONValue(data []byte, value any) error {
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.UseNumber()
	if err := decoder.Decode(value); err != nil {
		return err
	}

	if err := decoder.Decode(&struct{}{}); err != io.EOF {
		if err == nil {
			return errors.New("invalid JSON: multiple top-level values")
		}
		return err
	}

	return nil
}
