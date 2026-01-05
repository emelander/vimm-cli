package cli

import "strings"

type countFlag struct {
	value *int
}

func (c *countFlag) String() string {
	if c == nil || c.value == nil {
		return "0"
	}
	return intToString(*c.value)
}

func (c *countFlag) Set(_ string) error {
	if c != nil && c.value != nil {
		*c.value++
	}
	return nil
}

type stringSliceFlag struct {
	values *[]string
}

func (s *stringSliceFlag) String() string {
	if s == nil || s.values == nil {
		return ""
	}
	return strings.Join(*s.values, ",")
}

func (s *stringSliceFlag) Set(val string) error {
	if s != nil && s.values != nil {
		*s.values = append(*s.values, val)
	}
	return nil
}

func intToString(val int) string {
	if val == 0 {
		return "0"
	}
	negative := val < 0
	if negative {
		val = -val
	}
	var buf [20]byte
	i := len(buf)
	for val > 0 {
		i--
		buf[i] = byte('0' + val%10)
		val /= 10
	}
	if negative {
		i--
		buf[i] = '-'
	}
	return string(buf[i:])
}
