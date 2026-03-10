import { useRef } from 'react';

const TEMPLATE_REGEX = /(\{\{[^}]*(?:\}\}|$))/g;

export function renderHighlightedText(
  text: string,
  defaultClass = 'text-content-body',
  placeholder?: string,
) {
  if (!text) {
    return placeholder ? <span className="text-content-subtle">{placeholder}</span> : null;
  }

  return text.split(TEMPLATE_REGEX).map((part, index) => {
    if (part.startsWith('{{')) {
      const isClosed = part.endsWith('}}');
      const innerText = isClosed ? part.slice(2, -2) : part.slice(2);
      const isSnakeCase =
        /^[a-z0-9_]*$/.test(innerText) && innerText.length > 0 && !innerText.startsWith('secret.');

      const bgClass = part.startsWith('{{secret.')
        ? 'rounded-sm bg-syntax-template-secret-bg text-syntax-template-secret'
        : isSnakeCase
          ? 'rounded-sm bg-syntax-template-valid-bg text-syntax-template-valid'
          : 'rounded-sm bg-syntax-template-invalid-bg text-syntax-template-invalid underline decoration-syntax-template-invalid decoration-wavy decoration-1 underline-offset-4';

      return (
        <span key={`${part}-${index}`} className={bgClass}>
          {part}
        </span>
      );
    }

    return (
      <span key={`${part}-${index}`} className={defaultClass}>
        {part}
      </span>
    );
  });
}

interface HighlightedInputProps {
  value: string;
  onChange: (value: string) => void;
  placeholder?: string;
  className?: string;
}

export function HighlightedTemplateInput({
  value,
  onChange,
  placeholder,
  className,
}: HighlightedInputProps) {
  const inputRef = useRef<HTMLInputElement>(null);
  const backdropRef = useRef<HTMLDivElement>(null);

  return (
    <div className={`relative overflow-hidden ${className ?? ''}`}>
      <div
        ref={backdropRef}
        className="pointer-events-none absolute inset-0 overflow-hidden px-4 py-2 font-mono text-sm whitespace-pre"
        aria-hidden="true"
      >
        {renderHighlightedText(value, 'text-content-body', placeholder)}
      </div>
      <input
        ref={inputRef}
        type="text"
        value={value}
        onChange={(event) => onChange(event.target.value)}
        onKeyDown={(event) => {
          const input = event.currentTarget;
          const start = input.selectionStart ?? value.length;
          const end = input.selectionEnd ?? value.length;

          if (event.key === '{' && start > 0 && value.charAt(start - 1) === '{') {
            event.preventDefault();
            const nextValue = `${value.substring(0, start)}{}}${value.substring(end)}`;
            onChange(nextValue);
            setTimeout(() => {
              inputRef.current?.focus();
              inputRef.current?.setSelectionRange(start + 1, start + 1);
            }, 0);
          }
        }}
        onScroll={(event) => {
          if (backdropRef.current) {
            backdropRef.current.scrollLeft = event.currentTarget.scrollLeft;
          }
        }}
        className="caret-foreground absolute inset-0 z-10 h-full w-full bg-transparent px-4 py-2 font-mono text-sm text-transparent outline-none selection:bg-syntax-template-valid-bg selection:text-transparent"
        spellCheck={false}
      />
    </div>
  );
}

interface HighlightedTextareaProps {
  value: string;
  onChange: (value: string) => void;
  placeholder?: string;
  className?: string;
}

export function HighlightedTemplateTextarea({
  value,
  onChange,
  placeholder,
  className,
}: HighlightedTextareaProps) {
  const textareaRef = useRef<HTMLTextAreaElement>(null);
  const backdropRef = useRef<HTMLDivElement>(null);

  return (
    <div className={`relative overflow-hidden ${className ?? ''}`}>
      <div
        ref={backdropRef}
        className="pointer-events-none absolute inset-0 overflow-hidden p-4 font-mono text-sm whitespace-pre-wrap break-words"
        aria-hidden="true"
      >
        {renderHighlightedText(value, 'text-content-body', placeholder)}
      </div>
      <textarea
        ref={textareaRef}
        value={value}
        onChange={(event) => onChange(event.target.value)}
        onKeyDown={(event) => {
          const textarea = event.currentTarget;
          const start = textarea.selectionStart ?? value.length;
          const end = textarea.selectionEnd ?? value.length;

          if (event.key === '{' && start > 0 && value.charAt(start - 1) === '{') {
            event.preventDefault();
            const nextValue = `${value.substring(0, start)}{}}${value.substring(end)}`;
            onChange(nextValue);
            setTimeout(() => {
              textareaRef.current?.focus();
              textareaRef.current?.setSelectionRange(start + 1, start + 1);
            }, 0);
          }
        }}
        onScroll={(event) => {
          if (backdropRef.current) {
            backdropRef.current.scrollTop = event.currentTarget.scrollTop;
            backdropRef.current.scrollLeft = event.currentTarget.scrollLeft;
          }
        }}
        className="caret-foreground absolute inset-0 z-10 h-full w-full resize-none bg-transparent p-4 font-mono text-sm text-transparent outline-none selection:bg-syntax-template-valid-bg selection:text-transparent"
        spellCheck={false}
      />
    </div>
  );
}
