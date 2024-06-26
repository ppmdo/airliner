package kayak

import (
	"sync"
	"time"

	md "airliner/model"
)

func CreatePayloads(
	fromCity string,
	toCity string,
	initialDate time.Time,
	tripLength int,
	daysToLookup int,
	direct bool,
	ch chan *md.Payload,
	wg *sync.WaitGroup,
) {
	defer close(ch)
	defer wg.Done()

	i := 0
	for i < daysToLookup {
		initialDate2 := initialDate.Add(time.Duration(i) * md.Day)
		var returnDate time.Time

		if tripLength > 0 {
			returnDate = initialDate2.Add(time.Duration(tripLength) * md.Day)
		}

		ch <- &md.Payload{
			FromCity:      fromCity,
			ToCity:        toCity,
			DepartureDate: initialDate2,
			ReturnDate:    returnDate,
			Id:            i,
			Direct:        direct,
		}
		i++
	}
}
