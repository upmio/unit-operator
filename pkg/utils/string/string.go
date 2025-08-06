package string

import (
	"math/rand"
	"sort"
	"strings"
	"time"
)

func ContainsStringV2(target string, str_array []string) bool {
	sort.Strings(str_array)
	index := sort.SearchStrings(str_array, target)
	if index < len(str_array) && str_array[index] == target {
		return true
	}
	return false
}

func StringsDiffFunc(ps1, ps2 []string) []string {
	sort.Strings(ps1)
	sort.Strings(ps2)
	diff := make([]string, 0, 50)
	for _, p1 := range ps1 {
		found := false
		for _, p2 := range ps2 {
			if strings.Trim(p1, " ") == strings.Trim(p2, " ") {
				found = true
			}
		}
		if !found {
			diff = append(diff, strings.Trim(p1, " "))
		}
	}
	return diff
}

var letters = []rune("0123456789abcdefghijklmnopqrstuvwxyz")

// RandSeq generate random combinations of lowercase letters and numbers
func RandSeq(n int) string {
	b := make([]rune, n)
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	for i := range b {
		b[i] = letters[r.Intn(36)]
	}
	return string(b)
}

func CompareStringSlice(old []string, new []string) bool {
	if len(old) != len(new) {
		return false
	}
	sort.Strings(old)
	sort.Strings(new)
	for i, v := range old {
		if v != new[i] {
			return false
		}
	}

	return true
}
