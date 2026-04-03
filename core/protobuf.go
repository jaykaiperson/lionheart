package core

import "strings"

func PbVar(d []byte, o int) (uint64, int) {
	var v uint64
	for s := 0; o < len(d) && s < 64; s += 7 {
		b := d[o]
		o++
		v |= uint64(b&0x7f) << s
		if b < 0x80 {
			return v, o
		}
	}
	return 0, o
}

func PbAll(d []byte, f uint64) (r [][]byte) {
	for o := 0; o < len(d); {
		t, n := PbVar(d, o)
		if n == o {
			break
		}
		o = n
		switch t & 7 {
		case 0:
			_, o = PbVar(d, o)
		case 2:
			l, n := PbVar(d, o)
			o = n
			e := o + int(l)
			if e > len(d) || e < o {
				return
			}
			if t>>3 == f {
				r = append(r, d[o:e])
			}
			o = e
		case 1:
			o += 8
		case 5:
			o += 4
		default:
			return
		}
	}
	return
}

func PbStr(d []byte, f uint64) string {
	if a := PbAll(d, f); len(a) > 0 {
		return string(a[0])
	}
	return ""
}

// PbICE extracts TURN/STUN credentials from a protobuf message.
// WB Stream uses field 5 for ICE servers (standard LiveKit uses field 9).
func PbICE(d []byte) (res []TurnCred) {
	for o := 0; o < len(d); {
		t, n := PbVar(d, o)
		if n == o {
			break
		}
		o = n
		switch t & 7 {
		case 0:
			_, o = PbVar(d, o)
		case 2:
			l, n := PbVar(d, o)
			o = n
			e := o + int(l)
			if e > len(d) || e < o {
				return
			}
			inner := d[o:e]
			for _, f := range []uint64{5, 9} {
				for _, blk := range PbAll(inner, f) {
					urls := PbAll(blk, 1)
					hit := false
					for _, u := range urls {
						s := string(u)
						if strings.HasPrefix(s, "turn") || strings.HasPrefix(s, "stun") {
							hit = true
							break
						}
					}
					if !hit {
						continue
					}
					un, pw := PbStr(blk, 2), PbStr(blk, 3)
					for _, u := range urls {
						res = append(res, TurnCred{string(u), un, pw})
					}
					for _, blk2 := range PbAll(inner, f) {
						if len(blk2) > 0 && len(blk) > 0 && &blk2[0] == &blk[0] {
							continue
						}
						u2, p2 := PbStr(blk2, 2), PbStr(blk2, 3)
						for _, u := range PbAll(blk2, 1) {
							res = append(res, TurnCred{string(u), u2, p2})
						}
					}
					return
				}
			}
			o = e
		case 1:
			o += 8
		case 5:
			o += 4
		default:
			return
		}
	}
	return
}
