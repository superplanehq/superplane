# Semaphore RFCs (Requests for Comments)

## Introduction

This document outlines the RFC (Request for Comments) process for the Semaphore updates. It is a way for the Semaphore team to include the community and users in the discussion and design of future improvements.

RFCs are design documents used to propose, describe, and discuss significant changes to Semaphore. The RFC process provides a structured way to gather input from the community and users before implementing major features or changes.

## When should an RFC be created?

An RFC should be created for any substantial change to Semaphore that would benefit from broader discussion and consensus. This includes, but is not limited to:

- New features with significant impact on users or the architecture
- Major changes to existing features or APIs
- Introduction of new integrations with the Semaphore

Not every change will be covered by an RFC. RFCs serve to efficiently gather improvement ideas from the community and validate existing proposals through community feedback.

## How to create an RFC

1. Fork the Semaphore repository.

2. Create a new branch named `rfc/short-descriptive-title`. For example: `rfc/new-user-api`.

3. Create a new subfolder in the `rfcs` directory with the format `0000-your-feature`. For example: `rfcs/0000-new-user-api/`.

4. Copy the RFC template from `rfcs/0000-example/rfc.md` to your new subfolder, naming it `rfc.md`.

5. Fill in the RFC template with your proposal. Be as detailed as possible, including motivation, design details, alternatives considered, and potential drawbacks.

6. Submit a pull request to the Semaphore repository with your RFC.

7. Update the front matter of your RFC:
   - Assign yourself as the author
   - Set the status to "Draft"
   - Leave the RFC number as 0000 (it will be assigned later)

8. The pull request description should include a brief summary of the RFC and any additional context that might be helpful for reviewers.

## How are RFCs reviewed?

1. Once submitted, Semaphore team will assign a number to the RFC and update the folder name accordingly.

2. The community and Semaphore team will review the RFC, asking questions and providing feedback through pull request comments.

3. You should be prepared to revise your RFC based on the feedback received. Push your changes to the same branch to update the pull request.

4. RFCs must remain in the "Draft" status for a minimum of **two weeks** to allow sufficient time for review and discussion.

5. After the review period, Semaphore team will decide to either:
   - Accept the RFC and merge the pull request
   - Request further changes
   - Decline the RFC and close the pull request

6. Accepted RFCs will have their status updated to "Accepted" before merging and a number assigned.

## Implementation

Once an RFC is accepted:

1. Create a new issue in the Semaphore repository to track the implementation progress, linking back to the RFC.

2. The issue is to be used only to communicate the progress to community and users. 

3. Once the implementation is complete and merged, Semaphore team will update the RFC's status to "Implemented" and move it to the `archive` folder.

## End of life

As Semaphore evolves, some RFCs may become obsolete or be superseded by newer proposals. In such cases:

1. Create a pull request to update the RFC's status to "Deprecated" and move it to the `archive` folder. 

2. In the PR, provide a brief explanation of why the RFC is being deprecated and, if applicable, link to any new RFCs or documentation that replace it.

3. Once approved and merged, the RFC will be clearly marked as deprecated but will remain in the repository for historical context.

Remember, the RFC process is meant to be collaborative and iterative. Don't hesitate to ask for help or clarification throughout the process. Your contributions help shape the future of Semaphore!