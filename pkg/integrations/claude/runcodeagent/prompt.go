package runcodeagent

import (
	"fmt"
	"strings"
)

// buildPrompt wraps the user's task in a deterministic operating procedure that
// makes the agent clone, work, push, and open (or update) a pull request, then
// report the PR URL in a machine-readable form.
func buildPrompt(spec Spec, pr *pullRequestInfo, branch string, hasFiles bool, attr commitAttribution) string {
	if pr != nil {
		return buildPRPrompt(spec, pr, hasFiles, attr)
	}
	return buildRepositoryPrompt(spec, branch, hasFiles, attr)
}

func buildRepositoryPrompt(spec Spec, branch string, hasFiles bool, attr commitAttribution) string {
	var b strings.Builder
	writeIntro(&b, hasFiles)

	fmt.Fprintf(&b, "1. Clone the repository and enter it:\n")
	fmt.Fprintf(&b, "     git clone %s repo && cd repo\n", authenticatedCloneURL(spec.Repository))
	if base := strings.TrimSpace(spec.BaseBranch); base != "" {
		fmt.Fprintf(&b, "     git checkout %s\n", base)
	}
	writeIdentity(&b, attr)
	fmt.Fprintf(&b, "2. Create and switch to a working branch: git checkout -b %s\n", branch)
	fmt.Fprintf(&b, "3. Complete this task:\n%s\n", indent(spec.Task))
	fmt.Fprintf(&b, "4. %s and push the branch: git push -u origin %s\n", commitInstruction(attr), branch)
	if prEnabled(spec) {
		target := strings.TrimSpace(spec.BaseBranch)
		if target == "" {
			target = "the default branch"
		}
		fmt.Fprintf(&b, "5. Open a pull request from %s into %s (use the GitHub API with $GITHUB_TOKEN, or the gh CLI).\n", branch, target)
	}
	writeFinalMarker(&b, prEnabled(spec))
	return b.String()
}

func buildPRPrompt(spec Spec, pr *pullRequestInfo, hasFiles bool, attr commitAttribution) string {
	var b strings.Builder
	writeIntro(&b, hasFiles)
	fmt.Fprintf(&b, "You are updating the existing pull request %s.\n\n", pr.HTMLURL)

	fmt.Fprintf(&b, "1. Clone the repository and switch to the pull request's branch (it already exists on origin):\n")
	fmt.Fprintf(&b, "     git clone %s repo && cd repo\n", authenticatedCloneURL(pr.BaseRepo))
	fmt.Fprintf(&b, "     git switch %s\n", pr.HeadRef)
	writeIdentity(&b, attr)
	fmt.Fprintf(&b, "2. Complete this task:\n%s\n", indent(spec.Task))
	fmt.Fprintf(&b, "3. %s and push to the SAME branch (%s) — do NOT open a new pull request; pushing updates the existing one: git push origin %s\n", commitInstruction(attr), pr.HeadRef, pr.HeadRef)
	writeFinalMarker(&b, true)
	fmt.Fprintf(&b, "\nThe pull request URL is %s.\n", pr.HTMLURL)
	return b.String()
}

// writeIdentity configures git to attribute commits to the resolved user.
func writeIdentity(b *strings.Builder, attr commitAttribution) {
	if attr.enabled() {
		fmt.Fprintf(b, "     git config user.name %q && git config user.email %q\n", attr.AuthorName, attr.AuthorEmail)
	}
}

// commitInstruction returns the commit step wording, suppressing agent
// attribution trailers when commits should appear as the user.
func commitInstruction(attr commitAttribution) string {
	if attr.enabled() {
		return "Commit your changes with a clear message, without any \"Co-Authored-By\" or \"Generated with\" trailers"
	}
	return "Commit your changes with a clear message"
}

func writeIntro(b *strings.Builder, hasFiles bool) {
	b.WriteString("You are an autonomous coding agent in a fresh Linux sandbox with network access. ")
	b.WriteString("A GitHub token is available in the $GITHUB_TOKEN environment variable.\n")
	if hasFiles {
		fmt.Fprintf(b, "Attached files are available under %s.\n", attachmentsMountDir)
	}
	b.WriteString("\nFollow these steps:\n")
}

func writeFinalMarker(b *strings.Builder, prExpected bool) {
	b.WriteString("\nWhen finished, output on the FINAL line of your last message exactly one of:\n")
	if prExpected {
		b.WriteString("     PR_URL=<the pull request url>\n")
	}
	b.WriteString("     NO_PR   (if there were no changes to submit)\n")
}

func authenticatedCloneURL(repository string) string {
	repository = strings.TrimSpace(repository)
	if repoOwnerRepoPattern.MatchString(repository) {
		return fmt.Sprintf("https://x-access-token:$GITHUB_TOKEN@github.com/%s.git", repository)
	}
	if strings.HasPrefix(repository, "https://") {
		return "https://x-access-token:$GITHUB_TOKEN@" + strings.TrimPrefix(repository, "https://")
	}
	return repository
}

func prEnabled(spec Spec) bool {
	return spec.AutoCreatePr == nil || *spec.AutoCreatePr
}

func indent(text string) string {
	lines := strings.Split(strings.TrimRight(text, "\n"), "\n")
	for i, l := range lines {
		lines[i] = "   " + l
	}
	return strings.Join(lines, "\n")
}
