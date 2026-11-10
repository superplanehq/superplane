package models

import (
 "encoding/json"
 "gorm.io/datatypes"
 "gorm.io/gorm"
)

// LatestMergeConfidenceAssessment includes an unfinished newer run. A rerun
// immediately retires the old assessment, even when the head SHA is unchanged.
func (p *FactoryPullRequest) LatestMergeConfidenceAssessment(tx *gorm.DB, headSHA string) (*CanvasRun, datatypes.JSONMap, error) {
	runs, err := p.ListActivityRuns(tx)
	if err != nil {
		return nil, nil, err
	}
	for i := range runs {
		run := &runs[i]
		version, err := FindCanvasVersionInTransaction(tx, run.WorkflowID, run.VersionID)
		if err != nil {
			return nil, nil, err
		}
		if FactoryAppTemplateID(version.Nodes) != "risk-score" {
			continue
		}
		root, err := run.RootEvent(tx)
		if err != nil {
			return nil, nil, err
		}
		payload, ok := objectMap(RootEventSourcePayload(root.Data.Data()))
		if !ok {
			continue
		}
		pr, _ := objectMap(payload["pull_request"])
		head, _ := objectMap(pr["head"])
		if head["sha"] != headSHA {
			continue
		}
		report, err := run.MergeConfidenceAssessment(tx)
		if err != nil {
			return nil, nil, err
		}
		return run, report, nil
	}
	return nil, nil, nil
}

// MergeConfidenceAssessment reads only the server-owned assessment node from
// this run's frozen version and validates the execution that emitted it.
func (r *CanvasRun) MergeConfidenceAssessment(tx *gorm.DB) (datatypes.JSONMap, error) {
	version, err := FindCanvasVersionInTransaction(tx, r.WorkflowID, r.VersionID)
	if err != nil {
		return nil, err
	}
	if FactoryAppTemplateID(version.Nodes) != "risk-score" {
		return nil, nil
	}
	assessmentNode := mergeConfidenceAssessmentNode(version.Nodes)
	if assessmentNode == nil {
		return nil, nil
	}
	var events []CanvasEvent
	if err := tx.Where("run_id = ? AND execution_id IS NOT NULL", r.ID).
		Order("created_at ASC, id ASC").Find(&events).Error; err != nil {
		return nil, err
	}
	for _, event := range events {
		if event.NodeID != assessmentNode.ID || (event.Channel != "passed" && event.Channel != "failed") {
			continue
		}
		execution, err := FindNodeExecutionInTransaction(tx, r.WorkflowID, *event.ExecutionID)
		if err != nil {
			return nil, err
		}
		if execution.RunID != r.ID || execution.NodeID != assessmentNode.ID || (execution.FrozenComponentName() != "" && execution.FrozenComponentName() != assessmentNode.ComponentName()) {
			continue
		}
		envelope, ok := objectMap(event.Data.Data())
		if !ok {
			continue
		}
		if envelope["type"] != assessmentNode.ComponentName()+".finished" {
			continue
		}
		body, _ := objectMap(envelope["data"])
		if report, ok := objectMap(body["mergeConfidence"]); ok {
			return datatypes.JSONMap(report), nil
		}
	}
	return nil, nil
}

func mergeConfidenceAssessmentNode(nodes []Node) *Node {
	for i := range nodes {
		node := &nodes[i]
		if node.ID != "assess-risk" || !IsAgentRunnerComponent(node.ComponentName()) {
			continue
		}
		encoded, err := json.Marshal(node.Configuration)
		if err != nil {
			continue
		}
		var configuration struct {
			Environment []struct {
				Name string `json:"name"`
			} `json:"environment"`
		}
		if json.Unmarshal(encoded, &configuration) != nil {
			continue
		}
		for _, variable := range configuration.Environment {
			// This is the server-owned assessment scope, not an arbitrary agent result.
			if variable.Name == "SUPERPLANE_MERGE_CONFIDENCE_ORDER_ID" {
				return node
			}
		}
	}
	return nil
}

