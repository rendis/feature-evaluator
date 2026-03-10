import { useAuth } from './use-auth';

/** Return the current member. Throws if not authenticated. */
export function useCurrentUser() {
  const { member } = useAuth();
  if (!member) throw new Error('useCurrentUser must be used within an authenticated context');
  return member;
}
