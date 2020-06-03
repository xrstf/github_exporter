package fetcher

type integerQueue map[int]struct{}

func (q integerQueue) fillSliceUpTo(list []int, max int) []int {
	for number := range q {
		list = append(list, number)
		if len(list) >= max {
			break
		}
	}

	return list
}

func (q integerQueue) numbers() []int {
	result := []int{}

	for number := range q {
		result = append(result, number)
	}

	return result
}

type prioritizedIntegerQueue struct {
	priority integerQueue
	regular  integerQueue
}

func newPrioritizedIntegerQueue() prioritizedIntegerQueue {
	return prioritizedIntegerQueue{
		priority: integerQueue{},
		regular:  integerQueue{},
	}
}

func (q *prioritizedIntegerQueue) priorityEnqueue(numbers []int) {
	q.enqueue(numbers, q.priority)
}

func (q *prioritizedIntegerQueue) regularEnqueue(numbers []int) {
	q.enqueue(numbers, q.regular)
}

func (q *prioritizedIntegerQueue) enqueue(numbers []int, queue integerQueue) {
	for _, number := range numbers {
		queue[number] = struct{}{}
	}
}

func (q *prioritizedIntegerQueue) prioritySize() int {
	return len(q.priority)
}

func (q *prioritizedIntegerQueue) regularSize() int {
	return len(q.regular)
}

func (q *prioritizedIntegerQueue) getBatch(minBatchSize int, maxBatchSize int) []int {
	// get the first N random priority items
	items := q.priority.fillSliceUpTo([]int{}, maxBatchSize)

	// if the batch is already full, stop and return
	if len(items) >= maxBatchSize {
		return items
	}

	// otherwise, continue to add random regular items
	items = q.regular.fillSliceUpTo(items, maxBatchSize)

	// if we have reached the min batch size, it's good enough
	// and we can return
	if len(items) >= minBatchSize {
		return items
	}

	// not enough items at all
	return nil
}

func (q *prioritizedIntegerQueue) dequeue(numbers []int) {
	for _, number := range numbers {
		delete(q.priority, number)
		delete(q.regular, number)
	}
}
