import * as React from "react";
import { useQuery, useQueryClient } from "@tanstack/react-query";

import { api, type AuthStatus, type User } from "@/lib/api";

type AuthContextValue = {
  user: User | null;
  needsSetup: boolean;
  isLoading: boolean;
  isAuthenticated: boolean;
  isAdmin: boolean;
  /** Re-fetch auth status (call after login/logout/setup). */
  refresh: () => Promise<void>;
};

const AuthContext = React.createContext<AuthContextValue | null>(null);

const AUTH_KEY = ["auth"] as const;

export function AuthProvider({ children }: { children: React.ReactNode }) {
  const qc = useQueryClient();

  const { data, isLoading } = useQuery<AuthStatus>({
    queryKey: AUTH_KEY,
    queryFn: api.authStatus,
    staleTime: 60_000,
  });

  const refresh = React.useCallback(async () => {
    await qc.invalidateQueries({ queryKey: AUTH_KEY });
  }, [qc]);

  const user = data?.user ?? null;
  const value: AuthContextValue = {
    user,
    needsSetup: data?.needs_setup ?? false,
    isLoading,
    isAuthenticated: Boolean(data?.authenticated),
    isAdmin: user?.role === "admin",
    refresh,
  };

  return <AuthContext.Provider value={value}>{children}</AuthContext.Provider>;
}

// eslint-disable-next-line react-refresh/only-export-components
export function useAuth(): AuthContextValue {
  const ctx = React.useContext(AuthContext);
  if (!ctx) {
    throw new Error("useAuth must be used within an AuthProvider");
  }
  return ctx;
}
