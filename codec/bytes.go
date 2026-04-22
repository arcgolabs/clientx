package codec

import (
	"fmt"

	"github.com/samber/oops"
)

type bytesCodec struct{}

func (c bytesCodec) Name() string {
	return "bytes"
}

func (c bytesCodec) Marshal(v any) ([]byte, error) {
	switch value := v.(type) {
	case []byte:
		return append([]byte(nil), value...), nil
	default:
		return nil, oops.In("clientx/codec").
			With("op", "marshal_bytes", "codec", c.Name(), "value_type", fmt.Sprintf("%T", v)).
			Wrapf(ErrUnsupportedValue, "marshal bytes value")
	}
}

func (c bytesCodec) Unmarshal(data []byte, v any) error {
	target, ok := v.(*[]byte)
	if !ok {
		return oops.In("clientx/codec").
			With("op", "unmarshal_bytes", "codec", c.Name(), "target_type", fmt.Sprintf("%T", v), "payload_size", len(data)).
			Wrapf(ErrUnsupportedValue, "unmarshal bytes value")
	}
	*target = append((*target)[:0], data...)
	return nil
}

// Bytes is the built-in codec for raw byte slices.
var Bytes Codec = bytesCodec{}
