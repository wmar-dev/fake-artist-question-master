// Package web holds the pages served to browsers.
package web

import "embed"

//go:embed index.html game.html app.js style.css favicon.svg
var FS embed.FS
