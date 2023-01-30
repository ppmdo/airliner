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

	"github.com/chromedp/chromedp"
)

type Offer struct {
	price         float64
	departureDate time.Time
	returnDate    time.Time
}

type OfferResult struct {
	priceText      string
	pageScreenshot []byte
}

func (o *Offer) String() string {
	return fmt.Sprintf("Price: %.2f - From %s to %s", o.price, o.departureDate.Format("2006-01-02"), o.returnDate.Format("2006-01-02"))
}

type Payload struct {
	departureDate time.Time
	returnDate    time.Time
}

func (p *Payload) String() string {
	a := p.departureDate.Format("2006-01-02")
	b := p.returnDate.Format("2006-01-02")

	return fmt.Sprintf("%s/%s", a, b)
}

const day = 24 * time.Hour

func createPayloads(initialDate time.Time, tripLength int, daysToLookup int, ch chan *Payload, wg *sync.WaitGroup) {
    defer close(ch)
    defer wg.Done()

	i := 0
	for i < daysToLookup {
        initialDate2 := initialDate.Add(time.Duration(i) * day)
		ch <- &Payload{
            initialDate2,
			initialDate2.Add(time.Duration(tripLength) * day),
		}
        i++
	}
}

func takeAndSaveScreenshot(ctx *context.Context) {
	var buff []byte

	if err := chromedp.Run(*ctx,
		//chromedp.WaitVisible(adviceSelector, chromedp.ByQuery),
		chromedp.CaptureScreenshot(&buff),
	); err != nil {
		log.Fatal(err)
	}
	if err := os.WriteFile("fullScreenshot.png", buff, 0o644); err != nil {
		log.Fatal(err)
	}

	log.Printf("wrote elementScreenshot.png and fullScreenshot.png")
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

	if err := chromedp.Run(*ctx,
		chromedp.Nodes(selector, &nodes),
	); err != nil {
		log.Fatal(err)
	}

	if err := chromedp.Run(*ctx,
		chromedp.Text("[class$=price-text]", &result, chromedp.FromNode(nodes[0])),
	); err != nil {
		log.Fatal(err)
	}

    fmt.Println("Best offer extracted")
	return &result
}

func getInitialDate() time.Time {
	timeString := "2023-03-01"
	d, err := time.Parse("2006-01-02", timeString)
	if err != nil {
		fmt.Println("Could not parse time:", err)
	}
	return d
}

func asyncGetOfferForPayloads(inChan chan *Payload, outChan chan *Offer, sem chan int) {
    defer close(outChan)
    var wg sync.WaitGroup

    inner := func(v *Payload) {
		defer wg.Done()

        sem <- 1
		outChan <- getOfferForPayload(v)
        <- sem
	}

	for v := range inChan {
		if v != nil {
			wg.Add(1)
			go inner(v)
		}
	}

    wg.Wait()
}
func getOfferForPayload(payload *Payload) *Offer {
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
	ctx, cancel = context.WithTimeout(ctx, 60*time.Second)
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

    takeAndSaveScreenshot(&ctx)

	for {
		if isReady(&ctx) {
			break
		}
	}
    takeAndSaveScreenshot(&ctx)

	bestPrice := findBestOfferPrice(&ctx)

	v, err := strconv.ParseFloat(strings.Trim(strings.Replace(*bestPrice, "$", "", -1), " "), 8)
	if err != nil {
		fmt.Println("Fatal: Failed to parse float value for price")
	}

	return &Offer{
		v,
		payload.departureDate,
		payload.returnDate,
	}
}

func main() {
	outChan := make(chan *Offer)
    inChan := make(chan *Payload)
    sem := make(chan int, 1)

    var wg sync.WaitGroup

    initialDate := getInitialDate()

    wg.Add(2)
    go createPayloads(
		initialDate, 10, 2, inChan, &wg,
	)

    go func() {
        defer wg.Done()
        for v := range outChan {
		fmt.Println(v.String())
	}}()

	asyncGetOfferForPayloads(inChan, outChan, sem)
    wg.Wait()
}
