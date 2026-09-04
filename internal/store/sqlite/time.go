package sqlite

import (
	"database/sql"
	"time"
)

func timeValue(value time.Time) int64 {
	return value.UnixMilli()
}

func timeFrom(value int64) time.Time {
	return time.UnixMilli(value).UTC()
}

func nullTime(value *time.Time) sql.NullInt64 {
	if value == nil {
		return sql.NullInt64{}
	}
	return sql.NullInt64{Int64: timeValue(*value), Valid: true}
}

func timePtr(value sql.NullInt64) *time.Time {
	if !value.Valid {
		return nil
	}
	result := timeFrom(value.Int64)
	return &result
}

func nullString(value string) sql.NullString {
	return sql.NullString{String: value, Valid: value != ""}
}

func boolValue(value bool) int64 {
	if value {
		return 1
	}
	return 0
}
