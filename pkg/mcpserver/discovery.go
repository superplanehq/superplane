package mcpserver

func ProtectedResourceMetadata(origin string) map[string]any {
	resource := ResourceURL(origin)
	return map[string]any{
		"resource":              resource,
		"authorization_servers": []string{origin},
	}
}

func AuthorizationServerMetadata(origin string) map[string]any {
	return map[string]any{
		"issuer":                                origin,
		"authorization_endpoint":                origin + PathAuthorize,
		"token_endpoint":                        origin + PathToken,
		"registration_endpoint":                 origin + PathRegister,
		"response_types_supported":              []string{"code"},
		"grant_types_supported":                 []string{"authorization_code", "refresh_token"},
		"code_challenge_methods_supported":      []string{"S256"},
		"token_endpoint_auth_methods_supported": []string{"none"},
		"client_id_metadata_document_supported": true,
		"scopes_supported":                      GrantedScopes,
	}
}

func WWWAuthenticate(origin string) string {
	return `Bearer realm="mcp", resource_metadata="` + origin + PathProtectedResourceMCP + `"`
}
