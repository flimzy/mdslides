// +build js

package main

import (
	"bytes"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/microcosm-cc/bluemonday"
	"github.com/russross/blackfriday"
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
	Address string
	Title   string
	Body    []byte
	Ready   <-chan struct{}
}

var slides []*Slide
var slideInitDone <-chan struct{}
var scrollBarWidth int = 17 // Better to read this from the browser, but how?

func init() {
	done := make(chan struct{})
	slideInitDone = done
	go func() {
		if err := loadSlideShow(); err != nil {
			panic(fmt.Sprintf("Unable to load slide show: %s", err))
		}
		close(done)
	}()
}

func main() {
	jQuery(document).Ready(func() {
		go func() {
			resize()
			jQuery(window).Resize(resize)
			jQuery("#preview-toggle").On("click", previewToggle)
			jQuery("#content").On("click", fullScreen)
			jQuery("#handle").On("click", showHeader) // For touch screens
			jQuery("#handle").On("mouseover", showHeader)

			<-slideInitDone // Ensure we've finished loading the slides
			fmt.Printf("We have %d slides\n", len(slides))
			displaySlide(0)

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
	body := jQuery("body")
	preview := jQuery("#preview")

	bodyHeight := body.OuterHeight()
	headerHeight := jQuery("#header").OuterHeight()
	footerHeight := jQuery("#footer").OuterHeight()

	// Set Preview column dimensions
	thumb := jQuery("div.thumbnail").First()
	previewWidth := thumb.OuterWidth() + getCSSpx(thumb, "marginRight") + getCSSpx(thumb, "marginLeft") + scrollBarWidth
	previewHeight := bodyHeight - headerHeight - footerHeight
	preview.SetCss("top", headerHeight)
	preview.SetHeight(fmt.Sprintf("%dpx", previewHeight))
	preview.SetWidth(fmt.Sprintf("%dpx", previewWidth))
}

func getCSSpx(elem jquery.JQuery, tag string) int {
	px := elem.Css(tag)
	val, err := strconv.Atoi(strings.TrimSuffix(px, "px"))
	if err != nil {
		panic(fmt.Sprintf("Cannot convert `%s` to integer: %s", px, err))
	}
	return val
}

func previewToggle(event *js.Object) {
	event.Call("preventDefault")
	if jQuery("#preview").Is(":visible") {
		previewHide()
	} else {
		previewShow()
	}
}

func previewHide() {
	jQuery("#preview").Hide()
	toggle := jQuery("#preview-toggle")
	toggle.AddClass("fa-angle-down")
	toggle.RemoveClass("fa-angle-up")
}

func previewShow() {
	jQuery("#preview").Show()
	toggle := jQuery("#preview-toggle")
	toggle.AddClass("fa-angle-up")
	toggle.RemoveClass("fa-angle-down")
}

func loadSlideShow() error {
	var err error
	var htmlDoc []byte
	for _, addr := range []string{"slides/index.md", "/slides/index.html", "/slides/index.htm", "/slides"} {
		var resp *http.Response
		resp, err = fetchURL(addr)
		if err == nil {
			htmlDoc, err = responseToHTML(resp)
			if err == nil {
				fmt.Printf("Successfully loaded %s\n", addr)
				break
			}
			fmt.Printf("Found %s, but can't convert to HTML: %s\n", addr, err)
		}
	}
	if err != nil {
		return err
	}

	doc, err := html.Parse(bytes.NewReader(htmlDoc))
	if err != nil {
		return err
	}

	slides = make([]*Slide, 0, 5) // What slide show has fewer than 5 slides?

	var f func(*html.Node) error
	f = func(n *html.Node) error {
		if n.Type == html.ElementNode && n.Data == "a" {
			for _, attr := range n.Attr {
				if attr.Key == "href" {
					addr, err := url.Parse(attr.Val)
					if err != nil {
						return err
					}
					if !strings.HasPrefix(addr.Path, "/") && addr.Host == "" {
						addr.Path = "/slides/" + addr.Path
					}
					var title string
					if n.FirstChild.Type == html.TextNode {
						title = n.FirstChild.Data
					}
					slides = append(slides, &Slide{
						Address: addr.String(),
						Title:   title,
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
	template := jQuery(".template").Find("#thumbnail")
	preview := jQuery("#preview")
	for i, _ := range slides {
		thumb := template.Clone()
		thumb.Find(".overlay").SetHtml(fmt.Sprintf("<h2>%d</h2>", i+1))
		thumb.SetAttr("id", fmt.Sprintf("preview-%d", i))
		preview.Append(thumb)
		thumb.Show()
		idx := i
		thumb.On("click", func(event *js.Object) {
			fmt.Printf("clicked on %d\n", idx)
			event.Call("preventDefault")
			displaySlide(idx)
		})
	}
	return err
}

func fetchURL(addr string) (*http.Response, error) {
	resp, err := http.Get(addr)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("Unexpected HTTP status %d fetching `%s`", resp.StatusCode, addr)
	}
	return resp, nil
}

// responseToHTML takes an http.Response, and attempts to return HTML.
// If the response represents an MD document, it converts it to HTML.
// If the conversion cannot be completed, it returns an error
func responseToHTML(resp *http.Response) ([]byte, error) {
	var rawHTML []byte
	buf := new(bytes.Buffer)
	buf.ReadFrom(resp.Body)
	switch ct := resp.Header.Get("Content-Type"); {
	case ct == "text/html" || strings.HasPrefix(ct, "text/html;"):
		rawHTML = buf.Bytes()
	case ct == "text/markdown" || strings.HasPrefix(ct, "text/markdown;"):
		rawHTML = blackfriday.MarkdownCommon(buf.Bytes())
	default:
		fmt.Printf("Trying to detect content type\n")
		switch ct := http.DetectContentType(buf.Bytes()); {
		case ct == "text/html" || strings.HasPrefix(ct, "text/html;"):
			rawHTML = buf.Bytes()
		case ct == "text/markdown" || strings.HasPrefix(ct, "text/markdown;"):
			rawHTML = blackfriday.MarkdownCommon(buf.Bytes())
		default:
			return []byte{}, fmt.Errorf("Unknown content type: %s", ct)
		}
	}
	return bluemonday.UGCPolicy().SanitizeBytes(rawHTML), nil
}

func displaySlide(idx int) {
	cacheSlide(idx)
	if idx > 0 {
		cacheSlide(idx - 1)
	}
	if idx < len(slides)-1 {
		cacheSlide(idx + 1)
	}
	go func() {
		slide := slides[idx]
		<-slide.Ready // Wait until the cache is populated
		jQuery("#content").SetHtml(string(slide.Body))
	}()
}

func cacheSlide(idx int) {
	slide := slides[idx]
	if len(slide.Body) != 0 {
		fmt.Printf("Slide #%d is already cached\n", idx)
		// Slide is already cached, nothing to do
		return
	}
	fmt.Printf("Slide #%d needs to be cached\n", idx)
	done := make(chan struct{})
	slide.Ready = done
	go func() {
		fmt.Printf("Caching slide #%d\n", idx)
		resp, err := fetchURL(slide.Address)
		if err != nil {
			panic(fmt.Sprintf("Error fetching slide #%d: %s", idx, err))
		}
		body, err := responseToHTML(resp)
		if err != nil {
			panic(fmt.Sprintf("Error converting slide #%d to HTML: %s", idx, err))
		}
		slide.Body = body
		fmt.Printf("done caching slide #%d\n", idx)
		close(done)
		if preview := jQuery(fmt.Sprintf("#preview-%d", idx)); !jquery.IsEmptyObject(preview) {
			iframe := jQuery(document.Call("createElement", "iframe"))
			iframe.AddClass("thumbnail")
			iframe.SetAttr("src", "data:text/html;charset=utf-8,"+url.QueryEscape(string(slide.Body)))
			preview.Find("iframe.thumbnail").ReplaceWith(iframe)
		}
	}()
}

func fullScreen() {
	jQuery("#header").SlideUp()
	jQuery("#footer").Hide()
	previewHide()
	jQuery("#container").Hide()
}

func showHeader() {
	jQuery("#header").SlideDown()
	jQuery("#container").Show()
	jQuery("#footer").Show()
}
