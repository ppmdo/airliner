package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"sync"
	"time"

	calc "airliner/calculation"
	db "airliner/database"
	ky "airliner/kayak"
	md "airliner/model"
	tg "airliner/telegram"
)

func main() {
	var wg sync.WaitGroup

	var fromcity = flag.String("from", "", "3 letter upercase code for the city flying from.")
	var tocity = flag.String("to", "", "3 letter upercase code for the city flying to.")
	var lookahead = flag.Int("look-ahead", -1, "number of days to look ahead")
	var duration = flag.Int("duration", -1, "journey duration")

    var client = db.InitDB("test_influxdb.env")

	flag.Parse()

	if *fromcity == "" {
		fmt.Println("ERROR argument --from not supplied")
		return
	}
	if *tocity == "" {
		fmt.Println("ERROR argument --to not supplied")
		return
	}
	if *lookahead == -1 {
		fmt.Println("ERROR argument --look-ahead not supplied")
		return
	}
	if *duration == -1 {
		fmt.Println("ERROR argument --duration not supplied")
		return
	}

	outChan := make(chan *md.Offer)
	inChan := make(chan *md.Payload)
	offers := make([]*md.Offer, 0, 0)
	sem := make(chan int, 3)

	fromCity := *fromcity
	toCity := *tocity

	initialDate := ky.CalculateInitialDate(time.Now())
	tripDuration := *duration
	datesToLookAhead := *lookahead

	bot, err := tg.InitBot()
	if err != nil {
		log.Panic(err)
	}
	notifyStart(bot)

	wg.Add(2)
	go ky.CreatePayloads(
		fromCity, toCity, initialDate, tripDuration, datesToLookAhead, inChan, &wg,
	)

	go readOffers(
		outChan, &offers, &wg,
	)

	ky.AsyncGetOfferForPayloads(inChan, outChan, sem)
	wg.Wait()

	minOffer := calc.GetMinPriceOffer(offers)

	notifyEnd(bot, fromCity, toCity, &tripDuration, minOffer)

    saveOffersToDB(&offers, client)
	cleanupFiles(offers)
}

func notifyStart(bot *tg.Bot) {
	tg.SendMessage(bot, "Hi there... Query operation starting...")
}

func notifyEnd(bot *tg.Bot, fromCity string, toCity string, tripDuration *int, offer *md.Offer) {
	msgText := fmt.Sprintf(
		"The best offer to travel for %d days from %s to %s is: Price %.2f, Departure: %s, Return: %s",
		*tripDuration,
		fromCity,
		toCity,
		offer.Price,
		offer.DepartureDate.Format("2006-01-02"),
		offer.ReturnDate.Format("2006-01-02"),
	)
	tg.SendMessage(bot, msgText)

	reader, err := os.Open(offer.Screenshot)
	if err != nil {
		log.Panic(err)
	}
	tg.SendImage(bot, offer.Screenshot, reader)
}

func cleanupFiles(offers []*md.Offer) {
	for _, v := range offers {
		err := os.Remove(v.Screenshot)

		if err != nil {
			log.Panic(err)
		}
	}
}

func readOffers(ch chan *md.Offer, offers *[]*md.Offer, wg *sync.WaitGroup) {
	defer wg.Done()

	for v := range ch {
		fmt.Println(v.String())
		*offers = append(*offers, v)
	}
}

func saveOffersToDB(offers *[]*md.Offer, client db.DBClient) {

	for _, offer := range *offers {
		db.Write_event_with_fluent_Style(
			client,
			db.AirlineOffer{
				offer.Url,
				offer.FromAirport,
				offer.ToAirport,
				offer.DepartureDate,
				offer.ReturnDate,
				offer.Price,
				offer.CreatedOn,
			},
			db.Bucket,
		)
	}

}
