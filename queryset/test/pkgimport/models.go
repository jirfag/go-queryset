package models

//go:generate goqueryset -in models.go

import (
	forex "github.com/jirfag/go-queryset/queryset/test/pkgimport/forex/v1"
	forexAlias "github.com/jirfag/go-queryset/queryset/test/pkgimport/forex/v1"
)

// Example is a test struct
// gen:qs
type Example struct {
	PriceID   int64
	Currency1 forexAlias.Currency1
	Currency2 forex.Currency2
	Currency3 forex.Currency3
}
