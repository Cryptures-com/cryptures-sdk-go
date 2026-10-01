package cryptures

import "context"

// Iter walks every item of a cursor-paginated list, fetching pages lazily as
// it goes. Obtain one from a ListAutoPaging method and drive it like
// bufio.Scanner:
//
//	it := client.Compliance.Sessions.ListAutoPaging(ctx, &cryptures.ListSessionsParams{Status: "Approved"})
//	for it.Next() {
//		session := it.Current()
//		// ...
//	}
//	if err := it.Err(); err != nil {
//		// handle error
//	}
//
// An Iter is not safe for concurrent use.
type Iter[T any] struct {
	ctx   context.Context
	fetch func(ctx context.Context, cursor string) (items []T, nextCursor string, err error)

	page    []T
	index   int
	cursor  string
	started bool
	done    bool
	current T
	err     error
}

func newIter[T any](ctx context.Context, fetch func(ctx context.Context, cursor string) ([]T, string, error)) *Iter[T] {
	return &Iter[T]{ctx: ctx, fetch: fetch}
}

// Next advances to the next item, fetching the next page when the current
// one is exhausted. It returns false when there are no more items or an
// error occurred; check Err afterwards.
func (it *Iter[T]) Next() bool {
	if it.err != nil {
		return false
	}
	for it.index >= len(it.page) {
		if it.done || (it.started && it.cursor == "") {
			it.done = true
			return false
		}
		items, next, err := it.fetch(it.ctx, it.cursor)
		it.started = true
		if err != nil {
			it.err = err
			return false
		}
		it.page, it.index, it.cursor = items, 0, next
		if len(items) == 0 && next == "" {
			it.done = true
			return false
		}
	}
	it.current = it.page[it.index]
	it.index++
	return true
}

// Current returns the item Next most recently advanced to.
func (it *Iter[T]) Current() T { return it.current }

// Err returns the first error encountered while fetching pages, if any.
func (it *Iter[T]) Err() error { return it.err }
