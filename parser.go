package main

import (
	"fmt"
	"reflect"
	"strings"
	"time"

	"github.com/araddon/dateparse"
	"github.com/mitchellh/mapstructure"
)

// ParseTime will parse an arbitrary date string and try to create a time.Time.
func ParseTime(date string) (time.Time, error) {
	if strings.TrimSpace(date) == "" {
		return time.Unix(0, 0), nil
	}
	return dateparse.ParseAny(date)
}

func parseResult(input interface{}) (*Book, error) {
	var out Book
	config := mapstructure.DecoderConfig{
		DecodeHook: func(
			f reflect.Type,
			t reflect.Type,
			data interface{}) (interface{}, error) {
			if t == reflect.TypeOf(time.Time{}) && f == reflect.TypeOf("") {
				return ParseTime(data.(string))
			}

			return data, nil
		},
		Result: &out,
	}

	decoder, err := mapstructure.NewDecoder(&config)
	if err != nil {
		return nil, fmt.Errorf("creating decoder failed with error %w", err)
	}
	if err := decoder.Decode(input); err != nil {
		return nil, fmt.Errorf("decoding failed with error %w", err)
	}
	return &out, nil
}
