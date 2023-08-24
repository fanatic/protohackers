package jobcentre

import (
	"fmt"
	"sort"
)

func InsertIntoJobPrioritySliceSorted(s []*Job, j *Job) []*Job {
	i := sort.Search(len(s), func(i int) bool { return s[i].Priority >= j.Priority })
	s = append(s, nil) // increase slice size by 1
	copy(s[i+1:], s[i:])
	s[i] = j
	return s
}

func HighestPriorityJob(s []*Job, q []string) *Job {
	for i := len(s) - 1; i >= 0; i-- {
		for _, queue := range q {
			if s[i].Queue == queue {
				return s[i]
			}
		}
	}
	return nil
}

func RemoveFromJobPrioritySlice(s []*Job, j Job) []*Job {
	i := sort.Search(len(s), func(i int) bool { return s[i].Priority >= j.Priority })
	for ; i < len(s); i++ {
		if s[i].ID == j.ID {
			break
		}
	}
	if i == len(s) {
		fmt.Printf("Job not found in slice\n")
	} else if i == len(s)-1 {
		s = s[:i]
	} else {
		s = append(s[:i], s[i+1:]...)
	}
	return s
}
