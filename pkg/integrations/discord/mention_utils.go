package discord

import "strings"

func isBotAuthor(message map[string]any) bool {
	author, ok := message["author"].(map[string]any)
	if !ok {
		return false
	}

	isBot, ok := author["bot"].(bool)
	return ok && isBot
}

func messageMentionsBot(message map[string]any, botID, username string) bool {
	mentions, ok := message["mentions"].([]any)
	if ok {
		for _, mention := range mentions {
			mentionMap, ok := mention.(map[string]any)
			if !ok {
				continue
			}

			mentionID, _ := mentionMap["id"].(string)
			if botID != "" && mentionID == botID {
				return true
			}

			mentionUsername, _ := mentionMap["username"].(string)
			if username != "" && strings.EqualFold(mentionUsername, username) {
				return true
			}
		}
	}

	content, _ := message["content"].(string)
	if content == "" {
		return false
	}

	if botID != "" {
		if strings.Contains(content, "<@"+botID+">") || strings.Contains(content, "<@!"+botID+">") {
			return true
		}
	}

	if username != "" && strings.Contains(strings.ToLower(content), strings.ToLower("@"+username)) {
		return true
	}

	return false
}

func discordMessageToMap(message Message) map[string]any {
	result := map[string]any{
		"id":         message.ID,
		"channel_id": message.ChannelID,
		"content":    message.Content,
		"timestamp":  message.Timestamp,
		"author": map[string]any{
			"id":            message.Author.ID,
			"username":      message.Author.Username,
			"discriminator": message.Author.Discriminator,
			"bot":           message.Author.Bot,
		},
	}

	if message.GuildID != "" {
		result["guild_id"] = message.GuildID
	}

	if len(message.Mentions) > 0 {
		mentions := make([]map[string]any, 0, len(message.Mentions))
		for _, mention := range message.Mentions {
			mentions = append(mentions, map[string]any{
				"id":            mention.ID,
				"username":      mention.Username,
				"discriminator": mention.Discriminator,
				"bot":           mention.Bot,
			})
		}
		result["mentions"] = mentions
	}

	return result
}
