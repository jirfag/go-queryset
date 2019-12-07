package models

//go:generate goqueryset -in models.go

import (
	forex "github.com/jirfag/go-queryset/internal/queryset/generator/test/pkgimport/forex/v1"
	forexAlias "github.com/jirfag/go-queryset/internal/queryset/generator/test/pkgimport/forex/v1"
	uuid "github.com/satori/go.uuid"
)

// Example is a test struct
// gen:qs
type Example struct {
	ID        uuid.UUID
	PriceID   int64
	Currency1 forexAlias.Currency1
	Currency2 forex.Currency2
	Currency3 forex.Currency3
}
