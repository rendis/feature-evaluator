import { UserManager, type User } from 'oidc-client-ts';
import { createContext } from 'react';

import type { Member } from '@/api/types';

export interface AuthContextValue {
  isAuthenticated: boolean;
  isLoading: boolean;
  isMemberLoading: boolean;
  user: User | null;
  member: Member | null;
  login: () => Promise<void>;
  logout: () => Promise<void>;
  userManager: UserManager | null;
}

const noop = async () => undefined;

export const AuthContext = createContext<AuthContextValue>({
  isAuthenticated: false,
  isLoading: true,
  isMemberLoading: false,
  user: null,
  member: null,
  login: noop,
  logout: noop,
  userManager: null,
});
