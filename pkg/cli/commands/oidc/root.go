package oidc

import (
	"github.com/spf13/cobra"
	"github.com/superplanehq/superplane/pkg/cli/core"
)

func NewCommand(options core.BindOptions) *cobra.Command {
	root := &cobra.Command{
		Use:   "oidc",
		Short: "Verify SuperPlane OIDC execution tokens",
	}

	verifyCmd := &cobra.Command{
		Use:   "verify",
		Short: "Verify a SuperPlane OIDC execution token",
		Args:  cobra.NoArgs,
	}

	var token string
	var apiURL string
	var orgID string
	var canvasID string
	var nodeID string
	var component string
	var projectID string
	var pipelineFile string
	var ref string
	var commitSha string

	verifyCmd.Flags().StringVar(&token, "token", "", "OIDC token to verify (default: SUPERPLANE_OIDC_TOKEN env var)")
	verifyCmd.Flags().StringVar(&apiURL, "url", "", "SuperPlane API URL (default: configured context URL)")
	verifyCmd.Flags().StringVar(&orgID, "org-id", "", "expected organization ID")
	verifyCmd.Flags().StringVar(&canvasID, "canvas-id", "", "expected canvas ID")
	verifyCmd.Flags().StringVar(&nodeID, "node-id", "", "expected node ID")
	verifyCmd.Flags().StringVar(&component, "component", "", "expected component name")
	verifyCmd.Flags().StringVar(&projectID, "project-id", "", "expected Semaphore project ID")
	verifyCmd.Flags().StringVar(&pipelineFile, "pipeline-file", "", "expected Semaphore pipeline file")
	verifyCmd.Flags().StringVar(&ref, "ref", "", "expected git ref")
	verifyCmd.Flags().StringVar(&commitSha, "commit-sha", "", "expected commit SHA")

	core.Bind(verifyCmd, &verifyCommand{
		token:        &token,
		apiURL:       &apiURL,
		orgID:        &orgID,
		canvasID:     &canvasID,
		nodeID:       &nodeID,
		component:    &component,
		projectID:    &projectID,
		pipelineFile: &pipelineFile,
		ref:          &ref,
		commitSha:    &commitSha,
	}, options)

	root.AddCommand(verifyCmd)

	return root
}
