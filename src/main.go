package main

import (
	"fmt"
	"log"
	"os"
	"sync"
	"time"

    calc "airliner/calculation"
	tg "airliner/telegram"
    md "airliner/model"
    ky "airliner/kayak"
)


func main() {
	var wg sync.WaitGroup

	outChan := make(chan *md.Offer)
	inChan := make(chan *md.Payload)
	offers := make([]*md.Offer, 0, 0)
	sem := make(chan int, 3)

    fromCity := "MUC"
    toCity := "LIS"

    tm, _ := time.Parse("2006-01-02", "2023-03-28")
	initialDate := ky.CalculateInitialDate(tm)
	tripDuration := 10
	datesToLookAhead := 10

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
