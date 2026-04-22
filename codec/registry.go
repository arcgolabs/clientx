package codec

import (
	"errors"
	"strings"

	"github.com/DaiYuANg/arcgo/collectionx"
	"github.com/DaiYuANg/arcgo/collectionx/mapping"
	"github.com/samber/lo"
	"github.com/samber/mo"
	"github.com/samber/oops"
)

var (
	// ErrNilCodec indicates that a nil codec was provided.
	ErrNilCodec = errors.New("codec is nil")
	// ErrCodecNameEmpty indicates that a codec reported an empty name.
	ErrCodecNameEmpty = errors.New("codec name is empty")
	// ErrCodecExists indicates that a codec name is already registered.
	ErrCodecExists = errors.New("codec already exists")
	// ErrCodecNotFound indicates that a codec lookup failed.
	ErrCodecNotFound = errors.New("codec not found")
	// ErrUnsupportedValue indicates that a codec cannot handle the provided value type.
	ErrUnsupportedValue = errors.New("codec unsupported value type")
)

// Registry stores codecs by normalized name.
type Registry struct {
	codecs *mapping.ConcurrentMap[string, Codec]
}

// NewRegistry creates a Registry populated with codecs.
func NewRegistry(codecs ...Codec) *Registry {
	r := &Registry{codecs: mapping.NewConcurrentMap[string, Codec]()}
	lo.ForEach(codecs, func(c Codec, _ int) {
		mustRegisterCodec(r, c)
	})
	return r
}

// Register adds c to the registry under its normalized name.
func (r *Registry) Register(c Codec) error {
	if r == nil || r.codecs == nil {
		return oops.In("clientx/codec").
			With("op", "register_codec").
			New("codec registry is nil")
	}
	if c == nil {
		return oops.In("clientx/codec").
			With("op", "register_codec").
			Wrapf(ErrNilCodec, "validate codec")
	}
	name := strings.TrimSpace(strings.ToLower(c.Name()))
	if name == "" {
		return oops.In("clientx/codec").
			With("op", "register_codec").
			Wrapf(ErrCodecNameEmpty, "validate codec name")
	}

	if _, loaded := r.codecs.GetOrStore(name, c); loaded {
		return oops.In("clientx/codec").
			With("op", "register_codec", "codec", name).
			Wrapf(ErrCodecExists, "codec already exists")
	}
	return nil
}

// Get looks up a codec by name.
func (r *Registry) Get(name string) (Codec, bool) {
	if r == nil || r.codecs == nil {
		return nil, false
	}
	c, ok := r.codecs.Get(strings.TrimSpace(strings.ToLower(name)))
	return c, ok
}

// GetOption looks up a codec by name and returns an option-wrapped result.
func (r *Registry) GetOption(name string) mo.Option[Codec] {
	c, ok := r.Get(name)
	if !ok {
		return mo.None[Codec]()
	}
	return mo.Some(c)
}

// Must returns the named codec or panics when it is not registered.
func (r *Registry) Must(name string) Codec {
	c, ok := r.GetOption(name).Get()
	if !ok {
		panic(oops.In("clientx/codec").
			With("op", "must_get_codec", "codec", strings.TrimSpace(strings.ToLower(name))).
			Wrapf(ErrCodecNotFound, "codec not found"))
	}
	return c
}

// Names returns the registered codec names in sorted order.
func (r *Registry) Names() collectionx.List[string] {
	if r == nil || r.codecs == nil {
		return collectionx.NewList[string]()
	}
	names := collectionx.NewListWithCapacity[string](r.codecs.Len())
	r.codecs.Range(func(name string, _ Codec) bool {
		names.Add(name)
		return true
	})
	return names.Sort(func(left, right string) int {
		switch {
		case left < right:
			return -1
		case left > right:
			return 1
		default:
			return 0
		}
	})
}

var defaultRegistry = NewRegistry(
	JSON,
	Text,
	Bytes,
)

// Register adds c to the default registry.
func Register(c Codec) error {
	return defaultRegistry.Register(c)
}

// Get looks up a codec in the default registry.
func Get(name string) (Codec, bool) {
	return defaultRegistry.Get(name)
}

// GetOption looks up a codec in the default registry and returns an option-wrapped result.
func GetOption(name string) mo.Option[Codec] {
	return defaultRegistry.GetOption(name)
}

// Must returns a codec from the default registry or panics when not found.
func Must(name string) Codec {
	return defaultRegistry.Must(name)
}

// Names returns the codec names from the default registry.
func Names() collectionx.List[string] {
	return defaultRegistry.Names()
}

func mustRegisterCodec(r *Registry, c Codec) {
	if err := r.Register(c); err != nil {
		codecName := ""
		if c != nil {
			codecName = strings.TrimSpace(strings.ToLower(c.Name()))
		}
		panic(oops.In("clientx/codec").
			With("op", "register_builtin_codec", "codec", codecName).
			Wrapf(err, "register builtin codec"))
	}
}
