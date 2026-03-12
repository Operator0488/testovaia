package logger

import (
	"time"

	"go.uber.org/zap"
)

type Field = zap.Field

// Err Создает переменную лога с типом error. Метод не принимает аргумента с именем переменной, так как ошибка уже
// имеет стандартное именование для поля ("error")
func Err(err error) Field {
	return zap.Error(err)
}

// Any Создает переменную лога с автоматическим определением типа. Использовать данный метод следует с осторожностью,
// так как под капотом используется рефлексия, что может приводить к снижению производительности.
func Any(key string, value interface{}) Field {
	return zap.Any(key, value)
}

// Binary Создает переменную лога с типом Binary
func Binary(key string, val []byte) Field {
	return zap.Binary(key, val)
}

// Bool Создает переменную лога с типом bool
func Bool(key string, val bool) Field {
	return zap.Bool(key, val)
}

// Boolp Создает переменную лога с типом указателя на bool. Может быть безопасно использован, даже если указатель
// не был инициализирован
func Boolp(key string, val *bool) Field {
	return zap.Boolp(key, val)
}

// ByteString Создает переменную лога с типом ByteString
func ByteString(key string, val []byte) Field {
	return zap.ByteString(key, val)
}

// Float64 Создает переменную лога с типом float64
func Float64(key string, val float64) Field {
	return zap.Float64(key, val)
}

// Float64p Создает переменную лога с типом указателя на float64. Может быть безопасно использован, даже если указатель
// не был инициализирован
func Float64p(key string, val *float64) Field {
	return zap.Float64p(key, val)
}

// Float32 Создает переменную лога с типом float32
func Float32(key string, val float32) Field {
	return zap.Float32(key, val)
}

// Float32p Создает переменную лога с типом указателя на float32. Может быть безопасно использован, даже если указатель
// не был инициализирован
func Float32p(key string, val *float32) Field {
	return zap.Float32p(key, val)
}

// Int Создает переменную лога с типом int
func Int(key string, val int) Field {
	return zap.Int(key, val)
}

// Intp Создает переменную лога с типом указателя на int. Может быть безопасно использован, даже если указатель
// не был инициализирован
func Intp(key string, val *int) Field {
	return zap.Intp(key, val)
}

// Int64 Создает переменную лога с типом int64
func Int64(key string, val int64) Field {
	return zap.Int64(key, val)
}

// Int64p Создает переменную лога с типом указателя на int64. Может быть безопасно использован, даже если указатель
// не был инициализирован
func Int64p(key string, val *int64) Field {
	return zap.Int64p(key, val)
}

// Int32 Создает переменную лога с типом int32
func Int32(key string, val int32) Field {
	return zap.Int32(key, val)
}

// Int32p Создает переменную лога с типом указателя на int32. Может быть безопасно использован, даже если указатель
// не был инициализирован
func Int32p(key string, val *int32) Field {
	return zap.Int32p(key, val)
}

// Int16 Создает переменную лога с типом int16
func Int16(key string, val int16) Field {
	return zap.Int16(key, val)
}

// Int16p Создает переменную лога с типом указателя на int16. Может быть безопасно использован, даже если указатель
// не был инициализирован
func Int16p(key string, val *int16) Field {
	return zap.Int16p(key, val)
}

// Int8 Создает переменную лога с типом int8
func Int8(key string, val int8) Field {
	return zap.Int8(key, val)
}

// Int8p Создает переменную лога с типом указателя на int8. Может быть безопасно использован, даже если указатель
// не был инициализирован
func Int8p(key string, val *int8) Field {
	return zap.Int8p(key, val)
}

// String Создает переменную лога с типом String
func String(key string, val string) Field {
	return zap.String(key, val)
}

// Stringp Создает переменную лога с типом указателя на string. Может быть безопасно использован, даже если указатель
// не был инициализирован
func Stringp(key string, val *string) Field {
	return zap.Stringp(key, val)
}

// Uint Создает переменную лога с типом uint
func Uint(key string, val uint) Field {
	return zap.Uint(key, val)
}

// Uintp Создает переменную лога с типом указателя на uint. Может быть безопасно использован, даже если указатель
// не был инициализирован
func Uintp(key string, val *uint) Field {
	return zap.Uintp(key, val)
}

// Uint64 Создает переменную лога с типом uint64
func Uint64(key string, val uint64) Field {
	return zap.Uint64(key, val)
}

// Uint64p Создает переменную лога с типом указателя на uint64. Может быть безопасно использован, даже если указатель
// не был инициализирован
func Uint64p(key string, val *uint64) Field {
	return zap.Uint64p(key, val)
}

// Uint32 Создает переменную лога с типом uint32
func Uint32(key string, val uint32) Field {
	return zap.Uint32(key, val)
}

// Uint32p Создает переменную лога с типом указателя на uint32. Может быть безопасно использован, даже если указатель
// не был инициализирован
func Uint32p(key string, val *uint32) Field {
	return zap.Uint32p(key, val)
}

// Uint16 Создает переменную лога с типом uint16
func Uint16(key string, val uint16) Field {
	return zap.Uint16(key, val)
}

// Uint16p Создает переменную лога с типом указателя на uint16. Может быть безопасно использован, даже если указатель
// не был инициализирован
func Uint16p(key string, val *uint16) Field {
	return zap.Uint16p(key, val)
}

// Uint8 Создает переменную лога с типом uint8
func Uint8(key string, val uint8) Field {
	return zap.Uint8(key, val)
}

// Uint8p Создает переменную лога с типом указателя на uint8. Может быть безопасно использован, даже если указатель
// не был инициализирован
func Uint8p(key string, val *uint8) Field {
	return zap.Uint8p(key, val)
}

// Time Создает переменную лога с типом Time
func Time(key string, val time.Time) Field {
	return zap.Time(key, val)
}

// Timep Создает переменную лога с типом указателя на Time. Может быть безопасно использован, даже если указатель
// не был инициализирован
func Timep(key string, val *time.Time) Field {
	return zap.Timep(key, val)
}

// Duration Создает переменную лога с типом Duration
func Duration(key string, val time.Duration) Field {
	return zap.Duration(key, val)
}

// Durationp Создает переменную лога с типом указателя на Duration. Может быть безопасно использован,
// даже если указатель не был инициализирован
func Durationp(key string, val *time.Duration) Field {
	return zap.Durationp(key, val)
}
