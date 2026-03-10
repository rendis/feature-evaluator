import {
  buildParamCatalog,
  collectSecretKeys,
  detectDraftVariables,
  draftVariablesToExpressionVariables,
  draftVariablesToParams,
  jsonSchemaToExample,
  jsonToVisualNodes,
  parseParamInputValue,
  visualNodesToJson,
} from './external-api-builder-utils';

describe('external api builder utils', () => {
  it('detects url params by domain, path, and query while preserving optional flags', () => {
    const params = buildParamCatalog(
      {
        method: 'POST',
        urlTemplate:
          'https://api.{{env}}.example.com/v1/users/{{user_id}}?campus={{campus_code}}&page={{page}}',
        headers: [{ keyTemplate: 'Authorization', valueTemplate: 'Bearer {{secret.api_token}}' }],
        bodyTemplate: {
          enabled: '{{is_enabled}}',
        },
      },
      [{ name: 'page', type: 'number', required: false, locations: ['url'], urlKind: 'query' }],
    );

    expect(params).toEqual([
      { name: 'campus_code', type: 'any', required: false, locations: ['url'], urlKind: 'query' },
      { name: 'env', type: 'any', required: true, locations: ['url'], urlKind: 'domain' },
      { name: 'is_enabled', type: 'any', required: false, locations: ['body'] },
      { name: 'page', type: 'number', required: false, locations: ['url'], urlKind: 'query' },
      { name: 'user_id', type: 'any', required: true, locations: ['url'], urlKind: 'path' },
    ]);
  });

  it('collects referenced secret keys only once', () => {
    expect(
      collectSecretKeys({
        method: 'GET',
        urlTemplate: 'https://api.example.com?token={{secret.api_token}}',
        headers: [{ keyTemplate: 'Authorization', valueTemplate: 'Bearer {{secret.api_token}}' }],
        bodyTemplate: {
          nested: '{{secret.secondary}}',
        },
      }),
    ).toEqual(['api_token', 'secondary']);
  });

  it('round-trips visual nodes back to JSON', () => {
    const source = {
      campus_code: '{{campus_code}}',
      enabled: true,
      filters: [{ field: 'user_id', value: '{{user_id}}' }],
    };

    const nodes = jsonToVisualNodes(source);

    expect(visualNodesToJson(nodes, 'object')).toEqual(source);
  });

  it('builds a sample object from a json schema', () => {
    expect(
      jsonSchemaToExample({
        type: 'object',
        properties: {
          approved: { type: 'boolean' },
          items: {
            type: 'array',
            items: {
              type: 'object',
              properties: {
                code: { type: 'string' },
              },
            },
          },
        },
      }),
    ).toEqual({
      approved: false,
      items: [{ code: '' }],
    });
  });

  it('keeps plain any inputs as strings unless they look like json', () => {
    expect(parseParamInputValue('1436300002', 'any')).toBe('1436300002');
    expect(parseParamInputValue('{"approved":true}', 'any')).toEqual({ approved: true });
    expect(parseParamInputValue('[1,2,3]', 'any')).toEqual([1, 2, 3]);
    expect(parseParamInputValue('"campus"', 'any')).toBe('campus');
    expect(parseParamInputValue('true', 'any')).toBe(true);
  });

  it('preserves manual variables while re-detecting request placeholders', () => {
    const variables = detectDraftVariables(
      {
        method: 'POST',
        url: 'https://api.example.com/users/{{user_id}}',
        headers: [],
        bodyMode: 'json',
        bodyRaw: '{\n  "enabled": "{{is_enabled}}"\n}',
        bodyVisual: [],
        variables: {
          campus_code: {
            origin: 'manual',
            type: 'string',
            required: true,
            locations: new Set(),
          },
        },
      },
      {
        campus_code: {
          origin: 'manual',
          type: 'string',
          required: true,
          locations: new Set(),
        },
      },
    );

    expect(draftVariablesToParams(variables)).toEqual([
      { name: 'is_enabled', type: 'any', required: false, locations: ['body'] },
      { name: 'user_id', type: 'any', required: true, locations: ['url'], urlKind: 'path' },
    ]);
    expect(draftVariablesToExpressionVariables(variables)).toEqual([
      { name: 'campus_code', type: 'string', required: true },
    ]);
  });
});
