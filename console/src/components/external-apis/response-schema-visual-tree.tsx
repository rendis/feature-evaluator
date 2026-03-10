import { Plus, Trash2 } from 'lucide-react';
import { useTranslation } from 'react-i18next';

import {
  createVisualNodeDraft,
  type BodyRootType,
  type VisualNode,
  type VisualNodeType,
} from './external-api-builder-utils';

interface ResponseSchemaVisualTreeProps {
  nodes: VisualNode[];
  rootType: BodyRootType;
  onChange: (nodes: VisualNode[]) => void;
  addFieldLabel: string;
  addItemLabel: string;
  stringValuePlaceholder: string;
  numberValuePlaceholder: string;
}

export function ResponseSchemaVisualTree({
  nodes,
  rootType,
  onChange,
  addFieldLabel,
  addItemLabel,
  stringValuePlaceholder,
  numberValuePlaceholder,
}: ResponseSchemaVisualTreeProps) {
  const { t } = useTranslation('external-apis');
  const updateVisualNode = (
    currentNodes: VisualNode[],
    id: string,
    field: 'key' | 'type' | 'value',
    value: string,
  ): VisualNode[] =>
    currentNodes.map((node) => {
      if (node.id === id) {
        if (field === 'type') {
          const nextType = value as VisualNodeType;
          return {
            ...node,
            type: nextType,
            children: nextType === 'object' || nextType === 'array' ? node.children : [],
          };
        }

        return {
          ...node,
          [field]: value,
        };
      }

      if (node.children.length === 0) {
        return node;
      }

      return {
        ...node,
        children: updateVisualNode(node.children, id, field, value),
      };
    });

  const removeVisualNode = (currentNodes: VisualNode[], id: string): VisualNode[] =>
    currentNodes
      .filter((node) => node.id !== id)
      .map((node) => ({
        ...node,
        children: removeVisualNode(node.children, id),
      }));

  const addVisualNode = (currentNodes: VisualNode[], parentId?: string): VisualNode[] => {
    const draft = createVisualNodeDraft();
    if (!parentId) {
      return [...currentNodes, draft];
    }

    return currentNodes.map((node) => {
      if (node.id === parentId) {
        return {
          ...node,
          children: [...node.children, draft],
        };
      }

      return {
        ...node,
        children: addVisualNode(node.children, parentId),
      };
    });
  };

  const renderTree = (
    currentNodes: VisualNode[],
    level = 0,
    parentType: BodyRootType | VisualNodeType = rootType,
  ) =>
    currentNodes.map((row, index) => {
      const isCollection = row.type === 'object' || row.type === 'array';

      return (
        <div
          key={row.id}
          className={`flex flex-col gap-1 ${
            level > 0 ? 'mt-2 ml-6 border-l-2 border-border/50 pl-4' : 'mt-2'
          }`}
        >
          <div className="flex flex-wrap items-center gap-2 rounded border border-border/60 bg-surface-subtle/80 p-2 transition-colors focus-within:border-ring/50">
            {parentType === 'array' ? (
              <div className="text-content-subtle flex w-full items-center justify-center rounded border border-border-strong bg-surface-editor px-2 py-1.5 font-mono text-xs sm:w-1/3">
                [{index}]
              </div>
            ) : (
              <input
                type="text"
                placeholder={t('schemaModal.visualKeyPlaceholder')}
                value={row.key}
                onChange={(event) =>
                  onChange(updateVisualNode(nodes, row.id, 'key', event.target.value))
                }
                className="w-full rounded border border-border bg-background px-2 py-1.5 text-xs text-foreground placeholder:text-content-subtle focus:border-ring focus:outline-none sm:w-1/3"
              />
            )}

            <select
              value={row.type}
              onChange={(event) =>
                onChange(updateVisualNode(nodes, row.id, 'type', event.target.value))
              }
              className="w-full rounded border border-border bg-background px-2 py-1.5 pr-8 text-xs text-content-body focus:outline-none sm:w-24"
            >
              <option value="string">{t('types.string')}</option>
              <option value="number">{t('types.number')}</option>
              <option value="boolean">{t('types.boolean')}</option>
              <option value="object">{t('types.object')}</option>
              <option value="array">{t('types.array')}</option>
            </select>

            {isCollection ? (
              <div className="flex flex-1 items-center justify-end px-2">
                <button
                  type="button"
                  onClick={() => onChange(addVisualNode(nodes, row.id))}
                  className="text-accent-soft-foreground hover:text-primary-foreground flex items-center gap-1 text-xs transition-colors"
                >
                  <Plus className="h-3 w-3" />
                  {row.type === 'array' ? addItemLabel : addFieldLabel}
                </button>
              </div>
            ) : row.type === 'boolean' ? (
              <select
                value={row.value}
                onChange={(event) =>
                  onChange(updateVisualNode(nodes, row.id, 'value', event.target.value))
                }
                className="min-w-0 flex-1 rounded border border-border bg-background px-2 py-1.5 pr-8 text-xs text-content-body focus:outline-none"
              >
                <option value="true">true</option>
                <option value="false">false</option>
              </select>
            ) : (
              <div className="flex min-w-0 flex-1 items-center gap-1">
                <input
                  type="text"
                  placeholder={
                    row.type === 'number' ? numberValuePlaceholder : stringValuePlaceholder
                  }
                  value={row.value}
                  onChange={(event) =>
                    onChange(updateVisualNode(nodes, row.id, 'value', event.target.value))
                  }
                  className="min-w-0 flex-1 rounded border border-border bg-background px-2 py-1.5 font-mono text-xs text-success-soft-foreground placeholder:text-content-subtle focus:border-ring focus:outline-none"
                />
              </div>
            )}

            <button
              type="button"
              onClick={() => onChange(removeVisualNode(nodes, row.id))}
              className="text-danger-soft-foreground/70 hover:text-danger-soft-foreground p-1.5 transition-colors"
            >
              <Trash2 className="h-4 w-4" />
            </button>
          </div>

          {isCollection && row.children.length > 0 ? (
            <div className="w-full">{renderTree(row.children, level + 1, row.type)}</div>
          ) : null}
        </div>
      );
    });

  return <>{renderTree(nodes)}</>;
}
