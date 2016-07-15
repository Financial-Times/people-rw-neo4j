package people

import (
	"fmt"
	"github.com/spaolacci/murmur3"
	"github.com/ugorji/go/codec"
)

var handle = func() codec.JsonHandle {
	h := codec.JsonHandle{}
	h.Canonical = true
	return h
}()

func WriteHash(thing interface{}) (string, error) {
	h := murmur3.New128()
	enc := codec.NewEncoder(h, &handle)
	if err := enc.Encode(thing); err != nil {
		return "", err
	}
	return fmt.Sprintf("%x", h.Sum(nil)), nil
}
