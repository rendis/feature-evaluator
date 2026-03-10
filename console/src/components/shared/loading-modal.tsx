import * as DialogPrimitive from '@radix-ui/react-dialog';
import { LoaderCircle } from 'lucide-react';

import {
  Dialog,
  DialogDescription,
  DialogHeader,
  DialogOverlay,
  DialogPortal,
  DialogTitle,
} from '@/components/ui/dialog';

interface LoadingModalProps {
  open: boolean;
  title: string;
  description: string;
}

export function LoadingModal({ open, title, description }: LoadingModalProps) {
  return (
    <Dialog open={open}>
      <DialogPortal>
        <DialogOverlay className="z-[80]" />
        <div className="fixed inset-0 z-[81] flex items-center justify-center p-4">
          <DialogPrimitive.Content
            aria-busy="true"
            className="fe-panel-elevated w-full max-w-sm outline-none"
            onEscapeKeyDown={(event) => event.preventDefault()}
            onInteractOutside={(event) => event.preventDefault()}
            onPointerDownOutside={(event) => event.preventDefault()}
          >
            <DialogHeader className="items-center gap-4 px-6 py-8 text-center sm:text-center">
              <div className="bg-accent-soft text-accent-soft-foreground flex h-14 w-14 items-center justify-center rounded-full border border-accent-soft">
                <LoaderCircle className="h-7 w-7 animate-spin" />
              </div>
              <div className="space-y-2">
                <DialogTitle className="text-center text-base">{title}</DialogTitle>
                <DialogDescription className="text-center text-sm">{description}</DialogDescription>
              </div>
              <div aria-live="polite" className="sr-only" role="status">
                {title}
              </div>
            </DialogHeader>
          </DialogPrimitive.Content>
        </div>
      </DialogPortal>
    </Dialog>
  );
}
