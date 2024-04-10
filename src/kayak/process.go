package kayak

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"path"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/chromedp/cdproto/emulation"
	"github.com/chromedp/chromedp"
	"github.com/google/uuid"

	md "airliner/model"
)

const TIMEOUT_MINUTES = time.Minute * 10
const MAX_RETRIES = 5

func takeScreenshot(ctx *context.Context, buff *[]byte) {
	if err := chromedp.Run(*ctx,
		chromedp.CaptureScreenshot(buff),
	); err != nil {
		log.Fatal(err)
	}
}

func getHtml(ctx *context.Context, html *string) {
	if err := chromedp.Run(*ctx, chromedp.OuterHTML("/", html)); err != nil {
		log.Fatal(err)
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

func createFilename(timestamp string, base string, ext string) string {
	return fmt.Sprintf("%s-%s.%s", timestamp, base, ext)
}

func writeDebugData(prefix string, screenshot []byte, html string) {
	tstamp := time.Now().Format("2006_01_02__15_04_05")
	htmlFname := createFilename(tstamp, prefix, "html")
	screenshotFname := createFilename(tstamp, prefix, "png")

	if err := os.WriteFile(screenshotFname, screenshot, 0o644); err != nil {
		log.Fatal(err)
	}

	f, err := os.Create(htmlFname)
	if err != nil {
		log.Fatal(err)
	}
	defer f.Close()

	_, err = f.WriteString(html)
	if err != nil {
		log.Fatal(err)
	}
}

func isReady(ctx *context.Context) (bool, error) {
	retries := MAX_RETRIES
	sleepMultiplier := 2
	var err error
	var adviceText *string

	fmt.Println("Checking if ready...")

	for retries > 0 {
		adviceText, err = getAdviceText(ctx)

		if err == nil && !strings.Contains(*adviceText, "load") {
			break
		} else {
			log.Printf("Couldn't find advice text. Retrying in %d seconds... (Retries left: %d)\n", sleepMultiplier, retries)
			time.Sleep(time.Duration(sleepMultiplier) * time.Second)

			retries--
			sleepMultiplier *= 2
		}
	}

	if adviceText == nil {
		log.Printf("CRITICAL :: Failed to find advice text. Cannot continue.")
		return false, errors.New("advice text not found")
	}

	retries = MAX_RETRIES
	sleepMultiplier = 2
	for retries > 0 {
		nodeCount := countResultList(ctx)
		if nodeCount > 0 {
			return true, nil
		} else {
			fmt.Printf("Didn't find results section. Retrying in %d seconds... (Retries left: %d)\n", sleepMultiplier, retries)
			time.Sleep(time.Duration(sleepMultiplier) * time.Second)

			retries--
			sleepMultiplier *= 2
		}
	}

	return false, errors.New("result section not found")
}

func CalculateInitialDate(referenceDate time.Time) time.Time {
	return referenceDate.Add(time.Duration(28) * md.Day)
}

func AsyncGetOfferForPayloads(inChan chan *md.Payload, outChan chan *md.Offer, sem chan int) {
	defer close(outChan)
	var wg sync.WaitGroup

	inner := func(v *md.Payload) {
		defer wg.Done()

		sem <- 1
		off, _ := GetOfferForPayload(v)
		if off != nil {
			outChan <- off
		}
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

func remove(s []func(*chromedp.ExecAllocator), i int) []func(*chromedp.ExecAllocator) {
	s[i] = s[len(s)-1]
	return s[:len(s)-1]
}

func GetOfferForPayload(payload *md.Payload) (*md.Offer, error) {
	fmt.Println(fmt.Sprintf("Getting %s", payload.DateString()))
	var headless = true

	userDataDir := path.Join(os.TempDir(), "airliner-chrome"+uuid.NewString())

	opts := append(
		chromedp.DefaultExecAllocatorOptions[:],
		chromedp.UserDataDir(userDataDir),
		chromedp.UserAgent("Mozilla/5.0 (X11; Ubuntu; Linux x86_64; rv:109.0) Gecko/20100101 Firefox/109.0"),
	)

	if !headless {
		// Remove Headless
		opts = remove(opts, 2)
	}

	alloCtx, cancel := chromedp.NewExecAllocator(context.Background(), opts...)
	defer cancel()

	ctx, cancel := chromedp.NewContext(alloCtx)
	defer cancel()

	// create a timeout
	ctx, cancel = context.WithTimeout(ctx, TIMEOUT_MINUTES)
	defer cancel()

	url := "https://www.kayak.com/flights/" + payload.FromCity + "-" + payload.ToCity + "/" + payload.DateString() + "?sort=price_a"

	if payload.Direct {
		url += "&fs=stops=~0"
	}

	log.Printf("Fetching: %s\n", url)

	// set the viewport size, to know what screenshot size to expect
	width, height := 1024, 768

	if err := chromedp.Run(ctx,
		chromedp.Navigate(url),
		emulation.SetDeviceMetricsOverride(int64(width), int64(height), 1.0, false),
	); err != nil {
		log.Fatal(err)
	}

	rdy, err := isReady(&ctx)
	screenshot := takeAndSaveScreenshot(&ctx, fmt.Sprintf("%d", payload.Id))

	if !rdy || err != nil {
		return &md.Offer{
			Url:             url,
			FromAirport:     payload.FromCity,
			ToAirport:       payload.ToCity,
			DepartureDate:   payload.DepartureDate,
			ReturnDate:      payload.ReturnDate,
			Price:           -1,
			Screenshot:      screenshot,
			CreatedOn:       time.Now(),
			FetchSuccessful: false,
		}, nil
	}

	bestPrice := findBestOfferPrice(&ctx)

	v, err := strconv.ParseFloat(strings.Trim(strings.Replace(*bestPrice, "$", "", -1), " "), 8)
	if err != nil {
		fmt.Println("Fatal: Failed to parse float value for price")
	}

	return &md.Offer{
		Url:             url,
		FromAirport:     payload.FromCity,
		ToAirport:       payload.ToCity,
		DepartureDate:   payload.DepartureDate,
		ReturnDate:      payload.ReturnDate,
		Price:           v,
		Screenshot:      screenshot,
		CreatedOn:       time.Now(),
		FetchSuccessful: true,
	}, nil
}
