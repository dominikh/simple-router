package netdb

// #include <netdb.h>
import "C"

import (
	"errors"
	"unsafe"
)

type Protoent struct {
	Name    string
	Aliases []string
	Number  int
}

func (this Protoent) Equal(other Protoent) bool {
	return this.Number == other.Number
}

func cprotoentToProtoent(s *C.struct_protoent) Protoent {
	var aliases []string
	p := s.p_aliases
	q := uintptr(unsafe.Pointer(p))
	for {
		p = (**C.char)(unsafe.Pointer(q))
		if *p == nil {
			break
		}
		aliases = append(aliases, C.GoString(*p))
		q += unsafe.Sizeof(q)
	}

	return Protoent{
		Name:    C.GoString(s.p_name),
		Aliases: aliases,
		Number:  int(s.p_proto),
	}
}

func GetProtoByNumber(num int) (Protoent, error) {
	s := C.getprotobynumber(C.int(num))
	if s == nil {
		return Protoent{}, errors.New("Unknown protocol number")
	}

	return cprotoentToProtoent(s), nil
}

func GetProtoByName(name string) (Protoent, error) {
	s := C.getprotobyname(C.CString(name))
	if s == nil {
		return Protoent{}, errors.New("Unknown protocol name")
	}

	return cprotoentToProtoent(s), nil
}
