import { useState } from 'react';

import { ChangeDiffDialog } from './change-diff-dialog';
import { ChangeEventCard } from './change-event-card';

import type { ChangeEntry } from '@/api/changelog';

interface ChangeTimelineProps {
  entries: ChangeEntry[];
}

export function ChangeTimeline({ entries }: ChangeTimelineProps) {
  const [selected, setSelected] = useState<ChangeEntry | null>(null);

  return (
    <>
      <div className="space-y-2">
        {entries.map((entry) => (
          <ChangeEventCard
            key={entry.id}
            entry={entry}
            onClick={
              entry.fieldChanges && entry.fieldChanges.length > 0
                ? () => setSelected(entry)
                : undefined
            }
          />
        ))}
      </div>
      <ChangeDiffDialog
        entry={selected}
        open={!!selected}
        onOpenChange={(open) => !open && setSelected(null)}
      />
    </>
  );
}
