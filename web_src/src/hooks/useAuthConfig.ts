import { useQuery } from "@tanstack/react-query";

type AuthConfig = {
  providers: string[];
  passwordLoginEnabled: boolean;
  signupEnabled: boolean;
  publicAppBaseUrl: string;
};

const fetchAuthConfig = async (): Promise<AuthConfig> => {
  const res = await fetch("/auth/config");
  if (!res.ok) throw new Error("Failed to load auth config");
  return res.json() as Promise<AuthConfig>;
};

export const useAuthConfig = () => {
  return useQuery({
    queryKey: ["authConfig"],
    queryFn: fetchAuthConfig,
    staleTime: 60 * 60 * 1000, // 1 hour — config rarely changes
    gcTime: 60 * 60 * 1000,
  });
};

export const usePublicBaseURL = (): string => {
  const { data } = useAuthConfig();
  return data?.publicAppBaseUrl || window.location.origin;
};
