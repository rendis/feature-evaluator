export const env = {
  apiUrl: import.meta.env.VITE_API_URL ?? 'http://localhost:8080/features',
  authDisabled: import.meta.env.VITE_AUTH_DISABLED === 'true',
  oidcAuthority: import.meta.env.VITE_OIDC_AUTHORITY ?? '',
  oidcClientId: import.meta.env.VITE_OIDC_CLIENT_ID ?? '',
  oidcRedirectUri:
    import.meta.env.VITE_OIDC_REDIRECT_URI ?? `${window.location.origin}/auth/callback`,
  oidcPostLogoutRedirectUri:
    import.meta.env.VITE_OIDC_POST_LOGOUT_REDIRECT_URI ?? window.location.origin,
} as const;
