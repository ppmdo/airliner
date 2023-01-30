package main

import (
	"context"
	"fmt"
    "github.com/chromedp/cdproto/cdp"
    "io"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/chromedp/chromedp"
)

func writeHTML(content string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		io.WriteString(w, strings.TrimSpace(content))
	})
}

func getAdviceText(ctx *context.Context) *string {
	fmt.Println("Finding advice...")
	adviceSelector := "[class$=\"-advice\"]"
	var adviceText string

	if err := chromedp.Run(*ctx,
		chromedp.WaitVisible(adviceSelector, chromedp.ByQuery),
		chromedp.Text(adviceSelector, &adviceText, chromedp.ByQuery),
	); err != nil {
		log.Fatal(err)
	}
	adviceText = strings.ToLower(adviceText)
	return &adviceText
}
func isReady(ctx *context.Context, url *string) bool {
	if err := chromedp.Run(*ctx,
		chromedp.Navigate(*url),
	); err != nil {
		log.Fatal(err)
	}
	adviceText := *getAdviceText(ctx)
	for {
		if strings.Contains(adviceText, "buy") {
			fmt.Println("Advice found... Price is ready.")
			return true
		} else {
			fmt.Println("Waiting for Readyness")
			fmt.Println("Current advice:", adviceText)
			time.Sleep(2 * time.Second)
			adviceText = *getAdviceText(ctx)
		}
	}
}

func findBestOffer(ctx *context.Context) *string {
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

	return &result
}

func main() {
	opts := append(chromedp.DefaultExecAllocatorOptions[:], chromedp.Flag("headless", false))

	alloCtx, cancel := chromedp.NewExecAllocator(context.Background(), opts...)
	defer cancel()

	ctx, cancel := chromedp.NewContext(alloCtx)
	defer cancel()

	// create a timeout
	ctx, cancel = context.WithTimeout(ctx, 60*time.Second)
	defer cancel()

	url := "https://www.kayak.com/flights/MUC-LIS/2023-04-15/2023-04-22?sort=bestflight_a&fs=stops=~0"

	for {
		if isReady(&ctx, &url) {
			break
		}
	}

	bestOffer := findBestOffer(&ctx)

	fmt.Println(*bestOffer)
}
