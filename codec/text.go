package codec

import (
	"encoding"
	"fmt"

	"github.com/samber/oops"
)

type textCodec struct{}

func (c textCodec) Name() string {
	return "text"
}

func (c textCodec) Marshal(v any) ([]byte, error) {
	switch value := v.(type) {
	case string:
		return []byte(value), nil
	case []byte:
		return append([]byte(nil), value...), nil
	case encoding.TextMarshaler:
		data, err := value.MarshalText()
		if err != nil {
			return nil, oops.In("clientx/codec").
				With("op", "marshal_text", "codec", c.Name(), "value_type", fmt.Sprintf("%T", v)).
				Wrapf(err, "marshal text value")
		}
		return data, nil
	case fmt.Stringer:
		return []byte(value.String()), nil
	default:
		return nil, oops.In("clientx/codec").
			With("op", "marshal_text", "codec", c.Name(), "value_type", fmt.Sprintf("%T", v)).
			Wrapf(ErrUnsupportedValue, "marshal text value")
	}
}

func (c textCodec) Unmarshal(data []byte, v any) error {
	switch target := v.(type) {
	case *string:
		*target = string(data)
		return nil
	case *[]byte:
		*target = append((*target)[:0], data...)
		return nil
	case encoding.TextUnmarshaler:
		if err := target.UnmarshalText(data); err != nil {
			return oops.In("clientx/codec").
				With("op", "unmarshal_text", "codec", c.Name(), "target_type", fmt.Sprintf("%T", v), "payload_size", len(data)).
				Wrapf(err, "unmarshal text value")
		}
		return nil
	default:
		return oops.In("clientx/codec").
			With("op", "unmarshal_text", "codec", c.Name(), "target_type", fmt.Sprintf("%T", v), "payload_size", len(data)).
			Wrapf(ErrUnsupportedValue, "unmarshal text value")
	}
}

// Text is the built-in text codec.
var Text Codec = textCodec{}
