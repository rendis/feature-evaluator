import { useQuery } from '@tanstack/react-query';
import { AlertCircle, Code } from 'lucide-react';
import { useEffect, useMemo, useRef, useState } from 'react';
import { useTranslation } from 'react-i18next';

import {
  buildResponseExpressionCatalog,
  DEFAULT_EXTERNAL_API_EXPRESSION_PROFILE,
  getResponseExpressionCompletions,
  shouldAutoTriggerCompletion,
} from './response-expression-language';

import type { DraftVariableConfig } from './external-api-builder-utils';
import type { ExternalApiExpressionProfile } from '@/api/types';

import { externalApisApi } from '@/api/external-apis';
import { cn } from '@/lib/utils';
import { externalApiQueries } from '@/queries/external-api-queries';

interface ResponseExpressionEditorProps {
  value: string;
  onChange: (value: string) => void;
  sampleResponseText?: string;
  schema?: Record<string, unknown>;
  variables?: Record<string, DraftVariableConfig>;
  responseHeaderKeys?: string[];
  className?: string;
}

async function createEditorView({
  parent,
  doc,
  onChange,
  getCatalog,
  onBlur,
}: {
  parent: HTMLElement;
  doc: string;
  onChange: (value: string) => void;
  getCatalog: () => ReturnType<typeof buildResponseExpressionCatalog>;
  onBlur: () => void;
}) {
  const { autocompletion, completionKeymap, startCompletion } =
    await import('@codemirror/autocomplete');
  const { defaultKeymap } = await import('@codemirror/commands');
  const { HighlightStyle, StreamLanguage, syntaxHighlighting } =
    await import('@codemirror/language');
  const { EditorState } = await import('@codemirror/state');
  const { EditorView, keymap, placeholder } = await import('@codemirror/view');
  const { tags } = await import('@lezer/highlight');

  const exprLanguage = StreamLanguage.define({
    startState() {
      return {};
    },
    token(stream) {
      if (stream.eatSpace()) {
        return null;
      }

      if (stream.match(/\{\{[^}]*\}\}/)) {
        return 'special';
      }

      if (stream.match(/"(?:[^"\\]|\\.)*"?/)) {
        return 'string';
      }

      if (stream.match(/'(?:[^'\\]|\\.)*'?/)) {
        return 'string';
      }

      if (stream.match(/-?\d+(?:\.\d+)?/)) {
        return 'number';
      }

      if (stream.match(/==|!=|>=|<=|>|</)) {
        return 'operator';
      }

      if (stream.match(/\b(?:contains|startsWith|endsWith|matches|in)\b/)) {
        return 'operator';
      }

      if (stream.match(/\b(?:and|or|not|true|false|null|nil)\b/)) {
        return 'keyword';
      }

      if (stream.match(/\b(?:len|dateBefore|dateAfter|now)\b/)) {
        return 'function';
      }

      if (stream.match(/[()[\]{}.,]/)) {
        return 'punctuation';
      }

      if (stream.match(/[A-Za-z_][A-Za-z0-9_]*/)) {
        return 'variableName';
      }

      stream.next();
      return 'invalid';
    },
  });

  const exprHighlightStyle = HighlightStyle.define([
    { tag: tags.keyword, color: '#f472b6', fontWeight: '600' },
    { tag: tags.string, color: '#fbbf24' },
    { tag: tags.number, color: '#7dd3fc' },
    { tag: [tags.variableName, tags.special(tags.variableName)], color: '#34d399' },
    { tag: tags.function(tags.variableName), color: '#a78bfa' },
    { tag: [tags.operator, tags.punctuation], color: '#fda4af' },
    { tag: tags.invalid, color: '#fb7185', textDecoration: 'underline wavy' },
  ]);

  const completionSource = (
    context: Parameters<NonNullable<NonNullable<Parameters<typeof autocompletion>[0]>['override']>[number]>[0],
  ) => {
    const result = getResponseExpressionCompletions({
      doc: context.state.doc.toString(),
      cursor: context.pos,
      catalog: getCatalog(),
      explicit: context.explicit,
    });

    if (!result) {
      return null;
    }

    return {
      from: result.from,
      options: result.options.map((option) => ({
        label: option.label,
        detail: option.detail,
        type: option.type,
        boost: option.boost,
        apply(view: InstanceType<typeof EditorView>) {
          const selectionOffset = option.selectionOffset ?? option.insertText.length;
          const anchor = option.applyFrom + selectionOffset;
          view.dispatch({
            changes: {
              from: option.applyFrom,
              to: option.applyTo,
              insert: option.insertText,
            },
            selection: { anchor },
          });
          view.focus();
        },
      })),
    };
  };

  const view = new EditorView({
    state: EditorState.create({
      doc,
      extensions: [
        EditorView.lineWrapping,
        exprLanguage,
        syntaxHighlighting(exprHighlightStyle, { fallback: true }),
        placeholder('Ej: response.body.results[0].approved == true'),
        keymap.of([...completionKeymap, ...defaultKeymap]),
        autocompletion({
          override: [completionSource],
          activateOnTyping: false,
          maxRenderedOptions: 12,
        }),
        EditorView.domEventHandlers({
          focus(_event, view) {
            if (view.state.doc.toString().trim().length === 0) {
              startCompletion(view);
            }
            return false;
          },
          blur() {
            onBlur();
            return false;
          },
        }),
        EditorView.updateListener.of((update) => {
          if (update.docChanged) {
            const nextValue = update.state.doc.toString();
            onChange(nextValue);

            if (!update.view.hasFocus) {
              return;
            }

            const selection = update.state.selection.main;
            if (!selection.empty) {
              return;
            }

            const cursor = selection.head;
            if (shouldAutoTriggerCompletion(update.startState.doc.toString(), nextValue, cursor)) {
              startCompletion(update.view);
            }
          }
        }),
        EditorView.theme({
          '&': {
            height: '100%',
            fontSize: '13px',
            caretColor: '#e2e8f0',
          },
          '.cm-scroller': {
            overflow: 'auto',
            fontFamily:
              'ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, Liberation Mono, monospace',
          },
          '.cm-content': {
            minHeight: '160px',
            padding: '12px',
            color: '#e2e8f0',
            caretColor: '#e2e8f0',
          },
          '.cm-line': {
            caretColor: '#e2e8f0',
          },
          '.cm-cursor, .cm-focused .cm-cursor': {
            borderLeftColor: '#e2e8f0',
          },
          '.cm-dropCursor': {
            borderLeftColor: '#e2e8f0',
          },
          '.cm-tooltip-autocomplete': {
            border: '1px solid rgba(71, 85, 105, 0.9)',
            backgroundColor: '#020617',
          },
          '.cm-tooltip-autocomplete ul': {
            fontFamily:
              'ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, Liberation Mono, monospace',
          },
          '.cm-tooltip-autocomplete ul li[aria-selected]': {
            backgroundColor: 'rgba(79, 70, 229, 0.18)',
            color: '#e2e8f0',
          },
          '&.cm-focused': {
            outline: 'none',
          },
        }),
      ],
    }),
    parent,
  });

  return view;
}

export function ResponseExpressionEditor({
  value,
  onChange,
  sampleResponseText,
  schema,
  variables = {},
  responseHeaderKeys = [],
  className,
}: ResponseExpressionEditorProps) {
  const { t } = useTranslation('external-apis');
  const containerRef = useRef<HTMLDivElement>(null);
  const viewRef = useRef<{
    state: { doc: { toString: () => string } };
    dispatch: (spec: unknown) => void;
    destroy: () => void;
    // eslint-disable-next-line @typescript-eslint/no-explicit-any
  } | null>(null) as React.RefObject<any>;
  const latestValueRef = useRef(value);
  const validationSeqRef = useRef(0);
  const [validationError, setValidationError] = useState<string | null>(null);

  const { data: profileData } = useQuery({
    ...externalApiQueries.expressionProfile(),
    retry: false,
  });

  const profile: ExternalApiExpressionProfile =
    profileData ?? DEFAULT_EXTERNAL_API_EXPRESSION_PROFILE;
  const catalog = useMemo(
    () =>
      buildResponseExpressionCatalog({
        profile,
        sampleResponseText,
        schema,
        variables,
        responseHeaderKeys,
      }),
    [profile, responseHeaderKeys, sampleResponseText, schema, variables],
  );
  const latestCatalogRef = useRef(catalog);

  latestCatalogRef.current = catalog;
  latestValueRef.current = value;

  const validateExpression = async (expression: string) => {
    const currentSeq = validationSeqRef.current + 1;
    validationSeqRef.current = currentSeq;

    if (!expression.trim()) {
      setValidationError(null);
      return;
    }

    try {
      const result = await externalApisApi.validateExpression(expression);
      if (validationSeqRef.current !== currentSeq) {
        return;
      }
      setValidationError(result.valid ? null : (result.error ?? t('errors.invalidExpression')));
    } catch {
      if (validationSeqRef.current === currentSeq) {
        setValidationError(null);
      }
    }
  };

  useEffect(() => {
    const element = containerRef.current;
    if (!element) {
      return;
    }

    void createEditorView({
      parent: element,
      doc: value,
      onChange,
      getCatalog: () => latestCatalogRef.current,
      onBlur: () => {
        void validateExpression(latestValueRef.current);
      },
    }).then((view) => {
      viewRef.current = view;
    });

    return () => {
      viewRef.current?.destroy();
      viewRef.current = null;
    };
    // The editor manages its own doc updates after mount.
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  useEffect(() => {
    const currentDoc = viewRef.current?.state.doc.toString();
    if (!viewRef.current || currentDoc === value) {
      return;
    }

    viewRef.current.dispatch({
      changes: {
        from: 0,
        to: currentDoc?.length ?? 0,
        insert: value,
      },
    });
  }, [value]);

  useEffect(() => {
    const handle = window.setTimeout(() => {
      void validateExpression(value);
    }, 300);

    return () => {
      window.clearTimeout(handle);
    };
  }, [value]);

  const hasDynamicResponseSchema =
    typeof sampleResponseText === 'string' && sampleResponseText.trim().length > 0;

  return (
    <div className={cn('flex h-full min-h-[160px] w-full flex-col gap-2', className)}>
      <div
        className={cn(
          'flex min-h-[160px] flex-1 overflow-hidden rounded-lg border bg-slate-950',
          validationError ? 'border-rose-500/40' : 'border-slate-800',
        )}
      >
        <div ref={containerRef} className="min-h-[160px] flex-1" />
      </div>

      {!hasDynamicResponseSchema ? (
        <p className="text-xs text-slate-500">{t('expressionEditor.noSchema')}</p>
      ) : null}

      {validationError ? (
        <p className="flex items-start gap-2 text-xs text-rose-400">
          <AlertCircle className="mt-0.5 h-3.5 w-3.5 shrink-0" />
          <span>{validationError}</span>
        </p>
      ) : (
        <p className="flex items-start gap-2 text-xs text-slate-500">
          <Code className="mt-0.5 h-3.5 w-3.5 shrink-0" />
          <span>
            {t('expressionEditor.shortcuts')}{' '}
            <kbd className="rounded border border-slate-700 px-1 py-0.5">Ctrl</kbd>+
            <kbd className="rounded border border-slate-700 px-1 py-0.5">Space</kbd>
          </span>
        </p>
      )}
    </div>
  );
}
