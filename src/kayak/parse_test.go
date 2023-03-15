package kayak

import (
	"context"
	"fmt"
	"log"
	"os"
	"testing"

	"github.com/chromedp/cdproto/emulation"
	"github.com/chromedp/chromedp"
)

func TestFindBestOfferPrice(t *testing.T) {
	wd, err := os.Getwd()
	if err != nil {
		panic(fmt.Sprintf("could not get working directory: %v", err))
	}

	opts := chromedp.DefaultExecAllocatorOptions[:]

	alloCtx, cancel := chromedp.NewExecAllocator(context.Background(), opts...)
	defer cancel()

	ctx, cancel := chromedp.NewContext(alloCtx)
	defer cancel()

	// set the viewport size, to know what screenshot size to expect
	width, height := 1024, 768

	url := "file://" + wd + "/testdata/example_result.html"

	if err := chromedp.Run(ctx,
		chromedp.Navigate(url),
		emulation.SetDeviceMetricsOverride(int64(width), int64(height), 1.0, false),
	); err != nil {
		log.Fatal(err)
	}

	bestOffer := findBestOfferPrice(&ctx)
	if *bestOffer != "$80" {
		msg := fmt.Sprintf("Expected %s to equal '$80'", *bestOffer)
		t.Log(msg)
		t.Fail()
	}
}
