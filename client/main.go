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
var window jquery.JQuery = jQuery(js.Global)
var document jquery.JQuery = jQuery(js.Global.Get("document"))

type Slide struct {
	Address string
	Title   string
	Body    []byte
	Ready   <-chan struct{}
}

var slides []*Slide
var currentSlide int
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
	document.Ready(func() {
		go func() {
			loadCSS()
			resize()
			window.Resize(resize)
			jQuery("#preview-toggle").On("click", previewToggle)
			jQuery("#fullscreen").On("click", fullScreen)
			jQuery("#handle").On("click", showHeader) // For touch screens
			jQuery("#handle").On("mouseover", showHeader)
			jQuery("#nav-prev").On("click", prevSlide)
			jQuery("#nav-next").On("click", nextSlide)
			jQuery("#preview").On("scroll", cachePreviews)
			window.On("keydown", handleKeypress)

			<-slideInitDone // Ensure we've finished loading the slides
			displaySlide(0)

			// Hide the spinner, and show normal content
			jQuery("#wait-message").Hide()
			jQuery("#ready-content").Show()
		}()
	})
}

var css string
var cssInitDone <-chan struct{}

func loadCSS() {
	done := make(chan struct{})
	cssInitDone = done
	go func() {
		cssurl := jQuery("link[rel='stylesheet']").First().Attr("href")
		resp, err := fetchURL(cssurl)
		if err != nil {
			panic(fmt.Sprintf("Error loading CSS: %s\n", err))
		}
		buf := new(bytes.Buffer)
		buf.ReadFrom(resp.Body)
		resp.Body.Close()
		css = buf.String()
		close(done)
	}()
}

func log(message string) {
	jQuery("#status").Append(fmt.Sprintf("<p>%s</p>", message))
}

func resize() {
	body := jQuery("body")
	preview := jQuery("#preview")

	bodyHeight := body.OuterHeight()
	bodyWidth := body.OuterWidth()
	headerHeight := jQuery("#header").OuterHeight()
	footerHeight := jQuery("#footer").OuterHeight()

	// Set Preview column dimensions
	thumb := jQuery("div.thumbnail").First()
	previewWidth := thumb.OuterWidth() + getCSSpx(thumb, "marginRight") + getCSSpx(thumb, "marginLeft") + scrollBarWidth
	previewHeight := bodyHeight - headerHeight - footerHeight
	preview.SetCss("top", headerHeight)
	preview.SetHeight(fmt.Sprintf("%dpx", previewHeight))
	preview.SetWidth(fmt.Sprintf("%dpx", previewWidth))

	// Set fullscreen switch dimensions
	fullscreen := jQuery("#fullscreen")
	fsWidth := bodyWidth
	if preview.Is(":visible") {
		fsWidth -= previewWidth
	}
	fullscreen.SetCss("top", headerHeight)
	fullscreen.SetCss("left", bodyWidth-fsWidth)
	fullscreen.SetHeight(fmt.Sprintf("%dpx", previewHeight))
	fullscreen.SetWidth(fmt.Sprintf("%dpx", fsWidth))
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
	resize()
}

func previewShow() {
	jQuery("#preview").Show()
	toggle := jQuery("#preview-toggle")
	toggle.AddClass("fa-angle-up")
	toggle.RemoveClass("fa-angle-down")
	cachePreviews()
	resize()
}

func cachePreviews() {
	preview := jQuery("#preview")
	top := preview.ScrollTop()
	bot := top + preview.Height()
	for _, e := range preview.Find("div.thumbnail").ToArray() {
		elem := jQuery(e)
		elemTop := elem.Offset().Top
		elemBot := elemTop + elem.Height()

		if elemTop <= bot && elemBot >= top {
			id := elem.Attr("id")
			idx, err := strconv.Atoi(strings.TrimPrefix(id, "preview-"))
			if err != nil {
				panic(fmt.Sprintf("Unexpected id `%s`\n", id))
			}
			cacheSlide(idx)
		}
	}
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
	resp.Body.Close()
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
	prev := jQuery("#nav-prev")
	next := jQuery("#nav-next")
	if idx > 0 {
		cacheSlide(idx - 1)
		prev.RemoveClass("disabled")
		go func() {
			slide := slides[idx-1]
			<-slide.Ready
			prev.Find("#nav-prev-title").SetHtml(slide.Title)
		}()
	} else {
		prev.AddClass("disabled")
		prev.Find("#nav-prev-title").SetHtml("Previous")
	}
	if idx < len(slides)-1 {
		cacheSlide(idx + 1)
		next.RemoveClass("disabled")
		go func() {
			slide := slides[idx+1]
			<-slide.Ready
			next.Find("#nav-next-title").SetHtml(slide.Title)
		}()
	} else {
		next.AddClass("disabled")
		next.Find("#nav-next-title").SetHtml("Next")
	}
	go func() {
		slide := slides[idx]
		<-slide.Ready // Wait until the cache is populated
		jQuery("#content").SetHtml(string(slide.Body))
		currentSlide = idx
	}()
}

func cacheSlide(idx int) {
	slide := slides[idx]
	if slide.Ready != nil {
		// Slide is already cached, or being cached. Nothing to do
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
			iframe := jQuery("<iframe></iframe>")
			iframe.AddClass("thumbnail")
			body := `
				<html>
					<head>
						<style>
				` + css + `
						</style>
					</head>
					</body>
						<div id="content">
			` + string(slide.Body) + `
						</div>
					</body>
				</html>
			`
			iframe.SetAttr("src", "data:text/html;charset=utf-8,"+encodeURL(body))
			preview.Find("iframe.thumbnail").ReplaceWith(iframe)
		}
	}()
}

func fullScreen() {
	jQuery("#header").SlideUp()
	jQuery("#footer").Hide()
	previewHide()
	jQuery("#fullscreen").Hide()
}

func showHeader() {
	jQuery("#header").SlideDown(func() {
		fmt.Printf("done!\n")
		jQuery("#fullscreen").Show()
		jQuery("#footer").Show()
		resize()
	})
}

// encodeURL encodes a url with spaces as %20, same as JavaScript's native
// encodeURI, but without relying on JS
func encodeURL(s string) string {
	t := &url.URL{Path: s}
	return t.String()
}

func prevSlide(event *js.Object) {
	event.Call("preventDefault")
	if currentSlide > 0 {
		displaySlide(currentSlide - 1)
	}
}

func nextSlide(event *js.Object) {
	event.Call("preventDefault")
	if currentSlide < len(slides)-1 {
		displaySlide(currentSlide + 1)
	} else {
		showHeader()
	}
}

func handleKeypress(event *js.Object) {
	switch event.Get("keyCode").Int() {
	case 32, 34, 40: // Space, PgDwn, DnArr
		fullScreen()
		if window.ScrollTop()+window.Height() == document.Height() {
			nextSlide(event)
		}
	case 33, 38: // PgUp, UpArr
		fullScreen()
		if window.ScrollTop() == 0 {
			prevSlide(event)
		}
	}
}
