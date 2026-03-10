import { env } from './env';

import type { UserManagerSettings } from 'oidc-client-ts';

export const oidcConfig: UserManagerSettings = {
  authority: env.oidcAuthority,
  client_id: env.oidcClientId,
  redirect_uri: env.oidcRedirectUri,
  post_logout_redirect_uri: env.oidcPostLogoutRedirectUri,
  response_type: 'code',
  scope: 'openid profile email',
  automaticSilentRenew: true,
  userStore: undefined,
};
