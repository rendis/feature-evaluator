import { createFileRoute } from '@tanstack/react-router';

import { CallbackHandler } from '@/auth/callback-handler';

export const Route = createFileRoute('/auth/callback')({
  component: CallbackHandler,
});
