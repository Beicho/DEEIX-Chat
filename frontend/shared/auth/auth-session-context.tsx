"use client";

import * as React from "react";

import { getMe } from "@/shared/api/auth";
import type { UserDTO } from "@/shared/api/auth.types";
import { USER_PROFILE_UPDATED_EVENT } from "@/shared/auth/user-profile-events";
import { reportFingerprintOnce } from "@/shared/security/fingerprint-reporter";

type AuthSessionUserStatus = "loading" | "ready" | "failed" | "error";

function isAuthError(error: unknown): boolean {
  const status =
    (error as { status?: number })?.status ?? (error as { response?: { status?: number } })?.response?.status;
  return status === 401 || status === 403;
}

type AuthSessionContextValue = {
  accessToken: string;
  user: UserDTO | null;
  userStatus: AuthSessionUserStatus;
  refreshUser: () => Promise<UserDTO | null>;
};

const AuthSessionContext = React.createContext<AuthSessionContextValue | null>(null);

export function AuthSessionProvider({
  accessToken,
  children,
}: {
  accessToken: string;
  children: React.ReactNode;
}) {
  const [user, setUser] = React.useState<UserDTO | null>(null);
  const [userStatus, setUserStatus] = React.useState<AuthSessionUserStatus>("loading");

  const refreshUser = React.useCallback(async () => {
    setUserStatus("loading");
    try {
      const nextUser = await getMe(accessToken);
      setUser(nextUser);
      setUserStatus("ready");
      return nextUser;
    } catch (error) {
      // UI-010: 仅在鉴权错误时清空用户态；网络/5xx 闪断保留现有用户。
      if (isAuthError(error)) {
        setUser(null);
        setUserStatus("failed");
      } else {
        setUserStatus("error");
      }
      return null;
    }
  }, [accessToken]);

  React.useEffect(() => {
    let cancelled = false;

    async function loadUser() {
      setUserStatus("loading");
      try {
        const nextUser = await getMe(accessToken);
        if (!cancelled) {
          setUser(nextUser);
          setUserStatus("ready");
        }
      } catch (error) {
        if (!cancelled) {
          // UI-010: 仅在鉴权错误时清空用户态；网络/5xx 闪断保留现有用户。
          if (isAuthError(error)) {
            setUser(null);
            setUserStatus("failed");
          } else {
            setUserStatus("error");
          }
        }
      }
    }

    void loadUser();
    return () => {
      cancelled = true;
    };
  }, [accessToken]);

  React.useEffect(() => {
    void reportFingerprintOnce(accessToken);
  }, [accessToken]);

  React.useEffect(() => {
    function handleProfileUpdated(event: Event) {
      const nextUser = (event as CustomEvent<UserDTO>).detail;
      if (!nextUser) {
        return;
      }

      setUser(nextUser);
      setUserStatus("ready");
    }

    window.addEventListener(USER_PROFILE_UPDATED_EVENT, handleProfileUpdated as EventListener);
    return () => {
      window.removeEventListener(USER_PROFILE_UPDATED_EVENT, handleProfileUpdated as EventListener);
    };
  }, []);

  const value = React.useMemo<AuthSessionContextValue>(
    () => ({
      accessToken,
      user,
      userStatus,
      refreshUser,
    }),
    [accessToken, refreshUser, user, userStatus],
  );

  return <AuthSessionContext.Provider value={value}>{children}</AuthSessionContext.Provider>;
}

export function useAuthSession() {
  const context = React.useContext(AuthSessionContext);
  if (!context) {
    throw new Error("useAuthSession must be used within AuthSessionProvider");
  }
  return context;
}

export function useOptionalAuthSession() {
  return React.useContext(AuthSessionContext);
}
