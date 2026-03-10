import { createFileRoute } from '@tanstack/react-router';

import { FeatureBuilder } from '@/components/features/feature-builder';

export const Route = createFileRoute('/_authenticated/features/new')({
  component: NewFeaturePage,
});

function NewFeaturePage() {
  return <FeatureBuilder />;
}
