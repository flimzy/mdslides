// +build js

package main

import (
	"fmt"
	"net/http"
	"net/url"
	"strconv"

	"golang.org/x/net/html"

	"github.com/gopherjs/gopherjs/js"
	"github.com/gopherjs/jquery"
	//     "github.com/flimzy/jqeventrouter"
)

// Some spiffy shortcuts
var jQuery = jquery.NewJQuery
var jQMobile *js.Object
var window *js.Object = js.Global
var document *js.Object = js.Global.Get("document")

type Slide struct {
	URL   *url.URL
	Title string
}

var slides []*Slide

func main() {
	fmt.Printf("Before ready\n")
	jQuery(document).Ready(func() {
		go func() {
			fmt.Printf("Ready\n")
			resize()
			jQuery(window).Resize(resize)
			jQuery("#preview-toggle").On("click", previewToggle)

			if err := loadSlideShow(); err != nil {
				panic(fmt.Sprintf("Unable to load slide show: %s", err))
			}

			// Hide the spinner, and show normal content
			jQuery("#wait-message").Hide()
			jQuery("#ready-content").Show()
		}()
	})
}

func log(message string) {
	jQuery("#status").Append(fmt.Sprintf("<p>%s</p>", message))
}

func resize() {
	headerHeight := jQuery("#header").OuterHeight()
	bodyHeight := jQuery("body").Height()
	footerHeight := jQuery("#footer").OuterHeight()
	container := jQuery("#container")
	container.SetHeight(strconv.Itoa(bodyHeight - headerHeight - footerHeight))
	container.SetCss(map[string]int{"top": headerHeight})

	var previewWidth int
	preview := jQuery("#preview")
	if preview.Is(":visible") {
		previewWidth = jQuery("#preview").OuterWidth()
	}
	bodyWidth := jQuery("body").Width()
	content := jQuery("#content")
	content.SetWidth(strconv.Itoa(bodyWidth - previewWidth))
	// 	content.SetCss(map[string]int{"margin-left": previewWidth})
}

func previewToggle(event *js.Object) {
	event.Call("preventDefault")
	toggle := jQuery("#preview-toggle")
	preview := jQuery("#preview")
	if preview.Is(":visible") {
		preview.Hide()
		toggle.AddClass("fa-angle-down")
		toggle.RemoveClass("fa-angle-up")
	} else {
		preview.Show()
		toggle.AddClass("fa-angle-up")
		toggle.RemoveClass("fa-angle-down")
	}
	resize()
}

func loadSlideShow() error {
	resp, err := http.Get("/slides")
	if err != nil {
		return err
	}
	if resp.StatusCode != 200 {
		return fmt.Errorf("Unexpected HTTP status: %d\n", resp.StatusCode)
	}

	doc, err := html.Parse(resp.Body)
	if err != nil {
		return err
	}

	slides := make([]*Slide, 0, 5) // What slide show has fewer than 5 slides?

	var f func(*html.Node) error
	f = func(n *html.Node) error {
		if n.Type == html.ElementNode && n.Data == "a" {
			for _, attr := range n.Attr {
				if attr.Key == "href" {
					addr, err := url.Parse(attr.Val)
					if err != nil {
						return err
					}
					var title string
					if n.FirstChild.Type == html.TextNode {
						title = n.FirstChild.Data
					}
					slides = append(slides, &Slide{
						URL:   addr,
						Title: title,
					})
				}
			}
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			if err := f(c); err != nil {
				return err
			}
		}
		return nil
	}
	err = f(doc)
	return err
}
