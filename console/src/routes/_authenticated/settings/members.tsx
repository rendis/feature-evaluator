import { useSuspenseQuery } from '@tanstack/react-query';
import { createFileRoute } from '@tanstack/react-router';
import { UserPlus } from 'lucide-react';
import { useState } from 'react';
import { useTranslation } from 'react-i18next';

import { MemberForm } from '@/components/settings/member-form';
import { MemberTable } from '@/components/settings/member-table';
import { EmptyState } from '@/components/shared/empty-state';
import { ApiErrorState } from '@/components/shared/error-state';
import { LoadingSkeleton } from '@/components/shared/loading-skeleton';
import { PageHeader } from '@/components/shared/page-header';
import { PermissionButton } from '@/components/shared/permission-button';
import { memberQueries } from '@/queries/member-queries';

export const Route = createFileRoute('/_authenticated/settings/members')({
  component: MembersPage,
  pendingComponent: () => <LoadingSkeleton rows={5} />,
  errorComponent: ({ error }) => (
    <ApiErrorState error={error} />
  ),
});

function MembersPage() {
  const { t } = useTranslation('settings');
  const [formOpen, setFormOpen] = useState(false);
  const { data: members } = useSuspenseQuery(memberQueries.list());

  return (
    <div className="space-y-6">
      <PageHeader
        title={t('members.title')}
        description={t('members.description')}
        actions={
          <PermissionButton permission="members.manage" onClick={() => setFormOpen(true)}>
            <UserPlus className="mr-2 h-4 w-4" />
            {t('members.addMember')}
          </PermissionButton>
        }
      />

      {members.length === 0 ? (
        <EmptyState title={t('members.empty.title')} description={t('members.empty.description')} />
      ) : (
        <MemberTable members={members} />
      )}

      <MemberForm open={formOpen} onOpenChange={setFormOpen} />
    </div>
  );
}
