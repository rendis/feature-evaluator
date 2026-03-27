import { buildPayload, createDraft, serializeSnapshot } from './auth-profile-builder-utils';

import type { AuthProfile } from '@/api/types';

describe('auth-profile-builder-utils cache fields', () => {
  const profile: AuthProfile = {
    id: 'profile-1',
    key: 'custom-profile',
    name: 'Custom profile',
    active: true,
    type: 'custom',
    cacheEnabled: true,
    cacheTTLSeconds: 240,
    config: {
      url: 'https://example.com',
      method: 'POST',
      timeout: 5000,
      outboundAuthHeaderName: 'Authorization',
      headers: [
        {
          source: { type: 'request_header', name: 'x-user-id' },
          target: { type: 'header', name: 'x-user-id' },
        },
      ],
      body: [],
      requestHeaders: [{ key: 'x-auth-mode', value: 'jwt' }],
      successRule: {
        type: 'any_2xx',
      },
    },
    version: 1,
    hasSecret: false,
    createdAt: '',
    updatedAt: '',
    createdBy: '',
    updatedBy: '',
  };

  it('hydrates and serializes explicit cache settings', () => {
    const draft = createDraft(profile);

    expect(draft.cacheEnabled).toBe(true);
    expect(draft.cacheTTLSeconds).toBe('240');
    expect(serializeSnapshot(draft)).toContain('"cacheEnabled":true');
  });

  it('builds payloads with explicit cache flags', () => {
    const payload = buildPayload(createDraft(profile), profile.key);

    expect(payload.cacheEnabled).toBe(true);
    expect(payload.cacheTTLSeconds).toBe(240);
  });
});
