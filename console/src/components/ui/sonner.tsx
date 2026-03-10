import { Toaster as Sonner } from 'sonner';

import { useThemeStore } from '@/stores/theme-store';

function Toaster() {
  const { theme } = useThemeStore();

  return (
    <Sonner
      theme={theme === 'system' ? undefined : theme}
      className="toaster group"
      toastOptions={{
        classNames: {
          toast:
            'group toast group-[.toaster]:border-border-strong group-[.toaster]:bg-popover group-[.toaster]:text-foreground group-[.toaster]:shadow-xl',
          description: 'group-[.toast]:text-muted-foreground',
          actionButton: 'group-[.toast]:bg-primary group-[.toast]:text-primary-foreground',
          cancelButton: 'group-[.toast]:bg-secondary group-[.toast]:text-secondary-foreground',
        },
      }}
    />
  );
}

export { Toaster };
