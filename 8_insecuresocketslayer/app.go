package insecuresocketslayer

import (
	"strconv"
	"strings"
)

// msg contains a comma-separated list of toys to make, like so:
// 10x toy car,15x dog on a string,4x inflatable motorcycle
//
// Find out which toy from the request they need to make the most copies of, like so:
// 15x dog on a string
func findMaxToy(msg string) string {
	toys := strings.Split(msg, ",")
	var maxToy string
	var maxCount int
	for _, toy := range toys {
		parts := strings.Split(toy, "x ")
		count, err := strconv.Atoi(parts[0])
		if err != nil {
			return ""
		}
		if count > maxCount {
			maxCount = count
			maxToy = toy
		}
	}
	return maxToy
}
