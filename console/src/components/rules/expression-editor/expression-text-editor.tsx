import { useQuery } from '@tanstack/react-query';
import { useEffect, useRef } from 'react';
import { useTranslation } from 'react-i18next';

import type { ExpressionSchemaField } from '@/api/types';

import { expressionQueries } from '@/queries/expression-queries';


interface ExpressionTextEditorProps {
  value: string;
  onChange: (value: string) => void;
}

async function createEditorView(
  parent: HTMLElement,
  doc: string,
  fields: ExpressionSchemaField[],
  onUpdate: (val: string) => void,
) {
  const { EditorView, keymap } = await import('@codemirror/view');
  const { EditorState } = await import('@codemirror/state');
  const { javascript } = await import('@codemirror/lang-javascript');
  const { autocompletion } = await import('@codemirror/autocomplete');
  const { defaultKeymap } = await import('@codemirror/commands');

  const fieldCompletions = fields.map((f) => ({
    label: f.name,
    type: 'variable' as const,
    detail: f.type,
    info: f.description,
  }));

  const view = new EditorView({
    state: EditorState.create({
      doc,
      extensions: [
        keymap.of(defaultKeymap),
        javascript(),
        autocompletion({
          override: [
            (ctx) => {
              const word = ctx.matchBefore(/[\w.]+/);
              if (!word) return null;
              return { from: word.from, options: fieldCompletions };
            },
          ],
        }),
        EditorView.updateListener.of((update) => {
          if (update.docChanged) {
            onUpdate(update.state.doc.toString());
          }
        }),
        EditorView.theme({
          '&': { fontSize: '13px', maxHeight: '200px' },
          '.cm-scroller': { overflow: 'auto' },
          '.cm-content': { fontFamily: 'monospace' },
        }),
      ],
    }),
    parent,
  });

  return view;
}

export function ExpressionTextEditor({ value, onChange }: ExpressionTextEditorProps) {
  const { t } = useTranslation('rules');
  const containerRef = useRef<HTMLDivElement>(null);
  const viewRef = useRef<{ destroy: () => void } | null>(null);
  const { data: schema } = useQuery(expressionQueries.schema());
  const fields = schema?.fields ?? [];

  useEffect(() => {
    const el = containerRef.current;
    if (!el) return;

    void createEditorView(el, value, fields, onChange).then((view) => {
      viewRef.current = view;
    });

    return () => {
      viewRef.current?.destroy();
      viewRef.current = null;
    };
    // Only re-create on mount/fields change, not on value change
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [fields.length]);

  return (
    <div className="space-y-2">
      <p className="text-muted-foreground text-xs">{t('expression.textMode')}</p>
      <div
        ref={containerRef}
        className="rounded-md border focus-within:ring-2 focus-within:ring-ring"
      />
    </div>
  );
}
