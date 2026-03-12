package redis

import "encoding/json"

type codec interface {
	Marshal(v any) ([]byte, error)
	Unmarshal(b []byte, v any) error
}

type JSONCodec struct{}

func (JSONCodec) Marshal(v any) ([]byte, error) {
	return json.Marshal(v)
}

func (JSONCodec) Unmarshal(b []byte, v any) error {
	return json.Unmarshal(b, v)
}
