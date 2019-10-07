package booksing

import (
	"regexp"
	"sort"
	"strings"
	"unicode"

	"github.com/antzucaro/matchr"
	"golang.org/x/text/runes"
	"golang.org/x/text/transform"
	"golang.org/x/text/unicode/norm"
)

var onlyLower = regexp.MustCompile("[^a-z]+")
var leadingNumbers = regexp.MustCompile("^ *[0-9]+")
var betweenParentheses = regexp.MustCompile(`\(.*\)`)
var betweenBockHooks = regexp.MustCompile(`\[.*\]`)

var leadingZeroes = regexp.MustCompile(`^ *(0)([0-9]+) `)
var alphaNumeric = regexp.MustCompile(`[^a-z0-9]+`)

//var year = regexp.MustCompile(`(19[0-9]{2})|(20[0-9]{2})`)

var uselessWords = []string{
	"le", "la", "et",
	"de", "het", "en",
	"the", "and", "a", "an",
}

func HashBook(author, title string) string {
	author = strings.ToLower(author)
	author = strings.Replace(author, "-", " ", -1)
	title = strings.ToLower(title)

	authorParts := strings.Split(author, " ")
	lastName := authorParts[len(authorParts)-1]

	//remove author from title
	title = strings.Replace(title, author, "", -1)
	title = strings.Replace(title, lastName, "", -1)

	//remove leading numbers
	title = leadingNumbers.ReplaceAllString(title, "")

	//concatenate to half further actions
	title = lastName + " " + title

	title = removeAccents(title)

	//make sure no whitespace is on either end
	title = strings.TrimSpace(title)

	//remove everything between parenthesis
	title = betweenParentheses.ReplaceAllString(title, " ")

	//remove everything between blockhooks
	title = betweenBockHooks.ReplaceAllString(title, " ")

	//remove ': a novel'
	title = strings.Replace(title, ": a novel", " ", -1)

	//remove leading zeroes from numbers
	title = leadingZeroes.ReplaceAllString(title, " $2 ")

	//remove all non [a-z0-9]
	title = alphaNumeric.ReplaceAllString(title, "")

	/*
		id := sha1.New()
		io.WriteString(id, author)
		io.WriteString(id, title)
		return hex.EncodeToString(id.Sum(nil))
	*/
	return title
}

func removeAccents(in string) string {
	t := transform.Chain(norm.NFD, runes.Remove(runes.In(unicode.Mn)), norm.NFC)
	s, _, err := transform.String(t, in)
	if err != nil {
		return in
	}
	return s
}

func generalizer(s string) string {
	s = " " + s + " "
	s = onlyLower.ReplaceAllString(strings.ToLower(s), " ")
	for _, w := range uselessWords {
		s = strings.Replace(s, " "+w+" ", " ", -1)
	}
	keys := GetMetaphoneKeys(s)
	s = strings.Join(keys, "")

	return s
}

func getLowercasedSlice(s string) []string {
	var returnParts []string
	parts := strings.Split(s, " ")
	for _, part := range parts {
		cleaned := alphaNumeric.ReplaceAllString(strings.ToLower(part), "")
		returnParts = append(returnParts, cleaned)
	}

	returnParts = unique(returnParts)
	sort.Strings(returnParts)

	return returnParts
}

func GetMetaphoneKeys(s string) []string {
	parts := metaphonify(s)

	parts = unique(parts)

	sort.Strings(parts)

	return parts
}

func metaphonify(s string) []string {
	var nameParts []string
	names := strings.Split(s, " ")
	for _, name := range names {
		cleaned := onlyLower.ReplaceAllString(strings.ToLower(name), "")
		a, _ := matchr.DoubleMetaphone(cleaned)
		if len(a) >= 1 {
			nameParts = append(nameParts, a)
		}
	}
	sort.Strings(nameParts)
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
