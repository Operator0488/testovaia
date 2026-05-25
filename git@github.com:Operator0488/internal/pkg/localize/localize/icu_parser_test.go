package localize

import (
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
)

type testICU struct {
	name        string
	rawString   string
	key         string
	message     *Message
	expectedErr error
}

func TestICU_Parser(t *testing.T) {
	tests := []testICU{
		{
			name:      "parse plural with person",
			rawString: "{value, plural,other {{person} and {second} has # cats}one {{person} and {second} has # cat}}",
			key:       "test",
			message: &Message{
				ID:    "test",
				One:   "{.person} and {.second} has {.value} cat",
				Other: "{.person} and {.second} has {.value} cats",
			},
		},
		{
			name:      "parse plural with special symbols",
			rawString: "{value, plural,\nother {{person} and {second} has # cats}\none {{person} and {second} has # cat}\n}",
			key:       "test",
			message: &Message{
				ID:    "test",
				One:   "{.person} and {.second} has {.value} cat",
				Other: "{.person} and {.second} has {.value} cats",
			},
		},
		{
			name:      "parse with params",
			rawString: "Mister {name}! Welcome",
			key:       "test",
			message: &Message{
				ID:    "test",
				Other: "Mister {.name}! Welcome",
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			message, err := parseICU(test.key, test.rawString)

			if (err != nil && test.expectedErr == nil) || (err == nil && test.expectedErr != nil) {
				t.Errorf("\nexpected error: %#v\n     got error: %#v", test.expectedErr, err)
			}
			if err != nil && test.expectedErr != nil && !assert.EqualError(t, err, test.expectedErr.Error()) {
				t.Errorf("\nexpected error: %#v\n     got error: %#v", test.expectedErr, err)
			}
			if !reflect.DeepEqual(message, test.message) {
				t.Errorf("\nexpected parse result %q; got %q", test.message, message)
			}

			if (message == nil) != (test.message == nil) {
				t.Errorf("\nexpected parse result %q; got %q", test.message, message)
				return
			}

			if message != nil {
				assert.Equal(t, test.message.ID, message.ID)
				assert.Equal(t, test.message.One, message.One)
				assert.Equal(t, test.message.Two, message.Two)
				assert.Equal(t, test.message.Few, message.Few)
				assert.Equal(t, test.message.Many, message.Many)
				assert.Equal(t, test.message.Other, message.Other)
			}
		})
	}
}
