package mcpserver

import (
	_ "embed"
	"encoding/base64"
)

const ServerWebsiteURL = "https://superplane.com"

const dataPNGPrefix = "data:image/png;base64,"

//go:embed icon-light.png
var iconLightPNG []byte

//go:embed icon-dark.png
var iconDarkPNG []byte

//go:embed icon-192.png
var icon192PNG []byte

func dataPNGURI(raw []byte) string {
	return dataPNGPrefix + base64.StdEncoding.EncodeToString(raw)
}

func ServerInfo() map[string]any {
	return map[string]any{
		"name":       ServerName,
		"title":      ServerName,
		"version":    ServerVersion,
		"websiteUrl": ServerWebsiteURL,
		"icons": []map[string]any{
			{
				"src":      dataPNGURI(iconLightPNG),
				"mimeType": "image/png",
				"sizes":    []string{"96x96"},
				"theme":    "light",
			},
			{
				"src":      dataPNGURI(iconDarkPNG),
				"mimeType": "image/png",
				"sizes":    []string{"96x96"},
				"theme":    "dark",
			},
			{
				"src":      dataPNGURI(icon192PNG),
				"mimeType": "image/png",
				"sizes":    []string{"192x192"},
			},
		},
	}
}
