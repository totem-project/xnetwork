package xnetwork

import (
	"time"
)

//‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾
// Request struct
//_______________________________________________________________________

type Request struct {
	Network string
	Address string
	IsTLS   bool
	Raw     []byte
	sendAt  time.Time
}

func (r *Request) GetRaw() []byte {
	return r.Raw
}
