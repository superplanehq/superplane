package runner

import "fmt"

// SystemSkillTemplate is the current SuperPlane definition of a workspace skill.
// Increment Version when changing its instructions or frontmatter.
type SystemSkillTemplate struct {
	ID           string
	Version      int32
	Title        string
	Description  string
	Instructions string
}

// SystemSkillTemplateByID shares the existing default instructions with restores.
// Installation of these templates during onboarding is a separate step.
func SystemSkillTemplateByID(id string) (SystemSkillTemplate, bool) {
	if id != "refine-task" {
		return SystemSkillTemplate{}, false
	}
	return SystemSkillTemplate{
		ID: "refine-task", Version: 5, Title: "Refine task",
		Description:  "Refine a task into a clear implementation plan.",
		Instructions: PlanningSessionUserPromptMarkdown(),
	}, true
}

// Markdown retains the installed command name when restoring a template.
func (t SystemSkillTemplate) Markdown(commandName string) string {
	return fmt.Sprintf("---\nname: %s\ntitle: %s\ndescription: %s\n---\n\n%s", commandName, t.Title, t.Description, t.Instructions)
}
