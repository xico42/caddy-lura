package caddylura

import (
	"regexp"
	"strings"
)

var (
	simpleURLKeysPattern     = regexp.MustCompile(`\{([\w\-.:/]+)}`)
	caddyPlaceholdersPattern = regexp.MustCompile(`({{\.Resp042_\.(.+?)}})`)
)

type paramsSet map[string]interface{}

func newParamsSetFromPattern(subject string) paramsSet {
	matches := simpleURLKeysPattern.FindAllStringSubmatch(subject, -1)
	kset := make(map[string]interface{}, len(matches))
	for _, v := range matches {
		kset[v[1]] = nil
	}
	return kset
}

func (s paramsSet) contains(v string) bool {
	_, ok := s[v]
	return ok
}

func processBackendUrlPattern(subject string, backendParams paramsSet) string {
	output := subject

	for p := range backendParams {
		output = strings.ReplaceAll(output, "{"+p+"}", "{resp042_."+p+"}")
	}

	return output
}

func applyCaddyPlaceholders(subject string) string {
	return caddyPlaceholdersPattern.ReplaceAllString(subject, "{${2}}")
}
