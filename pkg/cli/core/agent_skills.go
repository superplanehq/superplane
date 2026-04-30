package core

const SkillsRepository = "superplanehq/skills"

func SkillsInstallCommand(skill string) string {
	return "npx skills add " + SkillsRepository + " --skill " + skill
}

func AgentSkillsHint() string {
	return "Tip (AI agents): install SuperPlane skills: " + SkillsInstallCommand("superplane-cli")
}

func AgentSkillsHelp() string {
	return `AI agents: SuperPlane skills are available to guide correct YAML formats, field names, and workflows.
Install: ` + SkillsInstallCommand("superplane-cli") + `
More skills: superplane-canvas-builder, superplane-monitor`
}
