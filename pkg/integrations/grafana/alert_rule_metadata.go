package grafana

import (
	"strings"

	"github.com/superplanehq/superplane/pkg/core"
)

type AlertRuleNodeMetadata struct {
	AlertRuleTitle string `json:"alertRuleTitle,omitempty" mapstructure:"alertRuleTitle"`
	FolderTitle    string `json:"folderTitle,omitempty" mapstructure:"folderTitle"`
}

func storeAlertRuleNodeMetadata(ctx core.SetupContext, alertRuleUID string, folderUID string) {
	trimmedAlertRuleUID := strings.TrimSpace(alertRuleUID)
	trimmedFolderUID := strings.TrimSpace(folderUID)
	if ctx.Metadata == nil || ctx.HTTP == nil || (trimmedAlertRuleUID == "" && trimmedFolderUID == "") {
		return
	}

	client, err := NewClient(ctx.HTTP, ctx.Integration, true)
	if err != nil {
		return
	}

	metadata := AlertRuleNodeMetadata{}

	if trimmedAlertRuleUID != "" {
		rule, err := client.GetAlertRule(trimmedAlertRuleUID)
		if err == nil {
			if title, ok := rule["title"].(string); ok {
				metadata.AlertRuleTitle = strings.TrimSpace(title)
			}
		}
	}

	if trimmedFolderUID != "" {
		folders, err := client.ListFolders()
		if err == nil {
			for _, folder := range folders {
				if strings.TrimSpace(folder.UID) != trimmedFolderUID {
					continue
				}

				metadata.FolderTitle = strings.TrimSpace(folder.Title)
				break
			}
		}
	}

	if metadata.AlertRuleTitle == "" && metadata.FolderTitle == "" {
		return
	}

	_ = ctx.Metadata.Set(metadata)
}
