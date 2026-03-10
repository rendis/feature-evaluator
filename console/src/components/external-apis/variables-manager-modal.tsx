import { AlertCircle, ListTree, Plus, Trash2, X } from 'lucide-react';
import { useEffect, useMemo, useState } from 'react';
import { useTranslation } from 'react-i18next';

import type { DraftVariableConfig } from './external-api-builder-utils';

import { Dialog, DialogContent, DialogDescription, DialogTitle } from '@/components/ui/dialog';

interface VariablesManagerModalProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  variables: Record<string, DraftVariableConfig>;
  onAddVariable: () => void;
  onRenameVariable: (currentName: string, nextName: string) => void;
  onRemoveVariable: (name: string) => void;
  onTypeChange: (name: string, type: DraftVariableConfig['type']) => void;
  onRequiredChange: (name: string, required: boolean) => void;
}

export function VariablesManagerModal({
  open,
  onOpenChange,
  variables,
  onAddVariable,
  onRenameVariable,
  onRemoveVariable,
  onTypeChange,
  onRequiredChange,
}: VariablesManagerModalProps) {
  const { t } = useTranslation('external-apis');
  const [nameDrafts, setNameDrafts] = useState<Record<string, string>>({});
  const entries = useMemo(
    () => Object.entries(variables).sort(([left], [right]) => left.localeCompare(right)),
    [variables],
  );
  const manualEntries = useMemo(
    () => entries.filter(([, config]) => config.origin === 'manual'),
    [entries],
  );

  useEffect(() => {
    if (!open) {
      return;
    }

    setNameDrafts((current) => {
      const next: Record<string, string> = {};
      for (const [name, config] of entries) {
        if (config.origin !== 'manual') {
          continue;
        }
        next[name] = current[name] ?? name;
      }
      const currentKeys = Object.keys(current).sort();
      const nextKeys = Object.keys(next).sort();
      if (
        currentKeys.length === nextKeys.length &&
        currentKeys.every((key, index) => key === nextKeys[index] && current[key] === next[key])
      ) {
        return current;
      }
      return next;
    });
  }, [entries, open]);

  const getManualNameError = (currentName: string, draftName: string) => {
    const normalized = draftName.trim();
    if (!normalized) {
      return t('variablesModal.validation.required');
    }
    if (!/^[a-z0-9_]+$/.test(normalized)) {
      return t('variablesModal.validation.snakeCase');
    }

    for (const [name, config] of entries) {
      if (config.origin === 'detected' && name === normalized && name !== currentName) {
        return t('variablesModal.validation.conflict');
      }
      if (config.origin === 'manual' && name === normalized && name !== currentName) {
        return t('variablesModal.validation.duplicate');
      }
    }

    for (const [name, value] of Object.entries(nameDrafts)) {
      if (name === currentName) {
        continue;
      }
      if (value.trim() === normalized) {
        return t('variablesModal.validation.duplicate');
      }
    }

    return null;
  };

  const commitManualName = (currentName: string) => {
    const nextName = (nameDrafts[currentName] ?? currentName).trim();
    const error = getManualNameError(currentName, nextName);
    if (error || nextName === currentName) {
      return;
    }

    onRenameVariable(currentName, nextName);
    setNameDrafts((current) => {
      const next = { ...current };
      delete next[currentName];
      next[nextName] = nextName;
      return next;
    });
  };

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="max-h-[80vh] max-w-4xl gap-0 overflow-hidden border-0 bg-transparent p-0 shadow-none [&>button.absolute]:hidden">
        <div className="fe-panel-elevated flex max-h-[80vh] flex-col overflow-hidden">
          <div className="flex items-center justify-between border-b border-border-strong bg-surface-panel/80 px-6 py-4">
            <div className="min-w-0">
              <DialogTitle className="text-content-strong flex items-center gap-2 text-lg font-bold">
                <ListTree className="text-accent-soft-foreground h-5 w-5" />
                {t('variablesModal.title')}
              </DialogTitle>
              <DialogDescription className="sr-only">
                {t('variablesModal.description')}
              </DialogDescription>
            </div>

            <button
              type="button"
              onClick={() => onOpenChange(false)}
              className="text-content-muted hover:text-content-strong transition-colors"
              aria-label={t('common:actions.cancel')}
            >
              <X className="h-5 w-5" />
            </button>
          </div>

          <div className="flex-1 overflow-y-auto px-6 py-6">
            <div className="mb-6 flex items-start justify-between gap-4">
              <p className="text-content-muted text-sm">{t('variablesModal.helper')}</p>
              <button
                type="button"
                onClick={onAddVariable}
                className="bg-accent-soft text-accent-soft-foreground hover:bg-accent-soft/80 inline-flex shrink-0 items-center gap-1 rounded border border-accent-soft px-3 py-1.5 text-xs font-medium transition-colors"
              >
                <Plus className="h-3.5 w-3.5" />
                {t('variablesModal.addVariable')}
              </button>
            </div>

            {entries.length === 0 ? (
              <div className="fe-empty-surface py-10 text-center">{t('variablesModal.empty')}</div>
            ) : (
              <table className="w-full text-left text-sm">
                <thead>
                  <tr className="text-content-muted border-b border-border">
                    <th className="pb-2 font-medium">{t('variablesModal.columns.name')}</th>
                    <th className="pb-2 font-medium">{t('variablesModal.columns.locations')}</th>
                    <th className="pb-2 font-medium">{t('variablesModal.columns.type')}</th>
                    <th className="pb-2 text-center font-medium">
                      {t('variablesModal.columns.required')}
                    </th>
                    <th className="pb-2 text-right font-medium">
                      {t('variablesModal.columns.actions')}
                    </th>
                  </tr>
                </thead>
                <tbody className="divide-y divide-border-strong/70">
                  {entries.map(([name, config]) => {
                    const isSnakeCase = /^[a-z0-9_]+$/.test(name);
                    const locations = [...config.locations].sort();
                    const forcedRequired = locations.some(
                      (location) => location === 'url_domain' || location === 'url_path',
                    );
                    const nameDraft = nameDrafts[name] ?? name;
                    const nameError =
                      config.origin === 'manual' ? getManualNameError(name, nameDraft) : null;

                    return (
                      <tr key={name} className="group">
                        <td className="py-3 font-mono">
                          {config.origin === 'manual' ? (
                            <div className="space-y-1">
                              <input
                                type="text"
                                value={nameDraft}
                                onChange={(event) =>
                                  setNameDrafts((current) => ({
                                    ...current,
                                    [name]: event.target.value,
                                  }))
                                }
                                onBlur={() => commitManualName(name)}
                                onKeyDown={(event) => {
                                  if (event.key === 'Enter') {
                                    event.preventDefault();
                                    commitManualName(name);
                                  }
                                }}
                                className="fe-editor w-full px-2 py-1 text-xs focus:border-ring focus:outline-none"
                              />
                              {nameError ? (
                                <div className="text-danger-soft-foreground flex items-center gap-1 text-[10px]">
                                  <AlertCircle className="h-3 w-3" />
                                  {nameError}
                                </div>
                              ) : null}
                            </div>
                          ) : (
                            <span
                              className={`flex items-center gap-1.5 ${
                                isSnakeCase
                                  ? 'text-accent-soft-foreground'
                                  : 'text-danger-soft-foreground'
                              }`}
                            >
                              {'{{'}
                              {name}
                              {'}}'}
                              {!isSnakeCase ? (
                                <AlertCircle
                                  className="h-3.5 w-3.5"
                                  aria-label={t('variablesModal.snakeCaseWarning')}
                                />
                              ) : null}
                            </span>
                          )}
                        </td>
                        <td className="py-3">
                          <div className="flex flex-wrap gap-1">
                            {locations.length === 0 ? (
                              <span className="fe-inline-meta">
                                {t('variablesModal.badges.manual')}
                              </span>
                            ) : (
                              locations.map((location) => (
                                <span key={`${name}-${location}`} className="fe-inline-meta">
                                  {location.replace('_', ' ')}
                                </span>
                              ))
                            )}
                          </div>
                        </td>
                        <td className="py-3 pr-4">
                          <select
                            value={config.type}
                            onChange={(event) =>
                              onTypeChange(name, event.target.value as DraftVariableConfig['type'])
                            }
                            className="fe-editor w-full px-2 py-1 pr-8 text-xs focus:border-ring focus:outline-none"
                          >
                            <option value="any">{t('types.any')}</option>
                            <option value="string">{t('types.string')}</option>
                            <option value="number">{t('types.number')}</option>
                            <option value="boolean">{t('types.boolean')}</option>
                          </select>
                        </td>
                        <td className="py-3 text-center">
                          <label className="inline-flex cursor-pointer items-center">
                            <input
                              type="checkbox"
                              checked={config.required}
                              onChange={(event) => onRequiredChange(name, event.target.checked)}
                              disabled={forcedRequired}
                              className="rounded border-border bg-background text-primary focus:ring-ring disabled:opacity-50"
                            />
                          </label>
                          {forcedRequired ? (
                            <div className="text-content-subtle mt-1 text-[10px]">
                              {t('variablesModal.forcedRequired')}
                            </div>
                          ) : null}
                        </td>
                        <td className="py-3 text-right">
                          {config.origin === 'manual' ? (
                            <button
                              type="button"
                              onClick={() => onRemoveVariable(name)}
                              className="text-danger-soft-foreground hover:text-danger-soft-foreground/80 inline-flex items-center gap-1 text-xs transition-colors"
                              aria-label={t('common:actions.delete')}
                            >
                              <Trash2 className="h-3.5 w-3.5" />
                            </button>
                          ) : (
                            <span className="text-content-subtle text-[10px]">
                              {t('variablesModal.badges.detected')}
                            </span>
                          )}
                        </td>
                      </tr>
                    );
                  })}
                </tbody>
              </table>
            )}

            {manualEntries.length === 0 ? (
              <p className="text-content-subtle mt-4 text-xs">{t('variablesModal.manualHint')}</p>
            ) : null}
          </div>

          <div className="flex justify-end border-t border-border-strong bg-surface-panel/40 px-6 py-4">
            <button
              type="button"
              onClick={() => onOpenChange(false)}
              className="rounded-lg bg-primary px-4 py-2 text-sm font-medium text-primary-foreground transition-colors hover:bg-primary/92"
            >
              {t('common:actions.close')}
            </button>
          </div>
        </div>
      </DialogContent>
    </Dialog>
  );
}
