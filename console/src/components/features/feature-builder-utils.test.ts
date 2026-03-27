import {
  buildCreatePayload,
  buildUpdatePayload,
  createDraft,
  serializeSnapshot,
} from './feature-builder-utils';

import type { Feature } from '@/api/types';

describe('feature-builder-utils cache fields', () => {
  const feature: Feature = {
    id: 'feature-1',
    key: 'feature-a',
    name: 'Feature A',
    description: 'Demo',
    enabled: true,
    evalCacheEnabled: true,
    evalCacheTTLSeconds: 180,
    valueType: 'boolean',
    defaultValue: true,
    metadata: {},
    tags: [],
    inputContract: { headers: [] },
    createdAt: '',
    updatedAt: '',
    createdBy: '',
    updatedBy: '',
  };

  it('hydrates cache state from the feature and serializes it into snapshots', () => {
    const draft = createDraft(feature);

    expect(draft.evalCacheEnabled).toBe(true);
    expect(draft.evalCacheTTLSeconds).toBe('180');
    expect(serializeSnapshot(draft)).toContain('"evalCacheEnabled":true');
    expect(serializeSnapshot(draft)).toContain('"evalCacheTTLSeconds":"180"');
  });

  it('includes cache controls in create and update payloads', () => {
    const draft = createDraft(feature);
    const createPayload = buildCreatePayload(draft);
    const updatePayload = buildUpdatePayload(draft);

    expect(createPayload.evalCacheEnabled).toBe(true);
    expect(createPayload.evalCacheTTLSeconds).toBe(180);
    expect(updatePayload.evalCacheEnabled).toBe(true);
    expect(updatePayload.evalCacheTTLSeconds).toBe(180);
  });
});
