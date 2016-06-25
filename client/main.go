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
			jQuery("#wait-message").Hide()
			jQuery("#ready-content").Show()
		}()
	})
}

func log(message string) {
	jQuery("#status").Append(fmt.Sprintf("<p>%s</p>", message))
}

func resize() {
	headerHeight := jQuery("#header").Height()
	bodyHeight := jQuery("body").Height()
	footerHeight := jQuery("#footer").Height()
	fmt.Printf("Header height = %d\nBody height = %d\nFooter height = %d\n", headerHeight, bodyHeight, footerHeight)
	container := jQuery("#container")
	container.SetHeight(strconv.Itoa(bodyHeight - headerHeight - footerHeight))
	container.SetCss(map[string]int{"top": headerHeight})

	previewWidth := jQuery("#preview").Width()
	bodyWidth := jQuery("body").Width()
	fmt.Printf("Preview width = %d\nBody width = %d\n", previewWidth, bodyWidth)
	content := jQuery("#content")
	content.SetWidth(strconv.Itoa(bodyWidth - previewWidth))
	// 	content.SetCss(map[string]int{"margin-left": previewWidth})
}
