// copy and modify from https://github.com/knadh/koanf/blob/master/providers/basicflag/basicflag.go
package config

import (
	"errors"
	"flag"

	"github.com/knadh/koanf/maps"
)

type pflag struct {
	delim   string
	flagset *flag.FlagSet
	cb      func(key string, value string) (string, interface{})
}

func providerWithValue(f *flag.FlagSet, delim string, cb func(key string, value string) (string, interface{})) *pflag {
	return &pflag{
		flagset: f,
		delim:   delim,
		cb:      cb,
	}
}

// Read reads the flag variables and returns a nested conf map.
func (p *pflag) Read() (map[string]interface{}, error) {
	mp := make(map[string]interface{})
	p.flagset.Visit(func(f *flag.Flag) {
		var (
			key             = f.Name
			val interface{} = f.Value.String()
		)
		if p.cb != nil {
			k, v := p.cb(f.Name, f.Value.String())
			// If the callback blanked the key, it should be omitted
			if k == "" {
				return
			}

			key = k
			val = v
		}
		mp[key] = val
	})
	return maps.Unflatten(mp, p.delim), nil
}

// ReadBytes is not supported by the basicflag koanf.
func (p *pflag) ReadBytes() ([]byte, error) {
	return nil, errors.New("basicflag provider does not support this method")
}
