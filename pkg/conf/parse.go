package conf

import (
	"fmt"
	"github.com/go-viper/mapstructure/v2"
	"io"
	"os"
	"reflect"
	_ "unsafe"

	"github.com/knadh/koanf/parsers/yaml"
	"github.com/knadh/koanf/providers/confmap"
	"github.com/knadh/koanf/providers/file"
	"github.com/knadh/koanf/providers/rawbytes"
	"github.com/knadh/koanf/v2"
)

const (
	// KeyDelimiter is used as the default key delimiter in the default koanf instance.
	KeyDelimiter = "."
)

// NewParser creates a new empty Parser instance.
func NewParser() *Parser {
	k := koanf.NewWithConf(koanf.Conf{Delim: KeyDelimiter, StrictMerge: false})
	return NewParserFromOperator(k)
}

// NewParserFromFile creates a new Parser by reading the given file.
func NewParserFromFile(fileName string) (*Parser, error) {
	return NewParserFromProvider(file.Provider(fileName))
}

// NewParserFromBuffer creates a new Parser by reading the given yaml buffer.
func NewParserFromBuffer(buf io.Reader) (*Parser, error) {
	content, err := io.ReadAll(buf)
	if err != nil {
		return nil, err
	}
	return NewParserFromProvider(rawbytes.Provider(content))
}

// NewParserFromStringMap creates a parser from a map[string]any.
func NewParserFromStringMap(data map[string]any) *Parser {
	p := NewParser()
	// Cannot return error because the koanf instance is empty.
	_ = p.k.Load(confmap.Provider(data, KeyDelimiter), nil)
	return p
}

// NewParserFromProvider creates a new Parser by the given provider.
func NewParserFromProvider(provider koanf.Provider) (*Parser, error) {
	p := NewParser()
	err := p.k.Load(provider, yaml.Parser())
	return p, err
}

// NewParserFromOperator creates a parser from a koanf.Koanf.
func NewParserFromOperator(k *koanf.Koanf) *Parser {
	return &Parser{k: k}
}

func fsProviderWithEnv(path string) (koanf.Provider, error) {
	fs, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	return rawbytes.Provider(ParseEnv(fs)), nil
}

// Parser loads configuration.
type Parser struct {
	k *koanf.Koanf
}

// Operator returns the koanf instance.
func (l *Parser) Operator() *koanf.Koanf {
	return l.k
}

// AllKeys returns all keys holding a value, regardless of where they are set.
// Nested keys are returned with a KeyDelimiter separator.
func (l *Parser) AllKeys() []string {
	return l.k.Keys()
}

// Unmarshal specified path config into a struct.
// Tags on the fields of the structure must be properly set.
func (l *Parser) Unmarshal(key string, dst any) (err error) {
	var input any
	if key == "" {
		input = l.ToStringMap()
	} else {
		input = l.Get(key)
	}

	decoder, err := mapstructure.NewDecoder(decoderConfig(dst))
	if err != nil {
		return err
	}
	return decoder.Decode(input)
}

// UnmarshalExact unmarshals the config into a struct, error if a field is nonexistent.
func (l *Parser) UnmarshalExact(key string, intoCfg any) (err error) {
	var input any
	if key == "" {
		input = l.ToStringMap()
	} else {
		input = l.Get(key)
	}
	dc := decoderConfig(intoCfg)
	dc.ErrorUnused = true
	decoder, err := mapstructure.NewDecoder(dc)
	if err != nil {
		return err
	}
	return decoder.Decode(input)
}

// Get can retrieve any value given the key to use.
func (l *Parser) Get(key string) any {
	return l.k.Get(key)
}

// Set sets the value for the key.
func (l *Parser) Set(key string, value any) {
	if err := l.k.Set(key, value); err != nil {
		panic(err)
	}
}

// IsSet checks to see if the key has been set in any of the data locations.
// IsSet is case-insensitive for a key.
func (l *Parser) IsSet(key string) bool {
	return l.k.Exists(key)
}

// LoadFileWithEnv loads the given file and env, and merges it into the config.
func (l *Parser) LoadFileWithEnv(path string) error {
	provider, err := fsProviderWithEnv(path)
	if err != nil {
		return err
	}
	return l.k.Load(provider, yaml.Parser())
}

// LoadProviderWithEnv loads the given provider and env, and merges it into the config.
func (l *Parser) LoadProviderWithEnv(provider koanf.Provider) error {
	bs, err := provider.ReadBytes()
	if err != nil {
		return err
	}
	return l.k.Load(rawbytes.Provider(ParseEnv(bs)), yaml.Parser())
}

// MergeStringMap merges the configuration from the given map with the existing config.
// Note that the given map may be modified.
func (l *Parser) MergeStringMap(cfg map[string]any) error {
	toMerge := koanf.New(KeyDelimiter)
	if err := toMerge.Load(confmap.Provider(cfg, KeyDelimiter), nil); err != nil {
		return err
	}
	return l.k.Merge(toMerge)
}

// Sub returns new Parser instance representing a sub-config of this instance.
// It returns an error is the sub-config is not a map (use Get()) or if none exists.
func (l *Parser) Sub(key string) (*Parser, error) {
	if !l.IsSet(key) {
		return nil, fmt.Errorf("key not exists:%s", key)
	}

	subParser := NewParser()
	subParser.k = l.k.Cut(key)

	if len(subParser.ToStringMap()) == 0 {
		if l.Get(key) != nil {
			return nil, fmt.Errorf("key is not a map")
		}
	}

	return subParser, nil
}

// ToStringMap creates a map[string]any from a Parser.
func (l *Parser) ToStringMap() map[string]any {
	return l.k.Raw()
}

// ToBytes takes a Parser implementation and marshals the config map into bytes,
// for example, to TOML or JSON bytes.
func (l *Parser) ToBytes(p koanf.Parser) ([]byte, error) {
	return l.k.Marshal(p)
}

//go:linkname textUnmarshalHookFunc github.com/knadh/koanf/v2.textUnmarshalerHookFunc
func textUnmarshalHookFunc() mapstructure.DecodeHookFuncType

// decoderConfig returns a default mapstructure.DecoderConfig capable of parsing time.Duration
// and weakly converting config field values to primitive types.  It also ensures that maps
// whose values are nil pointer structs resolved to the zero value of the target struct (see
// expandNilStructPointers). A decoder created from this mapstructure.DecoderConfig will decode
// its contents to the result argument.
func decoderConfig(result any) *mapstructure.DecoderConfig {
	return &mapstructure.DecoderConfig{
		Result:           result,
		Metadata:         nil,
		TagName:          "json",
		Squash:           true,
		WeaklyTypedInput: true,
		DecodeHook: mapstructure.ComposeDecodeHookFunc(
			expandNilStructPointers(),
			textUnmarshalHookFunc(),
			mapstructure.StringToTimeDurationHookFunc(),
			mapstructure.StringToSliceHookFunc(","),
		),
	}
}

// In cases where a config has a mapping of something to a struct pointers
// we want nil values to resolve to a pointer to the zero value of the
// underlying struct just as we want nil values of a mapping of something
// to a struct to resolve to the zero value of that struct.
//
// e.g. given a config type:
// type Configuration struct { Thing *SomeStruct `mapstructure:"thing"` }
//
// and yaml of:
// config:
//
//	thing:
//
// we want an unmarshalled Configuration to be equivalent to
// Configuration{Thing: &SomeStruct{}} instead of Configuration{Thing: nil}
func expandNilStructPointers() mapstructure.DecodeHookFunc {
	return func(from reflect.Value, to reflect.Value) (any, error) {
		// ensure we are dealing with map to map comparison
		if from.Kind() == reflect.Map && to.Kind() == reflect.Map {
			toElem := to.Type().Elem()
			// ensure that map values are pointers to a struct
			// (that may be nil and require manual setting w/ zero value)
			if toElem.Kind() == reflect.Ptr && toElem.Elem().Kind() == reflect.Struct {
				fromRange := from.MapRange()
				for fromRange.Next() {
					fromKey := fromRange.Key()
					fromValue := fromRange.Value()
					// ensure that we've run into a nil pointer instance
					if fromValue.IsNil() {
						newFromValue := reflect.New(toElem.Elem())
						from.SetMapIndex(fromKey, newFromValue)
					}
				}
			}
		}
		return from.Interface(), nil
	}
}
