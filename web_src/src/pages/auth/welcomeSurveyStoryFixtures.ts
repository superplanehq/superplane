import type { PostHogSurvey } from "./PostHogSurveyForm";

/**
 * Representative "New User Onboarding Survey" for Storybook.
 *
 * Question 1 matches the live PostHog poll. Question 2 is the main-role
 * poll. Question 3 uses the intended replacement copy.
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
      id: "q-role",
      question: "What best describes your role?",
      type: "single_choice",
      choices: [
        "Software engineer (writes and ships code)",
        "Engineering manager (leads a team)",
        "Product manager (owns the roadmap)",
        "Other",
      ],
    },
    {
      id: "q-automate-first",
      question: "What engineering work would you automate first?",
      type: "open",
      placeholder: "Describe the task",
    },
  ],
};
