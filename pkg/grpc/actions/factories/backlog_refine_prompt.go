package factories

import (
	"crypto/sha256"
	"encoding/hex"
	"strings"

	"github.com/superplanehq/superplane/pkg/models"
)

const backlogRefineStepName = "Refine Task"

// defaultRefinePromptDigests are SHA-256 digests of every default Refine Task
// prompt SuperPlane has seeded into a Backlog canvas, the current one
// included. A prompt that still matches one of them was never edited, so the
// upgrade may replace it with the current default. When
// analysis_user_prompt.md changes, add the digest of the new default here;
// Test__DefaultRefinePromptDigestIsListed fails until the list is current.
var defaultRefinePromptDigests = map[string]struct{}{
	// analysis_protocol.md was the node prompt before the protocol split.
	"259cebbeb0af3a4a679bfa0d7b6e95fbca34e0011ff460cb7bca9e202d4edb2d": {},
	"3a7aa4063d1447bdacf8ba66cb4c27e709e27ec030358f13e27690cf9e53540e": {},
	"8d58254967ddb0b94c8dd7704f466cd188c99b57092c5a9639bbf2a07352200a": {},
	"dd34dc842a301998b3c2160a47dbcb8f343699603133b1061f0f5ce939adcd75": {},
	// analysis_user_prompt.md: Clarity only, then Clarity and Confidence.
	"9529d67d85a62b5484169aadc85935f91c2aad130cf8d09c2c6d6b84df3f94c3": {},
	"3890f3859be319457357d36d38decdad5e5591a069dfae59bac1a547aadb3e28": {},
	"3ced312559cb5a5b07b8df0be27242940be17aae0bb1e9b89df9f085d0876d33": {},
	// Calibrated Confidence; refinement may raise Confidence.
	"50eae06fb6d645a9ca446d234920f780fbd579975a5edfd96fb6da6be0348aed": {},
	// Split the task: propose a split in a survey, create the parts on confirm.
	"fe98c9788445b9c1444cccd44762dbd30ceecae6ef0185855e879d646017c51b": {},
	// A part of a split treats the boundary as decided; no contractions.
	"89357bdc369d0ee8a7bb9173408c155338b0e9db6fc9a88a0a0345fe420b0151": {},
}

func refinePromptDigest(prompt string) string {
	sum := sha256.Sum256([]byte(strings.TrimSpace(prompt)))
	return hex.EncodeToString(sum[:])
}

// isDefaultRefinePrompt reports whether a Refine Task prompt is one SuperPlane
// generated, so replacing it loses no user edits.
func isDefaultRefinePrompt(prompt string) bool {
	_, ok := defaultRefinePromptDigests[refinePromptDigest(prompt)]
	return ok
}

// refreshBacklogRefinePrompt replaces a stale default Refine Task prompt with
// the current default. It reports whether a node changed. A prompt the user
// edited stays as it is.
func refreshBacklogRefinePrompt(nodes []models.Node) bool {
	for i := range nodes {
		if nodes[i].ID != backlogRefinementNodeID {
			continue
		}
		step := backlogRefineStep(nodes[i].Configuration)
		if step == nil {
			return false
		}
		prompt, _ := step["prompt"].(string)
		current := intakeRefinementPrompt()
		if strings.TrimSpace(prompt) == current || !isDefaultRefinePrompt(prompt) {
			return false
		}
		step["prompt"] = current
		return true
	}
	return false
}

func backlogRefineStep(configuration map[string]any) map[string]any {
	steps, _ := configuration["steps"].([]any)
	for _, entry := range steps {
		step, _ := entry.(map[string]any)
		if step["name"] == backlogRefineStepName && step["type"] == "prompt" {
			return step
		}
	}
	return nil
}
