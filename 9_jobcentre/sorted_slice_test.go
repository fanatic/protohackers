package jobcentre

import "testing"

func TestSortedSlice(t *testing.T) {

	t.Run("remove highest priority item", func(t *testing.T) {
		s := []*Job{}
		s = InsertIntoJobPrioritySliceSorted(s, &Job{ID: 1, Priority: 1, Queue: "a"})
		s = InsertIntoJobPrioritySliceSorted(s, &Job{ID: 3, Priority: 3, Queue: "b"})
		s = InsertIntoJobPrioritySliceSorted(s, &Job{ID: 2, Priority: 2, Queue: "c"})
		s = InsertIntoJobPrioritySliceSorted(s, &Job{ID: 4, Priority: 4, Queue: "d"})

		if HighestPriorityJob(s, []string{"a", "b", "c"}).Priority != 3 {
			t.Error("Expected highest priority job to be 3, got", HighestPriorityJob(s, []string{"a", "b", "c"}).Priority)
		}

		s = RemoveFromJobPrioritySlice(s, Job{ID: 3, Priority: 3})
		if HighestPriorityJob(s, []string{"a", "b", "c"}).Priority != 2 {
			t.Error("Expected highest priority job to be 2, got", HighestPriorityJob(s, []string{"a", "b", "c"}).Priority)
		}

		s = RemoveFromJobPrioritySlice(s, Job{ID: 2, Priority: 2})
		if HighestPriorityJob(s, []string{"a", "b", "c"}).Priority != 1 {
			t.Error("Expected highest priority job to be 1, got", HighestPriorityJob(s, []string{"a", "b", "c"}).Priority)
		}
	})

	t.Run("remove medium priority item", func(t *testing.T) {
		s := []*Job{}
		s = InsertIntoJobPrioritySliceSorted(s, &Job{ID: 6, Priority: 6})
		s = InsertIntoJobPrioritySliceSorted(s, &Job{ID: 5, Priority: 5})
		s = InsertIntoJobPrioritySliceSorted(s, &Job{ID: 4, Priority: 4})
		s = RemoveFromJobPrioritySlice(s, Job{ID: 5, Priority: 5})
		for _, j := range s {
			if j.Priority == 5 {
				t.Error("Expected 5 to be removed from slice")
			}
		}
	})

	t.Run("remove medium priority item", func(t *testing.T) {
		s := []*Job{}
		s = InsertIntoJobPrioritySliceSorted(s, &Job{ID: 0, Priority: 0})
		s = InsertIntoJobPrioritySliceSorted(s, &Job{ID: 1, Priority: 1})
		s = InsertIntoJobPrioritySliceSorted(s, &Job{ID: 2, Priority: 1})
		s = InsertIntoJobPrioritySliceSorted(s, &Job{ID: 3, Priority: 1})
		s = InsertIntoJobPrioritySliceSorted(s, &Job{ID: 4, Priority: 2})
		s = RemoveFromJobPrioritySlice(s, Job{ID: 1, Priority: 1})
		for _, j := range s {
			if j.ID == 1 {
				t.Error("Expected 1 to be removed from slice")
			}
		}
	})

	t.Run("remove medium priority item", func(t *testing.T) {
		s := []*Job{}
		s = InsertIntoJobPrioritySliceSorted(s, &Job{ID: 0, Priority: 0})
		s = InsertIntoJobPrioritySliceSorted(s, &Job{ID: 1, Priority: 1})
		s = InsertIntoJobPrioritySliceSorted(s, &Job{ID: 2, Priority: 1})
		s = InsertIntoJobPrioritySliceSorted(s, &Job{ID: 3, Priority: 1})
		s = InsertIntoJobPrioritySliceSorted(s, &Job{ID: 4, Priority: 2})
		s = RemoveFromJobPrioritySlice(s, Job{ID: 2, Priority: 1})
		for _, j := range s {
			if j.ID == 2 {
				t.Error("Expected 2 to be removed from slice")
			}
		}
	})
	t.Run("remove medium priority item", func(t *testing.T) {
		s := []*Job{}
		s = InsertIntoJobPrioritySliceSorted(s, &Job{ID: 0, Priority: 0})
		s = InsertIntoJobPrioritySliceSorted(s, &Job{ID: 1, Priority: 1})
		s = InsertIntoJobPrioritySliceSorted(s, &Job{ID: 2, Priority: 1})
		s = InsertIntoJobPrioritySliceSorted(s, &Job{ID: 3, Priority: 1})
		s = InsertIntoJobPrioritySliceSorted(s, &Job{ID: 4, Priority: 2})
		s = RemoveFromJobPrioritySlice(s, Job{ID: 3, Priority: 1})
		for _, j := range s {
			if j.ID == 3 {
				t.Error("Expected 2 to be removed from slice")
			}
		}
	})

}

func BenchmarkInsert(b *testing.B) {
	s := []*Job{}
	for i := 0; i < b.N; i++ {
		s = InsertIntoJobPrioritySliceSorted(s, &Job{ID: i, Priority: i})
	}
}
