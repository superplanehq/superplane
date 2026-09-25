package factories

import (
	"encoding/json"
	"fmt"
	"maps"
	"regexp"
	"slices"
	"strings"

	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/factories"
	"google.golang.org/protobuf/proto"
)

const (
	intakeLabelFilterInclude = "include"
	intakeLabelFilterExclude = "exclude"

	intakeAssignmentAny        = "any"
	intakeAssignmentAssigned   = "assigned"
	intakeAssignmentUnassigned = "unassigned"

	intakeAssignedCondition   = "len(root().data.issue.assignees) > 0"
	intakeUnassignedCondition = "len(root().data.issue.assignees) == 0"

	// Conditions built before the filters were valid expr-lang. Canvases
	// created back then still hold them, so keep reading them.
	intakeLegacyAssignedCondition   = "size(root().data.issue.assignees) > 0"
	intakeLegacyUnassignedCondition = "size(root().data.issue.assignees) == 0"

	intakeAuthorAccessCondition = `root().data.issue.author_association in ["COLLABORATOR", "MEMBER", "OWNER"]`

	// Label that a user adds to an issue to hand it to the factory.
	intakeSuperplaneLabel = "superplane"

	// Conditions are joined with `&&`, which binds tighter than `||`, so a
	// compound condition has to carry its own parentheses. The `||` also has
	// to short-circuit: only a `labeled` payload carries a label to read.
	intakeSuperplaneLabelCondition = `(root().data.action != "labeled" || (root().data.issue.state == "open" && root().data.label.name == "` +
		intakeSuperplaneLabel + `"))`

	intakeJiraAssignedCondition   = "root().data.issue.fields.assignee != null"
	intakeJiraUnassignedCondition = "root().data.issue.fields.assignee == null"

	intakeMetadataJiraMoveOnComplete   = "jiraMoveOnComplete"
	intakeMetadataJiraCompletionColumn = "jiraCompletionColumn"

	intakeSentryActionCreated    = "created"
	intakeSentryActionUnresolved = "unresolved"
	intakeSentryActionAssigned   = "assigned"
)

// intakeSettings is what a user can change about an intake without editing the
// canvas by hand. Filter fields are stored in, and read back from, the filter
// expression: the graph is what the workers run, so nothing is kept twice.
type intakeSettings struct {
	ConfidencePct     int
	Labels            []string
	LabelFilterMode   string
	Assignment        string
	AuthorsWithAccess bool
	NewIssues         bool
	ReopenedIssues    bool
	// Create a task when somebody adds the "superplane" label to an open issue.
	SuperplaneLabelAdded bool
	// Move the originating Jira issue when the work order completes.
	JiraMoveOnComplete bool
	// Jira status name to move the issue to. Empty means the Done column.
	JiraCompletionColumn string
	// Listen for created Sentry issues.
	SentryNewIssues bool
	// Listen for Sentry issues that become unresolved.
	SentryRegressedIssues bool
	// Listen for assigned Sentry issues.
	SentryAssignedIssues bool
	// Issue levels that still create a task. Empty means every level.
	SentryLevels []string
	// Skip Productive.io key tasks (milestones). Productive task intakes only.
	ExcludeKeyTasks bool
	// Severities that still create a task. Empty means every severity.
	// Dependabot alert intakes only.
	DependabotSeverities []string
	// Task list ids that still create a task. Empty means every task list.
	// Productive task intakes only.
	TaskListIDs []string
}

var intakeSentryKnownLevels = []string{"fatal", "error", "warning", "info", "debug"}

var intakeDependabotKnownSeverities = []string{"critical", "high", "medium", "low"}

func dependabotIntakeActions() []any {
	return []any{"created", "reopened", "reintroduced"}
}

func defaultIntakeSettings() intakeSettings {
	return intakeSettings{
		ConfidencePct:        DefaultIntakeConfidencePct,
		Labels:               []string{},
		LabelFilterMode:      intakeLabelFilterInclude,
		Assignment:           intakeAssignmentAny,
		NewIssues:            true,
		ReopenedIssues:       true,
		SuperplaneLabelAdded: true,
		JiraMoveOnComplete:   true,
		ExcludeKeyTasks:      true,
		// AuthorsWithAccess is off by default: false.
	}
}

func defaultJiraIntakeSettings() intakeSettings {
	settings := defaultIntakeSettings()
	settings.ReopenedIssues = true
	settings.SuperplaneLabelAdded = false
	settings.JiraMoveOnComplete = true
	return settings
}

func defaultSentryIntakeSettings() intakeSettings {
	settings := defaultIntakeSettings()
	settings.SentryNewIssues = true
	settings.SentryRegressedIssues = false
	settings.SentryAssignedIssues = false
	settings.SentryLevels = []string{}
	return settings
}

func defaultProductiveIntakeSettings() intakeSettings {
	settings := defaultIntakeSettings()
	settings.ExcludeKeyTasks = true
	return settings
}

func defaultDependabotIntakeSettings() intakeSettings {
	settings := defaultIntakeSettings()
	settings.DependabotSeverities = []string{}
	return settings
}

func intakeSourceHasFilterNode(source string) bool {
	return source == models.FactoryIntakeSourceGitHubIssues ||
		source == models.FactoryIntakeSourceJiraIssues ||
		source == models.FactoryIntakeSourceSentryExceptions ||
		source == models.FactoryIntakeSourceProductiveTasks ||
		source == models.FactoryIntakeSourceDependabotAlerts
}

func (s intakeSettings) normalized() intakeSettings {
	s.ConfidencePct = clampIntakeConfidence(s.ConfidencePct)

	if s.LabelFilterMode != intakeLabelFilterExclude {
		s.LabelFilterMode = intakeLabelFilterInclude
	}
	if s.Assignment != intakeAssignmentAssigned && s.Assignment != intakeAssignmentUnassigned {
		s.Assignment = intakeAssignmentAny
	}

	labels := make([]string, 0, len(s.Labels))
	for _, label := range s.Labels {
		label = strings.TrimSpace(label)
		if label != "" && !slices.Contains(labels, label) {
			labels = append(labels, label)
		}
	}
	s.Labels = labels
	s.JiraCompletionColumn = strings.TrimSpace(s.JiraCompletionColumn)
	s.SentryLevels = normalizeSentryLevels(s.SentryLevels)
	s.DependabotSeverities = normalizeDependabotSeverities(s.DependabotSeverities)
	s.TaskListIDs = normalizeTaskListIDs(s.TaskListIDs)

	return s
}

func normalizeTaskListIDs(ids []string) []string {
	normalized := make([]string, 0, len(ids))
	for _, id := range ids {
		id = strings.TrimSpace(id)
		if id == "" || slices.Contains(normalized, id) {
			continue
		}
		normalized = append(normalized, id)
	}
	return normalized
}

func normalizeSentryLevels(levels []string) []string {
	selected := make(map[string]bool, len(levels))
	for _, level := range levels {
		selected[strings.ToLower(strings.TrimSpace(level))] = true
	}

	normalized := make([]string, 0, len(intakeSentryKnownLevels))
	for _, level := range intakeSentryKnownLevels {
		if selected[level] {
			normalized = append(normalized, level)
		}
	}
	return normalized
}

func normalizeDependabotSeverities(severities []string) []string {
	selected := make(map[string]bool, len(severities))
	for _, severity := range severities {
		selected[strings.ToLower(strings.TrimSpace(severity))] = true
	}

	normalized := make([]string, 0, len(intakeDependabotKnownSeverities))
	for _, severity := range intakeDependabotKnownSeverities {
		if selected[severity] {
			normalized = append(normalized, severity)
		}
	}
	if len(normalized) == len(intakeDependabotKnownSeverities) {
		return []string{}
	}
	return normalized
}

// intakeFilterExpressionFor builds the gate in front of the work order from
// the filters the source supports. An empty filter set is `true`, so every
// matching event still creates a work order.
func intakeFilterExpressionFor(source string, settings intakeSettings) string {
	settings = settings.normalized()
	switch source {
	case models.FactoryIntakeSourceGitHubIssues:
		return intakeGitHubFilterExpression(settings)
	case models.FactoryIntakeSourceJiraIssues:
		return intakeJiraFilterExpression(settings)
	case models.FactoryIntakeSourceSentryExceptions:
		return intakeSentryFilterExpression(settings)
	case models.FactoryIntakeSourceProductiveTasks:
		return intakeProductiveFilterExpression(settings)
	case models.FactoryIntakeSourceDependabotAlerts:
		return intakeDependabotFilterExpression(settings)
	default:
		return "true"
	}
}

func intakeDependabotFilterExpression(settings intakeSettings) string {
	if len(settings.DependabotSeverities) == 0 {
		return "true"
	}

	severities, err := json.Marshal(settings.DependabotSeverities)
	if err != nil {
		return "true"
	}

	return fmt.Sprintf(`(root().data.alert.security_advisory.severity ?? "") in %s`, severities)
}

func intakeGitHubFilterExpression(settings intakeSettings) string {
	conditions := []string{}
	if len(settings.Labels) > 0 {
		if labels, err := json.Marshal(settings.Labels); err == nil {
			matches := fmt.Sprintf("any(root().data.issue.labels, .name in %s)", labels)
			if settings.LabelFilterMode == intakeLabelFilterExclude {
				matches = fmt.Sprintf("!(%s)", matches)
			}
			conditions = append(conditions, matches)
		}
	}

	switch settings.Assignment {
	case intakeAssignmentAssigned:
		conditions = append(conditions, intakeAssignedCondition)
	case intakeAssignmentUnassigned:
		conditions = append(conditions, intakeUnassignedCondition)
	}

	if settings.SuperplaneLabelAdded {
		conditions = append(conditions, intakeSuperplaneLabelCondition)
	}

	if len(conditions) == 0 {
		return "true"
	}

	return strings.Join(conditions, " && ")
}

func intakeJiraFilterExpression(settings intakeSettings) string {
	conditions := []string{}
	if len(settings.Labels) > 0 {
		if labels, err := json.Marshal(settings.Labels); err == nil {
			matches := fmt.Sprintf("any(root().data.issue.fields.labels, # in %s)", labels)
			if settings.LabelFilterMode == intakeLabelFilterExclude {
				matches = fmt.Sprintf("!(%s)", matches)
			}
			conditions = append(conditions, matches)
		}
	}

	switch settings.Assignment {
	case intakeAssignmentAssigned:
		conditions = append(conditions, intakeJiraAssignedCondition)
	case intakeAssignmentUnassigned:
		conditions = append(conditions, intakeJiraUnassignedCondition)
	}

	if len(conditions) == 0 {
		return "true"
	}

	return strings.Join(conditions, " && ")
}

func intakeSentryFilterExpression(settings intakeSettings) string {
	if len(settings.SentryLevels) == 0 {
		return "true"
	}

	levels, err := json.Marshal(settings.SentryLevels)
	if err != nil {
		return "true"
	}

	return fmt.Sprintf(`(root().data.data.issue?.level ?? "") in %s`, levels)
}

const intakeProductiveExcludeKeyTasksCondition = "root().data.data.attributes.type_id != 3"

func intakeProductiveFilterExpression(settings intakeSettings) string {
	conditions := []string{}
	if settings.ExcludeKeyTasks {
		conditions = append(conditions, intakeProductiveExcludeKeyTasksCondition)
	}
	if condition := intakeProductiveTaskListCondition(settings.TaskListIDs); condition != "" {
		conditions = append(conditions, condition)
	}
	if len(conditions) == 0 {
		return "true"
	}
	return strings.Join(conditions, " && ")
}

func intakeProductiveTaskListCondition(ids []string) string {
	if len(ids) == 0 {
		return ""
	}
	encoded, err := json.Marshal(ids)
	if err != nil {
		return ""
	}
	return fmt.Sprintf(`(root().data.data.relationships.task_list.data.id ?? "") in %s`, encoded)
}

func intakeTriggerActionsFor(settings intakeSettings) []any {
	actions := []any{}
	if settings.NewIssues {
		actions = append(actions, "opened")
	}
	if settings.ReopenedIssues {
		actions = append(actions, "reopened")
	}
	if settings.SuperplaneLabelAdded {
		actions = append(actions, "labeled")
	}
	return actions
}

func intakeTriggerEventsFor(settings intakeSettings) []any {
	events := []any{}
	if settings.NewIssues {
		events = append(events, "created")
	}
	if settings.ReopenedIssues {
		events = append(events, "updated")
	}
	return events
}

func intakeSentryActionsFor(settings intakeSettings) []any {
	actions := []any{}
	if settings.SentryNewIssues {
		actions = append(actions, intakeSentryActionCreated)
	}
	if settings.SentryRegressedIssues {
		actions = append(actions, intakeSentryActionUnresolved)
	}
	if settings.SentryAssignedIssues {
		actions = append(actions, intakeSentryActionAssigned)
	}
	return actions
}

func intakeSettingsChangeTrigger(source string, current, updated intakeSettings) bool {
	if source == models.FactoryIntakeSourceJiraIssues {
		return current.NewIssues != updated.NewIssues ||
			current.ReopenedIssues != updated.ReopenedIssues
	}
	if source == models.FactoryIntakeSourceSentryExceptions {
		return current.SentryNewIssues != updated.SentryNewIssues ||
			current.SentryRegressedIssues != updated.SentryRegressedIssues ||
			current.SentryAssignedIssues != updated.SentryAssignedIssues
	}
	return current.NewIssues != updated.NewIssues ||
		current.ReopenedIssues != updated.ReopenedIssues ||
		current.SuperplaneLabelAdded != updated.SuperplaneLabelAdded
}

func intakeSettingsChangeFilters(current, updated intakeSettings) bool {
	if current.SuperplaneLabelAdded != updated.SuperplaneLabelAdded {
		return true
	}
	if current.LabelFilterMode != updated.LabelFilterMode {
		return true
	}
	if current.Assignment != updated.Assignment {
		return true
	}
	if current.AuthorsWithAccess != updated.AuthorsWithAccess {
		return true
	}
	if !slices.Equal(current.Labels, updated.Labels) {
		return true
	}
	if !slices.Equal(current.SentryLevels, updated.SentryLevels) {
		return true
	}
	if !slices.Equal(current.DependabotSeverities, updated.DependabotSeverities) {
		return true
	}
	if !slices.Equal(current.TaskListIDs, updated.TaskListIDs) {
		return true
	}
	return current.ExcludeKeyTasks != updated.ExcludeKeyTasks
}

// The second alternative is the expression built before the label filter was
// valid expr-lang. Canvases created back then still hold it, so keep reading
// it; the next save rewrites the node with the `any(...)` form.
var intakeLabelsPattern = regexp.MustCompile(
	`(!\()?(?:any\(root\(\)\.data\.issue\.labels, \.name in|root\(\)\.data\.issue\.labels\.exists\(label, label\.name in) (\[[^\]]*\])\)`,
)

var intakeJiraLabelsPattern = regexp.MustCompile(
	`(!\()?any\(root\(\)\.data\.issue\.fields\.labels, # in (\[[^\]]*\])\)`,
)

var intakeSentryLevelsPattern = regexp.MustCompile(
	`\(root\(\)\.data\.data\.issue\?\.level \?\? ""\) in (\[[^\]]*\])`,
)

var intakeDependabotSeveritiesPattern = regexp.MustCompile(
	`\(root\(\)\.data\.alert\.security_advisory\.severity \?\? ""\) in (\[[^\]]*\])`,
)

var intakeProductiveTaskListsPattern = regexp.MustCompile(
	`\(root\(\)\.data\.data\.relationships\.task_list\.data\.id \?\? ""\) in (\[[^\]]*\])`,
)

// intakeSettingsFromGraph reads the settings back out of the filter
// expression. A hand-edited expression that no longer matches reports defaults
// rather than a wrong value.
func intakeSettingsFromGraph(source string, graph intakeGraph, spec models.LiveCanvasSpec) intakeSettings {
	settings := defaultIntakeSettings()
	switch source {
	case models.FactoryIntakeSourceJiraIssues:
		settings = defaultJiraIntakeSettings()
	case models.FactoryIntakeSourceSentryExceptions:
		settings = defaultSentryIntakeSettings()
	case models.FactoryIntakeSourceProductiveTasks:
		settings = defaultProductiveIntakeSettings()
	case models.FactoryIntakeSourceDependabotAlerts:
		settings = defaultDependabotIntakeSettings()
	}
	settings.ConfidencePct = graph.ConfidencePct

	trigger := findIntakeNode(spec.Nodes, graph.TriggerNodeID)
	if trigger != nil {
		switch source {
		case models.FactoryIntakeSourceJiraIssues:
			events := configurationStrings(trigger.Configuration["events"])
			settings.NewIssues = slices.Contains(events, "created")
			settings.ReopenedIssues = slices.Contains(events, "updated")
			settings = jiraCompletionSettingsFromMetadata(trigger.Metadata, settings)
		case models.FactoryIntakeSourceSentryExceptions:
			actions := configurationStrings(trigger.Configuration["actions"])
			settings.SentryNewIssues = slices.Contains(actions, intakeSentryActionCreated)
			settings.SentryRegressedIssues = slices.Contains(actions, intakeSentryActionUnresolved)
			settings.SentryAssignedIssues = slices.Contains(actions, intakeSentryActionAssigned)
		default:
			if source == models.FactoryIntakeSourceDependabotAlerts {
				break
			}
			actions := configurationStrings(trigger.Configuration["actions"])
			settings.NewIssues = slices.Contains(actions, "opened")
			settings.ReopenedIssues = slices.Contains(actions, "reopened")
			settings.SuperplaneLabelAdded = slices.Contains(actions, "labeled")
		}
	}

	filter := findIntakeNode(spec.Nodes, graph.FilterNodeID)
	if filter == nil {
		return settings
	}

	expression, _ := filter.Configuration["expression"].(string)
	if expression == "" {
		return settings
	}

	if source == models.FactoryIntakeSourceDependabotAlerts {
		if match := intakeDependabotSeveritiesPattern.FindStringSubmatch(expression); match != nil {
			var severities []string
			if err := json.Unmarshal([]byte(match[1]), &severities); err == nil {
				settings.DependabotSeverities = severities
			}
		}
		return settings.normalized()
	}

	if source == models.FactoryIntakeSourceSentryExceptions {
		if match := intakeSentryLevelsPattern.FindStringSubmatch(expression); match != nil {
			var levels []string
			if err := json.Unmarshal([]byte(match[1]), &levels); err == nil {
				settings.SentryLevels = levels
			}
		}

		return settings.normalized()
	}

	if source == models.FactoryIntakeSourceProductiveTasks {
		settings.ExcludeKeyTasks = strings.Contains(expression, intakeProductiveExcludeKeyTasksCondition)
		if match := intakeProductiveTaskListsPattern.FindStringSubmatch(expression); match != nil {
			var ids []string
			if err := json.Unmarshal([]byte(match[1]), &ids); err == nil {
				settings.TaskListIDs = ids
			}
		}
		return settings.normalized()
	}

	if source == models.FactoryIntakeSourceJiraIssues {
		if match := intakeJiraLabelsPattern.FindStringSubmatch(expression); match != nil {
			var labels []string
			if err := json.Unmarshal([]byte(match[2]), &labels); err == nil {
				settings.Labels = labels
				if match[1] != "" {
					settings.LabelFilterMode = intakeLabelFilterExclude
				}
			}
		}

		switch {
		case strings.Contains(expression, intakeJiraUnassignedCondition):
			settings.Assignment = intakeAssignmentUnassigned
		case strings.Contains(expression, intakeJiraAssignedCondition):
			settings.Assignment = intakeAssignmentAssigned
		}

		return settings.normalized()
	}

	if match := intakeLabelsPattern.FindStringSubmatch(expression); match != nil {
		var labels []string
		if err := json.Unmarshal([]byte(match[2]), &labels); err == nil {
			settings.Labels = labels
			if match[1] != "" {
				settings.LabelFilterMode = intakeLabelFilterExclude
			}
		}
	}

	switch {
	case strings.Contains(expression, intakeUnassignedCondition),
		strings.Contains(expression, intakeLegacyUnassignedCondition):
		settings.Assignment = intakeAssignmentUnassigned
	case strings.Contains(expression, intakeAssignedCondition),
		strings.Contains(expression, intakeLegacyAssignedCondition):
		settings.Assignment = intakeAssignmentAssigned
	}

	settings.AuthorsWithAccess = graph.AuthorPermissionNodeID != "" ||
		strings.Contains(expression, intakeAuthorAccessCondition)

	return settings.normalized()
}

func serializeIntakeSettings(source string, settings intakeSettings) *pb.FactoryIntake_Settings {
	serialized := &pb.FactoryIntake_Settings{
		ConfidencePct:        int32(settings.ConfidencePct),
		Labels:               settings.Labels,
		LabelFilterMode:      serializeIntakeLabelFilterMode(settings.LabelFilterMode),
		Assignment:           serializeIntakeAssignment(settings.Assignment),
		AuthorsWithAccess:    settings.AuthorsWithAccess,
		NewIssues:            proto.Bool(settings.NewIssues),
		ReopenedIssues:       proto.Bool(settings.ReopenedIssues),
		SuperplaneLabelAdded: proto.Bool(settings.SuperplaneLabelAdded),
	}
	if source == models.FactoryIntakeSourceJiraIssues {
		serialized.JiraMoveOnComplete = proto.Bool(settings.JiraMoveOnComplete)
		serialized.JiraCompletionColumn = settings.JiraCompletionColumn
	}
	if source == models.FactoryIntakeSourceSentryExceptions {
		serialized.SentryNewIssues = proto.Bool(settings.SentryNewIssues)
		serialized.SentryRegressedIssues = proto.Bool(settings.SentryRegressedIssues)
		serialized.SentryAssignedIssues = proto.Bool(settings.SentryAssignedIssues)
		serialized.SentryLevels = settings.SentryLevels
	}
	if source == models.FactoryIntakeSourceProductiveTasks {
		serialized.ExcludeKeyTasks = proto.Bool(settings.ExcludeKeyTasks)
		serialized.TaskListIds = settings.TaskListIDs
	}
	if source == models.FactoryIntakeSourceDependabotAlerts {
		serialized.DependabotSeverities = settings.DependabotSeverities
	}
	return serialized
}

// parseIntakeSettings merges a request over what the graph already says, so a
// caller that leaves an enum unspecified does not reset it.
func parseIntakeSettings(current intakeSettings, requested *pb.FactoryIntake_Settings) intakeSettings {
	if requested == nil {
		return current
	}

	updated := current
	updated.ConfidencePct = int(requested.GetConfidencePct())
	updated.Labels = requested.GetLabels()

	if mode := requested.GetLabelFilterMode(); mode != pb.FactoryIntake_Settings_LABEL_FILTER_MODE_UNSPECIFIED {
		updated.LabelFilterMode = parseIntakeLabelFilterMode(mode)
	}
	if assignment := requested.GetAssignment(); assignment != pb.FactoryIntake_Settings_ASSIGNMENT_UNSPECIFIED {
		updated.Assignment = parseIntakeAssignment(assignment)
	}
	updated.AuthorsWithAccess = requested.GetAuthorsWithAccess()
	if requested.NewIssues != nil {
		updated.NewIssues = requested.GetNewIssues()
	}
	if requested.ReopenedIssues != nil {
		updated.ReopenedIssues = requested.GetReopenedIssues()
	}
	if requested.SuperplaneLabelAdded != nil {
		updated.SuperplaneLabelAdded = requested.GetSuperplaneLabelAdded()
	}
	if requested.JiraMoveOnComplete != nil {
		updated.JiraMoveOnComplete = requested.GetJiraMoveOnComplete()
	}
	updated.JiraCompletionColumn = strings.TrimSpace(requested.GetJiraCompletionColumn())
	if requested.SentryNewIssues != nil {
		updated.SentryNewIssues = requested.GetSentryNewIssues()
	}
	if requested.SentryRegressedIssues != nil {
		updated.SentryRegressedIssues = requested.GetSentryRegressedIssues()
	}
	if requested.SentryAssignedIssues != nil {
		updated.SentryAssignedIssues = requested.GetSentryAssignedIssues()
	}
	updated.SentryLevels = requested.GetSentryLevels()
	updated.DependabotSeverities = requested.GetDependabotSeverities()
	if requested.ExcludeKeyTasks != nil {
		updated.ExcludeKeyTasks = requested.GetExcludeKeyTasks()
	}
	updated.TaskListIDs = requested.GetTaskListIds()

	return updated.normalized()
}

func jiraCompletionSettingsFromMetadata(metadata map[string]any, settings intakeSettings) intakeSettings {
	if metadata == nil {
		return settings
	}
	if value, ok := metadata[intakeMetadataJiraMoveOnComplete]; ok {
		settings.JiraMoveOnComplete = metadataBool(value, settings.JiraMoveOnComplete)
	}
	if value, ok := metadata[intakeMetadataJiraCompletionColumn]; ok {
		if column, ok := value.(string); ok {
			settings.JiraCompletionColumn = strings.TrimSpace(column)
		}
	}
	return settings
}

func jiraCompletionMetadata(settings intakeSettings) map[string]any {
	return map[string]any{
		intakeMetadataJiraMoveOnComplete:   settings.JiraMoveOnComplete,
		intakeMetadataJiraCompletionColumn: settings.JiraCompletionColumn,
	}
}

func mergeJiraCompletionMetadata(metadata map[string]any, settings intakeSettings) map[string]any {
	if metadata == nil {
		metadata = map[string]any{}
	} else {
		metadata = maps.Clone(metadata)
	}
	for key, value := range jiraCompletionMetadata(settings) {
		metadata[key] = value
	}
	return metadata
}

func metadataBool(value any, fallback bool) bool {
	switch typed := value.(type) {
	case bool:
		return typed
	case string:
		switch strings.ToLower(strings.TrimSpace(typed)) {
		case "true":
			return true
		case "false":
			return false
		}
	}
	return fallback
}

func configurationStrings(value any) []string {
	switch values := value.(type) {
	case []string:
		return values
	case []any:
		result := make([]string, 0, len(values))
		for _, value := range values {
			if text, ok := value.(string); ok {
				result = append(result, text)
			}
		}
		return result
	default:
		return nil
	}
}

func serializeIntakeLabelFilterMode(mode string) pb.FactoryIntake_Settings_LabelFilterMode {
	if mode == intakeLabelFilterExclude {
		return pb.FactoryIntake_Settings_LABEL_FILTER_MODE_EXCLUDE
	}
	return pb.FactoryIntake_Settings_LABEL_FILTER_MODE_INCLUDE
}

func parseIntakeLabelFilterMode(mode pb.FactoryIntake_Settings_LabelFilterMode) string {
	if mode == pb.FactoryIntake_Settings_LABEL_FILTER_MODE_EXCLUDE {
		return intakeLabelFilterExclude
	}
	return intakeLabelFilterInclude
}

func serializeIntakeAssignment(assignment string) pb.FactoryIntake_Settings_Assignment {
	switch assignment {
	case intakeAssignmentAssigned:
		return pb.FactoryIntake_Settings_ASSIGNMENT_ASSIGNED
	case intakeAssignmentUnassigned:
		return pb.FactoryIntake_Settings_ASSIGNMENT_UNASSIGNED
	default:
		return pb.FactoryIntake_Settings_ASSIGNMENT_ANY
	}
}

func parseIntakeAssignment(assignment pb.FactoryIntake_Settings_Assignment) string {
	switch assignment {
	case pb.FactoryIntake_Settings_ASSIGNMENT_ASSIGNED:
		return intakeAssignmentAssigned
	case pb.FactoryIntake_Settings_ASSIGNMENT_UNASSIGNED:
		return intakeAssignmentUnassigned
	default:
		return intakeAssignmentAny
	}
}
