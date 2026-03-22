package model

import (
	"database/sql/driver"
	"fmt"
	"time"
)

const timeFormat = "2006-01-02 15:04:05"

// LocalTime 自定义时间类型，JSON 序列化格式为 "2006-01-02 15:04:05"。
type LocalTime time.Time

func (t LocalTime) MarshalJSON() ([]byte, error) {
	stamp := fmt.Sprintf("\"%s\"", time.Time(t).Format(timeFormat))
	return []byte(stamp), nil
}

func (t *LocalTime) UnmarshalJSON(data []byte) error {
	if string(data) == "null" {
		return nil
	}
	parsed, err := time.ParseInLocation(`"`+timeFormat+`"`, string(data), time.Local)
	if err != nil {
		return err
	}
	*t = LocalTime(parsed)
	return nil
}

func (t LocalTime) Value() (driver.Value, error) {
	tt := time.Time(t)
	if tt.IsZero() {
		return nil, nil
	}
	return tt, nil
}

func (t *LocalTime) Scan(v interface{}) error {
	if v == nil {
		return nil
	}
	switch val := v.(type) {
	case time.Time:
		*t = LocalTime(val)
	default:
		return fmt.Errorf("cannot scan %T into LocalTime", v)
	}
	return nil
}
