import { useContext } from 'react';

import { AuthContext } from '@/auth/auth-context';

/** Access the current authentication state and user info. */
export function useAuth() {
  return useContext(AuthContext);
}
