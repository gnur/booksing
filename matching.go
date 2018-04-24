package main

import (
	"log"
	"regexp"
	"sort"
	"strings"

	"github.com/antzucaro/matchr"
)

func getLowercasedSlice(s string) []string {
	reg, _ := regexp.Compile("[^a-z]+")
	var returnParts []string
	parts := strings.Split(s, " ")
	for _, part := range parts {
		cleaned := reg.ReplaceAllString(strings.ToLower(part), "")
		returnParts = append(returnParts, cleaned)
	}

	log.Println(len(parts), parts)
	returnParts = unique(returnParts)
	sort.Strings(returnParts)
	log.Println(len(parts), parts)

	return returnParts
}

func getMetaphoneKeys(s string) []string {
	parts := metaphonify(s)

	parts = unique(parts)

	sort.Strings(parts)

	return parts
}

func metaphonify(s string) []string {
	var nameParts []string
	names := strings.Split(s, " ")
	for _, name := range names {
		a, _ := matchr.DoubleMetaphone(name)
		if len(a) >= 1 {
			nameParts = append(nameParts, a)
		}
	}
	return nameParts
}

func unique(input []string) []string {
	u := make([]string, 0, len(input))
	m := make(map[string]bool)

	for _, val := range input {
		if _, ok := m[val]; !ok {
			m[val] = true
			u = append(u, val)
		}
	}

	return u
}
