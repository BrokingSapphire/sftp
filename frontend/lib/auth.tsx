"use client";

import { createContext, useContext, useEffect, useState, type ReactNode } from "react";
import { useRouter } from "next/navigation";
import { authApi } from "./endpoints";
import { tokens } from "./api";
import type { UserInfo } from "./types";

interface AuthState {
  user: UserInfo | null;
  loading: boolean;
  login: (identifier: string, password: string, remember: boolean) => Promise<void>;
  logout: () => Promise<void>;
  has: (perm: string) => boolean;
  refreshUser: () => Promise<void>;
}

const AuthContext = createContext<AuthState | null>(null);

export function AuthProvider({ children }: { children: ReactNode }) {
  const [user, setUser] = useState<UserInfo | null>(null);
  const [loading, setLoading] = useState(true);
  const router = useRouter();

  async function loadUser() {
    if (!tokens.access()) {
      setUser(null);
      setLoading(false);
      return;
    }
    try {
      setUser(await authApi.me());
    } catch {
      setUser(null);
    } finally {
      setLoading(false);
    }
  }

  useEffect(() => {
    loadUser();
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  const value: AuthState = {
    user,
    loading,
    async login(identifier, password, remember) {
      const pair = await authApi.login(identifier, password, remember);
      setUser(pair.user ?? (await authApi.me()));
      router.push("/dashboard");
    },
    async logout() {
      await authApi.logout();
      setUser(null);
      router.push("/login");
    },
    has: (perm) => !!user?.permissions?.some((p) => p === perm || p === "admin.all"),
    refreshUser: loadUser,
  };

  return <AuthContext.Provider value={value}>{children}</AuthContext.Provider>;
}

export function useAuth() {
  const ctx = useContext(AuthContext);
  if (!ctx) throw new Error("useAuth must be used within AuthProvider");
  return ctx;
}
