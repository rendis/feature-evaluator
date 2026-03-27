import { useNavigate } from '@tanstack/react-router';
import { Copy, GripVertical, Pencil, Trash2 } from 'lucide-react';
import { useState } from 'react';
import { useTranslation } from 'react-i18next';
import { toast } from 'sonner';

import { RuleToggle } from './rule-toggle';

import type { Rule } from '@/api/types';

import { ConfirmDialog } from '@/components/shared/confirm-dialog';
import { PermissionButton } from '@/components/shared/permission-button';
import { Badge } from '@/components/ui/badge';
import { useDeleteRule, useReorderRules } from '@/mutations/rule-mutations';

interface RuleListProps {
  featureKey: string;
  rules: Rule[];
}

export function RuleList({ featureKey, rules }: RuleListProps) {
  const { t } = useTranslation('rules');
  const navigate = useNavigate();
  const deleteRule = useDeleteRule(featureKey);
  const reorderRules = useReorderRules(featureKey);
  const [deleteTarget, setDeleteTarget] = useState<Rule | null>(null);
  const [dragIndex, setDragIndex] = useState<number | null>(null);

  const handleDragStart = (index: number) => {
    setDragIndex(index);
  };

  const handleDragOver = (e: React.DragEvent, index: number) => {
    e.preventDefault();
    if (dragIndex === null || dragIndex === index) return;
    const reordered = [...rules];
    const [moved] = reordered.splice(dragIndex, 1);
    reordered.splice(index, 0, moved);
    const ruleIds = reordered.map((r) => r.id);
    reorderRules.mutate(ruleIds);
    setDragIndex(index);
  };

  const handleDragEnd = () => {
    setDragIndex(null);
  };

  const handleDelete = () => {
    if (!deleteTarget) return;
    deleteRule.mutate(deleteTarget.id, {
      onSuccess: () => {
        toast.success(t('delete.success'));
        setDeleteTarget(null);
      },
      onError: () => toast.error(t('delete.error')),
    });
  };

  return (
    <>
      <div className="space-y-1">
        {rules.map((rule, i) => (
          <RuleRow
            key={rule.id}
            rule={rule}
            index={i}
            featureKey={featureKey}
            onDragStart={handleDragStart}
            onDragOver={handleDragOver}
            onDragEnd={handleDragEnd}
            onEdit={() =>
              void navigate({
                to: '/features/$featureKey/rules/$ruleId/edit',
                params: { featureKey, ruleId: rule.id },
              })
            }
            onClone={() =>
              void navigate({
                to: '/features/$featureKey/rules/new',
                params: { featureKey },
                search: { cloneFrom: rule.id },
              })
            }
            onDelete={() => setDeleteTarget(rule)}
            isDragging={dragIndex === i}
          />
        ))}
      </div>
      <ConfirmDialog
        open={!!deleteTarget}
        onOpenChange={(open) => !open && setDeleteTarget(null)}
        title={t('delete.title')}
        description={t('delete.description', { name: deleteTarget?.name })}
        variant="destructive"
        onConfirm={handleDelete}
        loading={deleteRule.isPending}
      />
    </>
  );
}

interface RuleRowProps {
  rule: Rule;
  index: number;
  featureKey: string;
  onDragStart: (index: number) => void;
  onDragOver: (e: React.DragEvent, index: number) => void;
  onDragEnd: () => void;
  onEdit: () => void;
  onClone: () => void;
  onDelete: () => void;
  isDragging: boolean;
}

function RuleRow({
  rule,
  index,
  featureKey,
  onDragStart,
  onDragOver,
  onDragEnd,
  onEdit,
  onClone,
  onDelete,
  isDragging,
}: RuleRowProps) {
  const { t } = useTranslation('rules');

  return (
    <div
      draggable
      onDragStart={() => onDragStart(index)}
      onDragOver={(e) => onDragOver(e, index)}
      onDragEnd={onDragEnd}
      className={`flex items-center gap-3 rounded-md border p-3 transition-opacity ${isDragging ? 'opacity-50' : ''}`}
    >
      <button type="button" className="text-muted-foreground cursor-grab active:cursor-grabbing">
        <GripVertical className="h-4 w-4" />
      </button>
      <div className="flex-1 min-w-0">
        <div className="flex items-center gap-2">
          <span className="text-muted-foreground font-mono text-xs">#{rule.priority}</span>
          <span className="truncate font-medium">{rule.name}</span>
          <Badge variant={rule.enabled ? 'success' : 'secondary'}>
            {rule.enabled ? t('fields.enabled') : t('fields.disabled', { ns: 'common', defaultValue: 'Inactiva' })}
          </Badge>
          {rule.rolloutPercentage != null && rule.rolloutPercentage < 100 && (
            <Badge variant="outline" className="font-mono text-xs">
              {rule.rolloutPercentage}%
            </Badge>
          )}
        </div>
        {rule.expression ? (
          <p className="text-muted-foreground mt-1 truncate font-mono text-xs">{rule.expression}</p>
        ) : null}
      </div>
      <div className="flex items-center gap-2">
        <RuleToggle featureKey={featureKey} rule={rule} />
        <PermissionButton permission="features.write" variant="ghost" size="icon" onClick={onEdit}>
          <Pencil className="h-3 w-3" />
        </PermissionButton>
        <PermissionButton
          permission="features.write"
          variant="ghost"
          size="icon"
          onClick={onClone}
          aria-label={t('clone.action', { defaultValue: 'Clonar' })}
        >
          <Copy className="h-3 w-3" />
        </PermissionButton>
        <PermissionButton
          permission="features.write"
          variant="ghost"
          size="icon"
          className="text-destructive"
          onClick={onDelete}
        >
          <Trash2 className="h-3 w-3" />
        </PermissionButton>
      </div>
    </div>
  );
}
