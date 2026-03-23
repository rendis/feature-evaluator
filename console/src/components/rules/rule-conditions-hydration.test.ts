import { hydrateExternalApiConditions } from './rule-conditions-hydration';

import type { BuilderGroup } from './conditions-builder-types';
import type { ExternalApi, ExternalApiBinding } from '@/api/types';

describe('rule conditions hydration', () => {
  const baseRoot = (): BuilderGroup => ({
    id: 'root',
    kind: 'group',
    connector: 'and',
    items: [
      {
        id: 'api-condition',
        kind: 'condition',
        conditionKind: 'externalApi',
        externalApiKey: 'payment-validator',
        externalApiName: '',
        paramMappings: [],
        negate: false,
      },
    ],
  });

  const inputFields = [
    {
      category: 'headers' as const,
      path: 'headers.userId',
      label: 'User ID',
      type: 'string' as const,
    },
    {
      category: 'derived' as const,
      path: 'derived.workspaceId',
      label: 'Workspace ID',
      type: 'string' as const,
    },
  ];

  it('rehydrates external api mappings from saved bindings', () => {
    const root = baseRoot();
    const externalApiBindings: ExternalApiBinding[] = [
      {
        externalApiKey: 'payment-validator',
        paramMappings: [
          {
            paramName: 'userId',
            mode: 'input',
            inputPath: 'headers.userId',
            literalValue: '',
          },
          {
            paramName: 'workspaceId',
            mode: 'literal',
            literalValue: 'acme',
          },
        ],
        failMode: 'open',
        cacheTTL: 0,
      },
    ];
    const externalApis: ExternalApi[] = [
      {
        id: 'api-1',
        key: 'payment-validator',
        name: 'Payment Validator',
        active: true,
        request: { method: 'POST', urlTemplate: 'https://example.com', headers: [] },
        params: [
          { name: 'userId', type: 'string', required: true, locations: ['body'] },
          { name: 'workspaceId', type: 'string', required: false, locations: ['body'] },
        ],
        responseValidation: {
          mode: 'httpCode',
          http: { mode: 'any_2xx' },
          body: { expression: '' },
        },
        hasSecrets: false,
        version: 1,
        createdAt: '',
        updatedAt: '',
        createdBy: '',
        updatedBy: '',
      },
    ];

    const hydrated = hydrateExternalApiConditions(root, {
      externalApiBindings,
      externalApis,
      inputFields,
    });

    const condition = hydrated.items[0];
    if (condition.kind !== 'condition' || condition.conditionKind !== 'externalApi') {
      throw new Error('expected external api condition');
    }

    expect(condition.externalApiName).toBe('Payment Validator');
    expect(condition.paramMappings).toEqual([
      {
        paramName: 'userId',
        paramType: 'string',
        required: true,
        mode: 'input',
        inputRef: {
          refKind: 'input',
          category: 'headers',
          path: 'headers.userId',
          label: 'User ID',
          type: 'string',
        },
        literalValue: '',
      },
      {
        paramName: 'workspaceId',
        paramType: 'string',
        required: false,
        mode: 'literal',
        inputRef: null,
        literalValue: 'acme',
      },
    ]);
  });

  it('merges current api params with legacy saved params', () => {
    const root = baseRoot();
    const externalApiBindings: ExternalApiBinding[] = [
      {
        externalApiKey: 'payment-validator',
        paramMappings: [
          { paramName: 'userId', mode: 'input', inputPath: 'headers.userId' },
          { paramName: 'legacyParam', mode: 'literal', literalValue: 'legacy' },
        ],
        failMode: 'open',
        cacheTTL: 0,
      },
    ];
    const externalApis: ExternalApi[] = [
      {
        id: 'api-1',
        key: 'payment-validator',
        name: 'Payment Validator',
        active: true,
        request: { method: 'POST', urlTemplate: 'https://example.com', headers: [] },
        params: [
          { name: 'userId', type: 'string', required: true, locations: ['body'] },
          { name: 'campusId', type: 'string', required: false, locations: ['body'] },
        ],
        responseValidation: {
          mode: 'httpCode',
          http: { mode: 'any_2xx' },
          body: { expression: '' },
        },
        hasSecrets: false,
        version: 1,
        createdAt: '',
        updatedAt: '',
        createdBy: '',
        updatedBy: '',
      },
    ];

    const hydrated = hydrateExternalApiConditions(root, {
      externalApiBindings,
      externalApis,
      inputFields,
    });

    const condition = hydrated.items[0];
    if (condition.kind !== 'condition' || condition.conditionKind !== 'externalApi') {
      throw new Error('expected external api condition');
    }

    expect(condition.paramMappings.map((mapping) => mapping.paramName)).toEqual([
      'userId',
      'campusId',
      'legacyParam',
    ]);
    expect(condition.paramMappings[1]).toMatchObject({
      paramName: 'campusId',
      paramType: 'string',
      required: false,
      mode: 'input',
      inputRef: null,
      literalValue: '',
    });
    expect(condition.paramMappings[2]).toMatchObject({
      paramName: 'legacyParam',
      paramType: 'any',
      required: false,
      mode: 'literal',
      literalValue: 'legacy',
    });
  });

  it('creates fallback input refs when saved paths are no longer in the schema', () => {
    const root = baseRoot();
    const externalApiBindings: ExternalApiBinding[] = [
      {
        externalApiKey: 'payment-validator',
        paramMappings: [
          {
            paramName: 'userId',
            mode: 'input',
            inputPath: 'derived.missingUserId',
          },
        ],
        failMode: 'open',
        cacheTTL: 0,
      },
    ];

    const hydrated = hydrateExternalApiConditions(root, {
      externalApiBindings,
      externalApis: [],
      inputFields: [],
    });

    const condition = hydrated.items[0];
    if (condition.kind !== 'condition' || condition.conditionKind !== 'externalApi') {
      throw new Error('expected external api condition');
    }

    expect(condition.paramMappings[0]).toMatchObject({
      paramName: 'userId',
      paramType: 'any',
      required: false,
      mode: 'input',
      inputRef: {
        refKind: 'input',
        category: 'derived',
        path: 'derived.missingUserId',
        label: 'derived.missingUserId',
        type: 'unknown',
      },
    });
  });
});
