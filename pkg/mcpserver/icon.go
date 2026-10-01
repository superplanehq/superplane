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

func dataPNGURI(raw []byte) string {
	return "data:image/png;base64," + base64.StdEncoding.EncodeToString(raw)
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
		},
	}
}
