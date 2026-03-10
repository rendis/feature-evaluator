import { createFileRoute } from '@tanstack/react-router';

import { AuthProfileBuilder } from '@/components/auth-profiles/auth-profile-builder';

export const Route = createFileRoute('/_authenticated/settings/auth-profiles/new')({
  component: AuthProfileCreatePage,
});

function AuthProfileCreatePage() {
  return <AuthProfileBuilder key="new" />;
}
