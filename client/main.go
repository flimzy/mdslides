// +build js

package main

import (
	"fmt"
	"strconv"

	"github.com/gopherjs/gopherjs/js"
	"github.com/gopherjs/jquery"
	//     "github.com/flimzy/jqeventrouter"
)

// Some spiffy shortcuts
var jQuery = jquery.NewJQuery
var jQMobile *js.Object
var window *js.Object = js.Global
var document *js.Object = js.Global.Get("document")

func main() {
	fmt.Printf("Before ready\n")
	jQuery(document).Ready(func() {
		go func() {
			fmt.Printf("Ready\n")
			resize()
			jQuery(window).Resize(resize)
			jQuery("#preview-toggle").On("click", previewToggle)
			jQuery("#open-slideshow").On("click", openSlideshow)

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
	fmt.Printf("Header height = %d\nBody height = %d\nFooter height = %d\n", headerHeight, bodyHeight, footerHeight)
	container := jQuery("#container")
	container.SetHeight(strconv.Itoa(bodyHeight - headerHeight - footerHeight))
	container.SetCss(map[string]int{"top": headerHeight})

	var previewWidth int
	preview := jQuery("#preview")
	if preview.Is(":visible") {
		previewWidth = jQuery("#preview").OuterWidth()
	}
	bodyWidth := jQuery("body").Width()
	fmt.Printf("Preview width = %d\nBody width = %d\n", previewWidth, bodyWidth)
	content := jQuery("#content")
	content.SetWidth(strconv.Itoa(bodyWidth - previewWidth))
	// 	content.SetCss(map[string]int{"margin-left": previewWidth})
}

func previewToggle(event *js.Object) {
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
	event.Call("preventDefault")
}

func openSlideshow(event *js.Object) {
	fmt.Printf("You pressed the open button\n")
	event.Call("preventDefault")
}
