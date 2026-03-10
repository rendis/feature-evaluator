import { Braces, Plus, Trash2 } from 'lucide-react';
import { useTranslation } from 'react-i18next';

import {
  createVisualNodeDraft,
  type BodyRootType,
  type VisualNode,
  type VisualNodeType,
} from './external-api-builder-utils';

import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { cn } from '@/lib/utils';

interface BodyVisualEditorProps {
  nodes: VisualNode[];
  rootType: BodyRootType;
  onChange: (nodes: VisualNode[]) => void;
  title?: string;
  emptyText?: string;
  keyPlaceholder?: string;
  stringValuePlaceholder?: string;
  numberValuePlaceholder?: string;
  addFieldLabel?: string;
  addItemLabel?: string;
  allowTemplateValues?: boolean;
  variant?: 'default' | 'dark';
  showHeader?: boolean;
}

export function BodyVisualEditor({
  nodes,
  rootType,
  onChange,
  title,
  emptyText,
  keyPlaceholder,
  stringValuePlaceholder,
  numberValuePlaceholder = '123',
  addFieldLabel,
  addItemLabel,
  allowTemplateValues = true,
  variant = 'default',
  showHeader = true,
}: BodyVisualEditorProps) {
  const { t } = useTranslation('external-apis');
  const resolvedTitle = title ?? t('sections.bodyStructure');
  const resolvedEmptyText = emptyText ?? t('bodyActions.empty');
  const resolvedKeyPlaceholder = keyPlaceholder ?? t('placeholders.keyName');
  const resolvedStringValuePlaceholder = stringValuePlaceholder ?? t('placeholders.paramValue');
  const resolvedAddFieldLabel = addFieldLabel ?? t('actions.addField');
  const resolvedAddItemLabel = addItemLabel ?? t('actions.addItem');

  const updateNode = (id: string, updater: (node: VisualNode) => VisualNode) => {
    onChange(updateNodeTree(nodes, id, updater));
  };

  const removeNode = (id: string) => {
    onChange(removeNodeTree(nodes, id));
  };

  const addNode = (parentId?: string) => {
    onChange(addNodeTree(nodes, createVisualNodeDraft(), parentId));
  };

  return (
    <div className="space-y-3">
      {showHeader ? (
        <div className="flex items-center justify-between">
          <p
            className={cn(
              'text-sm font-medium',
              variant === 'dark' ? 'text-content-body' : 'text-foreground',
            )}
          >
            {resolvedTitle}
          </p>
          <Button
            type="button"
            variant={variant === 'dark' ? 'secondary' : 'outline'}
            size="sm"
            onClick={() => addNode()}
            className={
              variant === 'dark'
                ? 'border-accent-soft bg-accent-soft text-accent-soft-foreground hover:bg-accent-soft/80'
                : undefined
            }
          >
            <Plus className="mr-2 h-4 w-4" />
            {rootType === 'array' ? resolvedAddItemLabel : resolvedAddFieldLabel}
          </Button>
        </div>
      ) : null}

      {nodes.length === 0 ? (
        <div
          className={cn(
            'rounded-md border border-dashed px-4 py-6 text-sm',
            variant === 'dark'
              ? 'border-border-strong bg-surface-editor text-content-subtle'
              : 'text-muted-foreground',
          )}
        >
          {resolvedEmptyText}
        </div>
      ) : (
        <div className="space-y-3">
          {nodes.map((node, index) => (
            <BodyNodeRow
              key={node.id}
              node={node}
              index={index}
              parentType={rootType}
              level={0}
              keyPlaceholder={resolvedKeyPlaceholder}
              stringValuePlaceholder={resolvedStringValuePlaceholder}
              numberValuePlaceholder={numberValuePlaceholder}
              addFieldLabel={resolvedAddFieldLabel}
              addItemLabel={resolvedAddItemLabel}
              allowTemplateValues={allowTemplateValues}
              variant={variant}
              onAddChild={() => addNode(node.id)}
              onRemove={() => removeNode(node.id)}
              onUpdate={(updater) => updateNode(node.id, updater)}
            />
          ))}
        </div>
      )}
    </div>
  );
}

interface BodyNodeRowProps {
  node: VisualNode;
  index: number;
  parentType: BodyRootType | VisualNodeType;
  level: number;
  keyPlaceholder: string;
  stringValuePlaceholder: string;
  numberValuePlaceholder: string;
  addFieldLabel: string;
  addItemLabel: string;
  allowTemplateValues: boolean;
  variant: 'default' | 'dark';
  onAddChild: () => void;
  onRemove: () => void;
  onUpdate: (updater: (node: VisualNode) => VisualNode) => void;
}

function BodyNodeRow({
  node,
  index,
  parentType,
  level,
  keyPlaceholder,
  stringValuePlaceholder,
  numberValuePlaceholder,
  addFieldLabel,
  addItemLabel,
  allowTemplateValues,
  variant,
  onAddChild,
  onRemove,
  onUpdate,
}: BodyNodeRowProps) {
  const { t } = useTranslation('external-apis');
  const isCollection = node.type === 'object' || node.type === 'array';

  return (
    <div
      className={cn(
        'space-y-2',
        level > 0 &&
          (variant === 'dark' ? 'ml-6 border-l-2 border-border/50 pl-4' : 'ml-5 border-l pl-4'),
      )}
    >
      <div
        className={cn(
          'flex flex-wrap items-start gap-2 rounded-md border p-3',
          variant === 'dark'
            ? 'border-border/60 bg-surface-subtle/80 focus-within:border-ring/50'
            : 'bg-muted/20',
        )}
      >
        {parentType === 'array' ? (
          <div
            className={cn(
              'flex h-10 min-w-20 items-center justify-center rounded-md border px-3 text-xs font-mono',
              variant === 'dark'
                ? 'border-border-strong bg-surface-editor text-content-subtle'
                : 'text-muted-foreground bg-background',
            )}
          >
            [{index}]
          </div>
        ) : (
          <Input
            value={node.key}
            onChange={(event) => onUpdate((current) => ({ ...current, key: event.target.value }))}
            placeholder={keyPlaceholder}
            className={cn(
              'min-w-40 flex-1',
              variant === 'dark' &&
                'border-border bg-background text-foreground placeholder:text-content-subtle',
            )}
          />
        )}

        <select
          value={node.type}
          onChange={(event) =>
            onUpdate((current) => ({
              ...current,
              type: event.target.value as VisualNodeType,
              children:
                event.target.value === 'object' || event.target.value === 'array'
                  ? current.children
                  : [],
            }))
          }
          className={cn(
            'h-10 rounded-md border px-3 pr-10 text-sm',
            variant === 'dark' ? 'border-border bg-background text-content-body' : 'bg-background',
          )}
        >
          <option value="string">{t('types.string')}</option>
          <option value="number">{t('types.number')}</option>
          <option value="boolean">{t('types.boolean')}</option>
          <option value="object">{t('types.object')}</option>
          <option value="array">{t('types.array')}</option>
        </select>

        {isCollection ? (
          <Button
            type="button"
            variant={variant === 'dark' ? 'ghost' : 'outline'}
            size="sm"
            className={cn(
              'h-10',
              variant === 'dark' &&
                'justify-start px-2 text-xs text-accent-soft-foreground hover:bg-transparent hover:text-primary-foreground',
            )}
            onClick={onAddChild}
          >
            <Plus className="mr-2 h-4 w-4" />
            {node.type === 'array' ? addItemLabel : addFieldLabel}
          </Button>
        ) : (
          <>
            {node.type === 'boolean' ? (
              <select
                value={node.value}
                onChange={(event) =>
                  onUpdate((current) => ({ ...current, value: event.target.value }))
                }
                className={cn(
                  'h-10 min-w-36 rounded-md border px-3 pr-10 text-sm',
                  variant === 'dark'
                    ? 'border-border bg-background text-content-body'
                    : 'bg-background',
                )}
              >
                <option value="true">true</option>
                <option value="false">false</option>
              </select>
            ) : (
              <Input
                value={node.value}
                onChange={(event) =>
                  onUpdate((current) => ({ ...current, value: event.target.value }))
                }
                placeholder={
                  node.type === 'number' ? numberValuePlaceholder : stringValuePlaceholder
                }
                className={cn(
                  'min-w-56 flex-[2]',
                  variant === 'dark' &&
                    'border-border bg-background font-mono text-success-soft-foreground placeholder:text-content-subtle',
                )}
              />
            )}
            {allowTemplateValues ? (
              <Button
                type="button"
                variant="outline"
                size="icon"
                className={cn(
                  'h-10 w-10',
                  variant === 'dark' &&
                    'border-border bg-background text-content-muted hover:bg-surface-subtle hover:text-content-strong',
                )}
                onClick={() =>
                  onUpdate((current) => {
                    const rawValue = current.value.trim();
                    const nextValue =
                      rawValue.startsWith('{{') && rawValue.endsWith('}}')
                        ? rawValue.slice(2, -2).trim()
                        : `{{${rawValue}}}`;
                    return { ...current, value: nextValue };
                  })
                }
              >
                <Braces className="h-4 w-4" />
              </Button>
            ) : null}
          </>
        )}

        <Button
          type="button"
          variant="ghost"
          size="icon"
          className={cn(
            'h-10 w-10',
            variant === 'dark'
              ? 'text-danger-soft-foreground/70 hover:bg-transparent hover:text-danger-soft-foreground'
              : undefined,
          )}
          onClick={onRemove}
        >
          <Trash2 className="h-4 w-4" />
        </Button>
      </div>

      {isCollection
        ? node.children.map((child, childIndex) => (
            <BodyNodeRow
              key={child.id}
              node={child}
              index={childIndex}
              parentType={node.type}
              level={level + 1}
              keyPlaceholder={keyPlaceholder}
              stringValuePlaceholder={stringValuePlaceholder}
              numberValuePlaceholder={numberValuePlaceholder}
              addFieldLabel={addFieldLabel}
              addItemLabel={addItemLabel}
              allowTemplateValues={allowTemplateValues}
              variant={variant}
              onAddChild={() =>
                onUpdate((current) => ({
                  ...current,
                  children: addNodeTree(current.children, createVisualNodeDraft(), child.id),
                }))
              }
              onRemove={() =>
                onUpdate((current) => ({
                  ...current,
                  children: removeNodeTree(current.children, child.id),
                }))
              }
              onUpdate={(updater) =>
                onUpdate((current) => ({
                  ...current,
                  children: updateNodeTree(current.children, child.id, updater),
                }))
              }
            />
          ))
        : null}
    </div>
  );
}

function updateNodeTree(
  nodes: VisualNode[],
  id: string,
  updater: (node: VisualNode) => VisualNode,
): VisualNode[] {
  return nodes.map((node) => {
    if (node.id === id) {
      return updater(node);
    }
    if (node.children.length === 0) {
      return node;
    }
    return {
      ...node,
      children: updateNodeTree(node.children, id, updater),
    };
  });
}

function removeNodeTree(nodes: VisualNode[], id: string): VisualNode[] {
  return nodes
    .filter((node) => node.id !== id)
    .map((node) => ({
      ...node,
      children: removeNodeTree(node.children, id),
    }));
}

function addNodeTree(nodes: VisualNode[], draft: VisualNode, parentId?: string): VisualNode[] {
  if (!parentId) {
    return [...nodes, draft];
  }
  return nodes.map((node) => {
    if (node.id === parentId) {
      return {
        ...node,
        children: [...node.children, draft],
      };
    }
    return {
      ...node,
      children: addNodeTree(node.children, draft, parentId),
    };
  });
}
