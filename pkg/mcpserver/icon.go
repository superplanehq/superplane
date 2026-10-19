package mcpserver

import (
	_ "embed"
	"encoding/base64"
)

const ServerWebsiteURL = "https://superplane.com"

//go:embed icon-light.png
var iconLightPNG []byte

//go:embed icon-dark.png
var iconDarkPNG []byte

//go:embed icon-light-192.png
var iconLight192PNG []byte

//go:embed icon-dark-192.png
var iconDark192PNG []byte

func dataPNGURI(raw []byte) string {
	return "data:image/png;base64," + base64.StdEncoding.EncodeToString(raw)
}

func icon(size string, raw []byte, theme string) map[string]any {
	return map[string]any{
		"src":      dataPNGURI(raw),
		"mimeType": "image/png",
		"sizes":    []string{size},
		"theme":    theme,
	}
}

func ServerInfo() map[string]any {
	return map[string]any{
		"name":       ServerName,
		"title":      ServerName,
		"version":    ServerVersion,
		"websiteUrl": ServerWebsiteURL,
		"icons": []map[string]any{
			icon("96x96", iconLightPNG, "light"),
			icon("96x96", iconDarkPNG, "dark"),
			icon("192x192", iconLight192PNG, "light"),
			icon("192x192", iconDark192PNG, "dark"),
		},
	}
}
