package kayak

import (
	"context"
	"errors"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/chromedp/cdproto/cdp"
	"github.com/chromedp/chromedp"
)

func getAdviceText(ctx *context.Context) (*string, error) {
	adviceSelector := "[class$=\"-advice\"]"
	var adviceText = ""

	if err := chromedp.Run(*ctx,
		chromedp.Text(adviceSelector, &adviceText, chromedp.ByQuery, chromedp.AtLeast(0)),
	); err != nil || adviceText == "" {
		return nil, errors.New("couldn't find advice text")
	}

	adviceText = strings.ToLower(adviceText)
	return &adviceText, nil
}

func countResultList(ctx *context.Context) int {
	selector := "//div[@data-resultid]"
	nodes := make([]*cdp.Node, 25)

	if err := chromedp.Run(*ctx,
		chromedp.Nodes(selector, &nodes, chromedp.AtLeast(0)),
	); err != nil {
		fmt.Println(err)
		return 0
	}

	fmt.Printf("Found %d result nodes.\n", len(nodes))
	return len(nodes)
}

func findBestOfferPrice(ctx *context.Context) *string {
	fmt.Println("Extracting best offer...")
	var nodes = make([]*cdp.Node, 10)
	var result string

	selector := "//div[@data-resultid]"

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
			repeat--
		}
	}

	if err := chromedp.Run(*ctx,
		chromedp.Text("[class$=price-text]", &result, chromedp.FromNode(nodes[0])),
	); err != nil {
		log.Fatal(err)
	}

	println("Found best offer...")
	return &result
}
