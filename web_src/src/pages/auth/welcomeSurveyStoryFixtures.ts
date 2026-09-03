import type { PostHogSurvey } from "./PostHogSurveyForm";

/**
 * Representative "New User Onboarding Survey" for Storybook.
 *
 * Question 1 matches the live PostHog poll. Question 3 uses the intended
 * replacement copy. Question 2 documents the multiple-choice poll; the live
 * middle question still comes from PostHog at runtime.
 */
export const onboardingSurvey: PostHogSurvey = {
  id: "new-user-onboarding",
  name: "New User Onboarding Survey",
  questions: [
    {
      id: "q-heard",
      question: "How did you hear about SuperPlane?",
      type: "single_choice",
      choices: [
        "GitHub",
        "Dev / AI-content (Blog, YouTube, Tutorial, AI-Dev video, Podcast)",
        "Social media (LinkedIn, Instagram, X, Discord)",
        "University / campus event (Student meetup, Hackathon at faculty, Promotion at uni)",
        "Hackathon / community event (Hackathon, Dev meetup, Online event)",
        "Partner / integration (GitHub, Daytona, Dash0, etc.)",
        "Referral (Friend, Teammate, Colleague)",
        "Other",
      ],
    },
    {
      id: "q-tools",
      question: "Which tools does your team use today?",
      type: "multiple_choice",
      allow_multiple: true,
      choices: [
        "GitHub (source control)",
        "GitHub Actions (CI/CD)",
        "PagerDuty (incidents)",
        "Terraform (infrastructure)",
      ],
    },
    {
      id: "q-agent-task",
      question:
        "If you had a single task for an AI agent on your software development process today, what would it be?",
      type: "open",
      placeholder: "Describe the task",
    },
  ],
};
