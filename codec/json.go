package codec

import (
	"encoding/json"
	"fmt"

	"github.com/samber/oops"
)

type jsonCodec struct{}

func (c jsonCodec) Name() string {
	return "json"
}

func (c jsonCodec) Marshal(v any) ([]byte, error) {
	data, err := json.Marshal(v)
	if err != nil {
		return nil, oops.In("clientx/codec").
			With("op", "marshal_json", "codec", c.Name(), "value_type", fmt.Sprintf("%T", v)).
			Wrapf(err, "marshal json value")
	}
	return data, nil
}

func (c jsonCodec) Unmarshal(data []byte, v any) error {
	if err := json.Unmarshal(data, v); err != nil {
		return oops.In("clientx/codec").
			With("op", "unmarshal_json", "codec", c.Name(), "target_type", fmt.Sprintf("%T", v), "payload_size", len(data)).
			Wrapf(err, "unmarshal json value")
	}
	return nil
}

// JSON is the built-in JSON codec.
var JSON Codec = jsonCodec{}
