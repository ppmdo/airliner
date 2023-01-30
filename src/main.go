package main

import (
	"context"
	"fmt"
	"github.com/chromedp/cdproto/cdp"
	"github.com/chromedp/cdproto/emulation"
	"log"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

    calc "airliner/calculation"
	tg "airliner/telegram"
    md "airliner/model"
	"github.com/chromedp/chromedp"
)


const day = 24 * time.Hour

func createPayloads(initialDate time.Time, tripLength int, daysToLookup int, ch chan *md.Payload, wg *sync.WaitGroup) {
	defer close(ch)
	defer wg.Done()

	i := 0
	for i < daysToLookup {
		initialDate2 := initialDate.Add(time.Duration(i) * day)
		ch <- &md.Payload{
			initialDate2,
			initialDate2.Add(time.Duration(tripLength) * day),
			i,
		}
		i++
	}
}

func takeAndSaveScreenshot(ctx *context.Context, fname string) string {
	var buff []byte

	fname = fmt.Sprintf("%s.png", fname)

	if err := chromedp.Run(*ctx,
		chromedp.CaptureScreenshot(&buff),
	); err != nil {
		log.Fatal(err)
	}
	if err := os.WriteFile(fname, buff, 0o644); err != nil {
		log.Fatal(err)
	}

	return fname
}

func getAdviceText(ctx *context.Context) *string {
	adviceSelector := "[class$=\"-advice\"]"
	var adviceText string

	if err := chromedp.Run(*ctx,
		//chromedp.WaitVisible(adviceSelector, chromedp.ByQuery),
		chromedp.Text(adviceSelector, &adviceText, chromedp.ByQuery),
	); err != nil {
		log.Fatal(err)
	}

	adviceText = strings.ToLower(adviceText)
	return &adviceText
}
func isReady(ctx *context.Context) bool {
	adviceText := *getAdviceText(ctx)
	for {
		if strings.Contains(adviceText, "buy") {
			return true
		} else {
			time.Sleep(2 * time.Second)
			adviceText = *getAdviceText(ctx)
		}
	}
}

func findBestOfferPrice(ctx *context.Context) *string {
	fmt.Println("Extracting best offer...")
	var nodes = make([]*cdp.Node, 1)
	var result string

	selector := "//div[@data-resultid and contains(., 'Best')]"

    repeat := 3
    for {
        if err := chromedp.Run(*ctx,
            chromedp.Nodes(selector, &nodes),
            ); err != nil {
            log.Fatal(err)
        }
        if len(nodes) > 0 {
            break

        } else {
            fmt.Println("Couldn't find best offer, retrying...")
            time.Sleep(time.Second)
            repeat --
        }
    }

	if err := chromedp.Run(*ctx,
		chromedp.Text("[class$=price-text]", &result, chromedp.FromNode(nodes[0])),
	); err != nil {
		log.Fatal(err)
	}

	return &result
}

func calculateInitialDate(referenceDate time.Time) time.Time {
	return referenceDate.Add(time.Duration(28) * day)
}

func asyncGetOfferForPayloads(inChan chan *md.Payload, outChan chan *md.Offer, sem chan int) {
	defer close(outChan)
	var wg sync.WaitGroup

	inner := func(v *md.Payload) {
		defer wg.Done()

		sem <- 1
		outChan <- getOfferForPayload(v)
		<-sem
	}

	for v := range inChan {
		if v != nil {
			wg.Add(1)
			go inner(v)
		}
	}

	wg.Wait()
}
func getOfferForPayload(payload *md.Payload) *md.Offer {
	fmt.Println(fmt.Sprintf("Getting %s", payload.String()))

	opts := append(
		chromedp.DefaultExecAllocatorOptions[:],
		chromedp.UserAgent("Mozilla/5.0 (Macintosh; Intel Mac OS X 11_2_3) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/89.0.4389.114 Safari/537.36"),
	)

	alloCtx, cancel := chromedp.NewExecAllocator(context.Background(), opts...)
	defer cancel()

	ctx, cancel := chromedp.NewContext(alloCtx)
	defer cancel()

	// create a timeout
	ctx, cancel = context.WithTimeout(ctx, 180*time.Second)
	defer cancel()

	url := "https://www.kayak.com/flights/MUC-LIS/" + payload.String() + "?sort=bestflight_a&fs=stops=~0"

	// set the viewport size, to know what screenshot size to expect
	width, height := 1024, 768

	if err := chromedp.Run(ctx,
		chromedp.Navigate(url),
		emulation.SetDeviceMetricsOverride(int64(width), int64(height), 1.0, false),
	); err != nil {
		log.Fatal(err)
	}

	for {
		if isReady(&ctx) {
			break
		}
	}
	screenshot := takeAndSaveScreenshot(&ctx, fmt.Sprintf("%d", payload.Id))

	bestPrice := findBestOfferPrice(&ctx)

	v, err := strconv.ParseFloat(strings.Trim(strings.Replace(*bestPrice, "$", "", -1), " "), 8)
	if err != nil {
		fmt.Println("Fatal: Failed to parse float value for price")
	}

	return &md.Offer{
		v,
		payload.DepartureDate,
		payload.ReturnDate,
		screenshot,
	}
}


func main() {
	var wg sync.WaitGroup

	outChan := make(chan *md.Offer)
	inChan := make(chan *md.Payload)
	offers := make([]*md.Offer, 0, 0)
	sem := make(chan int, 3)

	initialDate := calculateInitialDate(time.Now())
	tripDuration := 10
	datesToLookAhead := 10

	bot, err := tg.InitBot()
	if err != nil {
		log.Panic(err)
	}
	notifyStart(bot)

	wg.Add(2)
	go createPayloads(
		initialDate, tripDuration, datesToLookAhead, inChan, &wg,
	)

    go readOffers(
        outChan, &offers, &wg,
    )

	asyncGetOfferForPayloads(inChan, outChan, sem)
	wg.Wait()

	minOffer := calc.GetMinPriceOffer(offers)

	notifyEnd(bot, &tripDuration, minOffer)
	cleanupFiles(offers)
}

func notifyStart(bot *tg.Bot) {
	tg.SendMessage(bot, "Hi there... Query operation starting...")
}

func notifyEnd(bot *tg.Bot, tripDuration *int, offer *md.Offer) {
	msgText := fmt.Sprintf(
        "The best offer to travel for %d days to Lisbon is: Price %.2f, Departure: %s, Return: %s",
        *tripDuration,
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
