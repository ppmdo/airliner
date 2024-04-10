package kayak

import (
	"context"
	"fmt"
	"os"
	"path"
	"sync"
	"testing"

	"github.com/chromedp/cdproto/emulation"
	"github.com/chromedp/chromedp"
)

var (
	browserCtx  context.Context
	testdataDir string = ""
	allocOpts          = chromedp.DefaultExecAllocatorOptions[:]
)

func testAllocateSeparate(t testing.TB) (context.Context, context.CancelFunc) {
	// Entirely new browser, unlike testAllocate.
	alloCtx, cancel := chromedp.NewExecAllocator(context.Background(), allocOpts...)
	ctx, cancel := chromedp.NewContext(alloCtx)

	// set the viewport size, to know what screenshot size to expect
	width, height := 1024, 768
	if err := chromedp.Run(
		ctx,
		emulation.SetDeviceMetricsOverride(int64(width), int64(height), 1.0, false),
	); err != nil {
		t.Fatal(err)
	}

	cancel = func() {
		if err := chromedp.Cancel(ctx); err != nil {
			t.Error(err)
		}
	}

	return ctx, cancel
}

var allocateOnce sync.Once

func init() {
	wd, err := os.Getwd()
	if err != nil {
		panic(fmt.Sprintf("could not get working directory: %v", err))
	}
	testdataDir = "file://" + path.Join(wd, "testdata")

	// Disabling the GPU helps portability with some systems like Travis,
	// and can slightly speed up the tests on other systems.
	allocOpts = append(allocOpts, chromedp.DisableGPU)

	if noHeadless := os.Getenv("NO_HEADLESS"); noHeadless != "" && noHeadless != "false" {
		allocOpts = append(allocOpts, chromedp.Flag("headless", false))
	}
}

func testAllocate(t testing.TB, name string) (context.Context, context.CancelFunc) {

	// Start the browser exactly once, as needed.
	allocateOnce.Do(func() { browserCtx, _ = testAllocateSeparate(t) })

	if browserCtx == nil {
		// allocateOnce.Do failed; continuing would result in panics.
		t.FailNow()
	}

	// Same browser, new tab; not needing to start new chrome browsers for
	// each test gives a huge speed-up.
	ctx, _ := chromedp.NewContext(browserCtx)

	if err := chromedp.Run(ctx, chromedp.Navigate(testdataDir+"/"+name)); err != nil {
		t.Fatal(err)
	}

	cancel := func() {
		if err := chromedp.Cancel(ctx); err != nil {
			t.Error(err)
		}
	}
	return ctx, cancel
}

func TestFindBestOfferPrice(t *testing.T) {
	t.Parallel()

	ctx, _ := testAllocate(t, "example_result.html")

	bestOffer := findBestOfferPrice(&ctx)
	if *bestOffer != "273 €" {
		msg := fmt.Sprintf("Expected %s to equal '273'", *bestOffer)
		t.Log(msg)
		t.Fail()
	}
}

func TestCountResultNodes(t *testing.T) {
	t.Parallel()

	ctx, _ := testAllocate(t, "example_result.html")

	resultNodes := countResultList(&ctx)

	if resultNodes != 15 {
		msg := fmt.Sprintf("Expected %d to equal: 15", resultNodes)
		t.Log(msg)
		t.Fail()
	}
}
