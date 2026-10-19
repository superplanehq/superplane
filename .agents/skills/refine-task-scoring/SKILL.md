---
name: refine-task-scoring
description: Rubric for scoring a Backlog draft task on Clarity, Complexity, and Verifiability (1-3), how to raise scores, and how to write score summaries. Use only when refining a draft task.
---

# Refine task scoring

Use this skill when you score a draft task during Backlog refinement.

## 3. Score Clarity

Clarity is how well the task is defined: the outcome, the scope, and what done means are decided, so the plan names real behavior. It is not about whether an agent can do the work. That is Complexity. Keep refining until every score is 3.

- 3: outcome, scope, and done are decided. The plan names real behavior with no open choice.
- 2: the outcome is clear, but a choice is still open: the scope, the definition of done, or a decision that changes the plan.
- 1: the task does not say what should change.

If Clarity is 1, do not write a specification. Say you do not have enough Clarity to write a plan. Invite the user to refine. Publish the scores alone.

If Clarity is 2, write the plan. Say it is a first pass. Invite the user to refine.

If Clarity is 3, write the specification.

## 4. Score Complexity

Complexity is whether a coding agent finishes this task in one run, with no steering, once the plan is clear. A clear task can still be a poor fit.

Assume a capable agent that has the full repository, the plan you write, and the tests. It reads code well, follows an existing pattern well, and runs commands to check its work. It struggles when it must guess at taste, product intent, or context that is not written down.

Judge from what you found in the repository, not from the task text alone:
- Size: whether the work fits one run. Count the seams the agent must touch, not the lines of the description.
- Pattern: a similar change already exists to copy.
- Judgment: taste, product calls, or context that is not written down.
- Blast radius: migrations, auth, billing, production data, or work that is hard to undo.

### Calibration

Start at 3 for a bounded change that has a pattern in the repository. Move down only for a concrete reason you found in the code, and name that reason in the summary.

- A judgment call that the plan already decides does not lower the score. Only open judgment does.
- Touching two or three files in one area is normal. Only crossing areas that need different context lowers the score.
- A long task description is not a large task. A short one is not a small task. Score the work, not the text.

### Anchors

- 3: fits one run. A pattern exists or the change is contained.
- 2: fits one run, but expect steering: the change crosses areas with different context, or one judgment call is open.
- 1: does not fit one run, or depends on unwritten context, or touches irreversible or security-sensitive work. Name a narrower scope for this task, or a human step.

Be direct when the task is too big for one run. Say so in the Complexity summary. Then name the narrower scope for this task. Do not soften a 1 to spare the user; a wrong 3 costs them a failed run.

Keep this task as one task. Do not create another task. Do not tell the user to split the work. If the work is too large, name the narrower scope for this task and ask the user to confirm it. Keep the original scope in the plan until the user confirms.

## 5. Score Verifiability

Verifiability is whether the run can prove the change works.

- 3: a test, a build, or another check the agent can run covers the change.
- 2: a check the agent can run covers only part of the change.
- 1: only a person can judge the result.

Missing tests alone drop Verifiability to 2 at most when the change is small and easy to check by hand.

### Raise the scores through refinement

The scores are not fixed. When you see a way to raise one, propose it. Most tasks get more agent-friendly when the scope narrows or a missing fact gets written down. Use chat or a survey question for this.

Moves that raise a score:
- Drop a part that adds risk but not value to this task. Leave that part out of this plan. Do not make it a new task.
- Narrow the scope so the remaining work fits one run.
- Add a failing test first so done is provable.
- Turn an open judgment call into a decision in the plan.
- Put the risky step behind a human review, and keep the rest for the agent.

### Score summaries

Each summary is one sentence of 14 words or fewer. Name the one decisive fact. Do not chain a list of parts or steps. Write the summary about the task, not to the user. Do not repeat the number; the UI shows it next to the text. Do not write a test or an acceptance check in a summary.

The Clarity summary names the missing fact or decision, or says the task is decided. Do not describe agent fit there. The Complexity summary names the size problem, not missing decisions and not proof. The Verifiability summary names the proof, or the missing proof.

In chat, name only the weakest sub-parameter and what raises it. Do not recite all three scores.
