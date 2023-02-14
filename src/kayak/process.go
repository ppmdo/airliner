package kayak

import (
    "context"
    "errors"
    "fmt"
	"github.com/chromedp/cdproto/cdp"
	"github.com/chromedp/cdproto/emulation"
	"github.com/chromedp/chromedp"
    "log"
    "os"
    "strconv"
    "strings"
    "sync"
    "time"

    md "airliner/model"
)

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

func getAdviceText(ctx *context.Context) (*string, error) {
	adviceSelector := "[class$=\"-advice\"]"
	var adviceText string

	if err := chromedp.Run(*ctx,
		chromedp.Text(adviceSelector, &adviceText, chromedp.ByQuery, chromedp.AtLeast(0)),
	); err != nil {
        return nil, errors.New("couldn't find advice text")
	}

	adviceText = strings.ToLower(adviceText)
	return &adviceText, nil
}

func countResultList(ctx *context.Context) int {
    selector := "//div[@data-resultid and contains(., 'Best')]"
    nodes := make([]*cdp.Node, 10)

    if err := chromedp.Run(*ctx,
        chromedp.Nodes(selector, &nodes, chromedp.AtLeast(0)),
        ); err != nil {
            fmt.Println(err)
            return 0
        }

    fmt.Printf("Found %d result nodes.\n", len(nodes))
    return len(nodes)
}

func isReady(ctx *context.Context) (bool, error) {
    retries := 5

    fmt.Println("Checking if ready...")

	adviceText, err := getAdviceText(ctx)
	for {
		if !strings.Contains(*adviceText, "load") {
			break
		} else {
			time.Sleep(2 * time.Second)
			adviceText, err = getAdviceText(ctx)
		}
	}

    if err != nil {
        return false, err
    }

    for retries > 0 {
        nodeCount := countResultList(ctx)
        if nodeCount > 0 {
            return true, nil
        } else {
            fmt.Printf("Didn't find results section, retrying... (Retries left: %d)\n", retries)
            retries--
            time.Sleep(time.Second)
        }
    }

    return false, errors.New("result section not found")
}

func findBestOfferPrice(ctx *context.Context) *string {
	fmt.Println("Extracting best offer...")
	var nodes = make([]*cdp.Node, 10)
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

    println("Found best offer...")
	return &result
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


func GetOfferForPayload(payload *md.Payload) (*md.Offer, error) {
	fmt.Println(fmt.Sprintf("Getting %s", payload.DateString()))

	opts := append(
		chromedp.DefaultExecAllocatorOptions[:],
		chromedp.UserAgent("Mozilla/5.0 (Macintosh; Intel Mac OS X 11_2_3) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/89.0.4389.114 Safari/537.36"),
	)

	alloCtx, cancel := chromedp.NewExecAllocator(context.Background(), opts...)
	defer cancel()

	ctx, cancel := chromedp.NewContext(alloCtx)
	defer cancel()

	// create a timeout
	ctx, cancel = context.WithTimeout(ctx, time.Minute)
	defer cancel()

	url := "https://www.kayak.com/flights/"+ payload.FromCity + "-" + payload.ToCity + "/" + payload.DateString() + "?sort=bestflight_a&fs=stops=~0"

	// set the viewport size, to know what screenshot size to expect
	width, height := 1024, 768

	if err := chromedp.Run(ctx,
		chromedp.Navigate(url),
		emulation.SetDeviceMetricsOverride(int64(width), int64(height), 1.0, false),
	); err != nil {
		log.Fatal(err)
	}

	for {
		rdy, err := isReady(&ctx)
        if rdy {
			break
		} else {
            return nil, err
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
	}, nil
}