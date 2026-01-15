package utils

import "strings"

type StringSlice struct {
	values []string
}

func (s *StringSlice) String() string {
	return strings.Join(s.values, ",")
}

func (s *StringSlice) Set(value string) error {
	if value == "" {
		return nil
	}
	s.values = append(s.values, value)
	return nil
}

func (s *StringSlice) Values() []string {
	return append([]string(nil), s.values...)
}
