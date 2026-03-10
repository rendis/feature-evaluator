import { UserManager, type User } from 'oidc-client-ts';
import { useCallback, useEffect, useMemo, useState, type ReactNode } from 'react';

import type { Member } from '@/api/types';

import { ApiClientError, setTokenProvider } from '@/api/client';
import { membersApi } from '@/api/members';
import { AuthContext } from '@/auth/auth-context';
import { env } from '@/config/env';
import { oidcConfig } from '@/config/oidc';
import { useWorkspaceStore } from '@/stores/workspace-store';

const mockMember: Member = {
  id: 'dev-user',
  email: 'dev@local.dev',
  role: 'owner',
  displayName: 'Dev User',
  addedBy: 'system',
  createdAt: new Date().toISOString(),
  updatedAt: new Date().toISOString(),
};

function useOidcSession() {
  const [user, setUser] = useState<User | null>(null);
  const [isLoading, setIsLoading] = useState(!env.authDisabled);
  const [userManager, setUserManager] = useState<UserManager | null>(null);

  useEffect(() => {
    if (env.authDisabled) {
      setTokenProvider(() => 'dev-token');
      return;
    }

    const mgr = new UserManager(oidcConfig);
    setUserManager(mgr);

    const onLoaded = (nextUser: User) => {
      setUser(nextUser);
      setTokenProvider(() => nextUser.access_token);
    };
    const onUnloaded = () => {
      setUser(null);
      setTokenProvider(() => null);
    };

    const initAuth = async () => {
      try {
        const nextUser = await mgr.getUser();
        if (nextUser && !nextUser.expired) {
          setUser(nextUser);
          setTokenProvider(() => nextUser.access_token);
        }
      } catch {
        // User is not signed in yet.
      } finally {
        setIsLoading(false);
      }
    };

    mgr.events.addUserLoaded(onLoaded);
    mgr.events.addUserUnloaded(onUnloaded);
    void initAuth();

    return () => {
      mgr.events.removeUserLoaded(onLoaded);
      mgr.events.removeUserUnloaded(onUnloaded);
      setTokenProvider(() => null);
      setUserManager(null);
    };
  }, []);

  return { isLoading, user, userManager };
}

function useWorkspaceMember(user: User | null, workspaceKey: string) {
  const [member, setMember] = useState<Member | null>(env.authDisabled ? mockMember : null);
  const [isMemberLoading, setIsMemberLoading] = useState(false);

  useEffect(() => {
    if (env.authDisabled) {
      setMember(mockMember);
      setIsMemberLoading(false);
      return;
    }

    if (!user || user.expired || !workspaceKey) {
      setMember(null);
      setIsMemberLoading(false);
      return;
    }

    let cancelled = false;
    setIsMemberLoading(true);

    const ignoredErrors = new Set([
      'error.memberNotFound',
      'error.accessDenied',
      'error.workspaceRequired',
      'error.workspaceArchived',
      'error.workspaceNotFound',
    ]);

    const loadMember = async () => {
      try {
        const nextMember = await membersApi.me();
        if (!cancelled) {
          setMember(nextMember);
        }
      } catch (error) {
        if (!cancelled) {
          const shouldClearMember =
            !(error instanceof ApiClientError) || ignoredErrors.has(error.messageKey);
          if (shouldClearMember) {
            setMember(null);
          }
        }
      } finally {
        if (!cancelled) {
          setIsMemberLoading(false);
        }
      }
    };

    void loadMember();

    return () => {
      cancelled = true;
    };
  }, [user, workspaceKey]);

  return { isMemberLoading, member };
}

export function AuthProvider({ children }: { children: ReactNode }) {
  const workspaceKey = useWorkspaceStore((state) => state.workspaceKey);
  const { isLoading, user, userManager } = useOidcSession();
  const { isMemberLoading, member } = useWorkspaceMember(user, workspaceKey);

  const login = useCallback(async () => {
    await userManager?.signinRedirect();
  }, [userManager]);

  const logout = useCallback(async () => {
    await userManager?.signoutRedirect();
  }, [userManager]);

  const value = useMemo(
    () => ({
      isAuthenticated: env.authDisabled || (!!user && !user.expired),
      isLoading,
      isMemberLoading,
      user,
      member,
      login,
      logout,
      userManager,
    }),
    [user, member, isLoading, isMemberLoading, login, logout, userManager],
  );

  return <AuthContext.Provider value={value}>{children}</AuthContext.Provider>;
}
