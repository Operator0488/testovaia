package plural

import "golang.org/x/text/language"

// Rules представляет собой набор правил множественного числа по тегу языка.
type Rules map[language.Tag]*Rule

// Rule rвозвращает ближайшее соответствующее правило множественного числа для тега языка или nil, если правило не найдено.
func (r Rules) Rule(tag language.Tag) *Rule {
	t := tag
	for {
		if rule := r[t]; rule != nil {
			return rule
		}
		t = t.Parent()
		if t.IsRoot() {
			break
		}
	}
	base, _ := tag.Base()
	baseTag, _ := language.Parse(base.String())
	return r[baseTag]
}
