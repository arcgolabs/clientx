package codec

// Codec marshals and unmarshals values for transport payloads.
type Codec interface {
	Name() string
	Marshal(v any) ([]byte, error)
	Unmarshal(data []byte, v any) error
}
