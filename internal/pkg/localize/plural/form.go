package plural

// http://cldr.unicode.org/index/cldr-spec/plural-rules
type Form string

// Все доступные формы множественного числа.
const (
	Invalid Form = ""
	Zero    Form = "zero"
	One     Form = "one"
	Two     Form = "two"
	Few     Form = "few"
	Many    Form = "many"
	Other   Form = "other"
)
