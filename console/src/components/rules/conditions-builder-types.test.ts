import {
  extractBuilderRoot,
  serializeConditionTree,
  withBuilderMetadata,
  withoutBuilderMetadata,
} from './conditions-builder-types';

import type { BuilderGroup } from './conditions-builder-types';

describe('conditions builder types', () => {
  it('serializes static and segment conditions and collects bindings', () => {
    const root: BuilderGroup = {
      id: 'root',
      kind: 'group',
      connector: 'and',
      items: [
        {
          id: 'condition-1',
          kind: 'condition',
          conditionKind: 'static',
          left: {
            refKind: 'input',
            category: 'headers',
            path: 'headers.tenantId',
            label: 'Tenant ID',
            type: 'string',
          },
          operator: '==',
          rightLiteral: 'abc',
        },
        {
          id: 'condition-2',
          kind: 'condition',
          conditionKind: 'segment',
          segmentKey: 'students',
          segmentName: 'Students',
          lookupInputRef: {
            refKind: 'input',
            category: 'derived',
            path: 'derived.userId',
            label: 'User ID',
            type: 'string',
          },
          fieldOps: [
            {
              id: 'op-1',
              fieldPath: 'status',
              fieldLabel: 'Status',
              fieldType: 'string',
              operator: '==',
              rightMode: 'literal',
              rightLiteral: 'active',
              rightInputRef: null,
            },
          ],
          fieldOpsConnector: 'and',
        },
      ],
    };

    const result = serializeConditionTree(root);

    expect(result.expression).toBe(
      'headers.tenantId == "abc" && students.status == "active"',
    );
    expect(result.sourceBindings).toEqual({
      segments: [{ segmentKey: 'students', lookupPath: 'derived.userId' }],
    });
    expect(result.externalApiBindings).toEqual([]);
  });

  it('serializes externalApi conditions with bindings', () => {
    const root: BuilderGroup = {
      id: 'root',
      kind: 'group',
      connector: 'and',
      items: [
        {
          id: 'condition-1',
          kind: 'condition',
          conditionKind: 'externalApi',
          externalApiKey: 'payment-validator',
          externalApiName: 'Payment Validator',
          paramMappings: [
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
          ],
          negate: false,
        },
      ],
    };

    const result = serializeConditionTree(root);

    expect(result.expression).toBe('externalApi("payment-validator")');
    expect(result.externalApiBindings).toEqual([
      {
        externalApiKey: 'payment-validator',
        paramMappings: [
          {
            paramName: 'userId',
            mode: 'input',
            inputPath: 'headers.userId',
            literalValue: '',
          },
        ],
        failMode: 'open',
        cacheTTL: 0,
      },
    ]);
  });

  it('stores and restores builder metadata without leaking stale keys', () => {
    const root: BuilderGroup = {
      id: 'root',
      kind: 'group',
      connector: 'or',
      items: [],
    };

    const metadata = withBuilderMetadata({ other: 'value' }, root);

    expect(extractBuilderRoot(metadata)).toEqual({
      ...root,
      items: expect.any(Array),
    });
    expect(withoutBuilderMetadata(metadata)).toEqual({ other: 'value' });
  });
});
