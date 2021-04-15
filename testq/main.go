package main

//
// Interview question - fix the bug
//
// Prepared as a technical test when interviewing backend
// engineers.
//
// Expected output
// 2009/11/10 23:00:00 Processing action ABB [before AB]
// 2009/11/10 23:00:00 Processing action AB [before A]
// 2009/11/10 23:00:00 Processing action ACB [before AC]
// 2009/11/10 23:00:00 Processing action AC [before A]
// 2009/11/10 23:00:00 Processing action A
// 2009/11/10 23:00:00 Processing action BB [before B]
// 2009/11/10 23:00:00 Processing action B [next after A]
//

import (
	"context"
	"log"
	"time"
)

type Action interface {
	Process(ctx context.Context) error
	Before() []Action
	Next() Action
	Name() string
}

type A struct {
	procFn func(context.Context)
	before []Action
	next   Action
	name   string
}

func (a *A) Process(ctx context.Context) error {
	a.procFn(ctx)
	return nil
}

func (a *A) Before() []Action {
	return a.before
}

func (a *A) Next() Action {
	return a.next
}

func (a *A) Name() string {
	return a.name
}

func process(ctx context.Context, a Action) error {
	var fn func(context.Context, Action) error

	run := func(ctx context.Context, r Action) error {
		fn = func(ctx context.Context, f Action) error {
			before := f.Before()
			for _, b := range before {
				if err := fn(ctx, b); err != nil {
					return err
				}
			}

			go func() {
				if err := f.Process(ctx); err != nil {
					log.Fatal(err)
				}
			}()

			if f.Next() == nil {
				return nil
			}

			if err := fn(ctx, f.Next()); err != nil {
				return err
			}

			return nil
		}

		return fn(ctx, r)
	}

	return run(ctx, a)
}

func main() {
	ctx, cancel := context.WithTimeout(context.Background(), 1000*time.Millisecond)
	defer cancel()

	a := &A{
		name: "A",
		procFn: func(ctx context.Context) {
			log.Printf("Processing action A")
		},
		before: []Action{
			&A{
				name: "AB",
				procFn: func(ctx context.Context) {
					log.Printf("Processing action AB [before A]")
				},
				before: []Action{
					&A{
						name: "ABB",
						procFn: func(ctx context.Context) {
							log.Printf("Processing action ABB [before AB]")
						},
					},
				},
			},
			&A{
				name: "AC",
				procFn: func(ctx context.Context) {
					log.Printf("Processing action AC [before A]")
				},
				before: []Action{
					&A{
						name: "ACB",
						procFn: func(ctx context.Context) {
							log.Printf("Processing action ACB [before AC]")
						},
					},
				},
			},
		},
		next: &A{
			name: "B",
			procFn: func(ctx context.Context) {
				log.Printf("Processing action B [next after A]")
			},
			before: []Action{
				&A{
					name: "BB",
					procFn: func(ctx context.Context) {
						log.Printf("Processing action BB [before B]")
					},
				},
			},
		},
	}
	if err := process(ctx, a); err != nil {
		log.Fatal(err)
	}
}
