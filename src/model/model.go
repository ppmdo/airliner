package model

import (
    "fmt"
    "time"
)

const Day = 24 * time.Hour

type Offer struct {
    Url string
    FromAirport string
    ToAirport string
	DepartureDate time.Time
	ReturnDate    time.Time
	Price         float64
	Screenshot    string
    CreatedOn     time.Time
}

func (o *Offer) String() string {
	return fmt.Sprintf("Price: %.2f - From %s to %s", o.Price, o.DepartureDate.Format("2006-01-02"), o.ReturnDate.Format("2006-01-02"))
}

type Payload struct {
    FromCity string
    ToCity string
	DepartureDate time.Time
	ReturnDate    time.Time
	Id            int
}

func (p *Payload) DateString() string {
	a := p.DepartureDate.Format("2006-01-02")
	b := p.ReturnDate.Format("2006-01-02")

	return fmt.Sprintf("%s/%s", a, b)
}
