package main

import (
	"log"
	"strings"
)

type errorVal struct {
	errmsg string
}

func (e *errorVal) Error() string {
	return e.errmsg
}

func copyMapCount(src map[string][]int32) map[string]int {
	if src == nil {
		return nil
	}
	ret := make(map[string]int)
	for k, v := range src {
		ret[k] = len(v)
	}
	return ret
}

func filter(input []string, f func(string) bool) []string {
	ret := make([]string, 0)
	for _, element := range input {
		if f(element) {
			ret = append(ret, strings.TrimSpace(element))
		}
	}
	return ret
}

func get_list_from_string(input string, separator string, failed_msg string) []string {
	out_list := strings.Split(input, separator)
	out_list = filter(out_list, func(inp string) bool {
		return strings.TrimSpace(inp) != ""
	})
	if len(out_list) == 0 {
		log.Printf("Error: %s", failed_msg)
	}
	return out_list
}

func unique_list(input []string) (ret []string) {
	for _, el := range input {
		if in(el, ret) {
			continue
		}
		ret = append(ret, el)
	}
	return
}

func in(input string, els []string) (found bool) {
	for _, el := range els {
		if el == input {
			found = true
			break
		}
	}
	return
}

func get_group_map(input *arrayFlags, failed_msg string) (ret map[string][]string) {
	els := input.GetEls()
	if len(els) == 0 {
		log.Fatal(failed_msg)
	}
	ret = make(map[string][]string)
	for _, el := range els {
		vals := get_list_from_string(el, ":", "some problem")
		if len(vals) != 2 {
			log.Fatalf("Invalid value for groups: %s", el)
		}
		group := vals[0]
		topics := unique_list(get_list_from_string(vals[1], ",", "some problem"))
		if len(topics) == 0 {
			log.Fatalf("Invalid value for groups: %s", el)
		}
		grptopics := ret[group]
		if grptopics == nil {
			grptopics = make([]string, 0)
		}
		grptopics = append(grptopics, topics...)
		ret[group] = unique_list(grptopics)
	}
	return
}

func check_input_str(input string, failed_msg string) string {
	out := strings.TrimSpace(input)
	if "" == out {
		log.Fatalf("Error: %s", failed_msg)
	}
	return out
}

func check_error(msg string, err error) {
	if err != nil {
		log.Printf("%s: %v", msg, err)
	}
}

func check_error_exit(msg string, err error) {
	if err != nil {
		log.Fatalf("%s: %v", msg, err)
	}
}
