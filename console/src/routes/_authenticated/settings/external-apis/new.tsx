import { createFileRoute } from '@tanstack/react-router';

import { ExternalApiBuilder } from '@/components/external-apis/external-api-builder';

export const Route = createFileRoute('/_authenticated/settings/external-apis/new')({
  component: ExternalApiCreatePage,
});

function ExternalApiCreatePage() {
  return <ExternalApiBuilder key="new" />;
}
