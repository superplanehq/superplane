const isValidRedirectPath = (path: string | null): path is string => {
  if (!path || path[0] !== "/") {
    return false;
  }

  if (path.length > 1 && path[1] === "/") {
    return false;
  }

  return !path.startsWith("/welcome");
};

export const getWelcomeSurveyRedirectPath = (rawRedirect: string | null): string => {
  if (!rawRedirect) {
    return "/";
  }

  try {
    const decoded = decodeURIComponent(rawRedirect);
    return isValidRedirectPath(decoded) ? decoded : "/";
  } catch {
    return "/";
  }
};
