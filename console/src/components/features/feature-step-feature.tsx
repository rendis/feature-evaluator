import { useQuery } from '@tanstack/react-query';
import { useState } from 'react';
import { useTranslation } from 'react-i18next';

import type { FeatureDraft } from './feature-builder-utils';
import type { FeatureAccessPolicy } from '@/api/types';
import type { Dispatch, SetStateAction } from 'react';

import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select';
import { normalizeResourceKey, slugifyResourceKey } from '@/lib/resource-key';
import { authProfileQueries } from '@/queries/auth-profile-queries';


interface StepFeatureProps {
  draft: FeatureDraft;
  derivedKey: string;
  isEditing: boolean;
  onChange: Dispatch<SetStateAction<FeatureDraft>>;
}

export function StepFeature({ draft, derivedKey, isEditing, onChange }: StepFeatureProps) {
  const { t } = useTranslation('features');
  const { data: authProfiles = [] } = useQuery(authProfileQueries.list());
  const activeProfiles = authProfiles.filter((p) => p.active);
  const [autoSlug, setAutoSlug] = useState(!isEditing);

  const handleNameChange = (value: string) => {
    onChange((c) => ({
      ...c,
      name: value,
      ...(autoSlug && !isEditing ? { key: slugifyResourceKey(value) } : {}),
    }));
  };

  const handleKeyChange = (value: string) => {
    setAutoSlug(false);
    onChange((c) => ({ ...c, key: normalizeResourceKey(value) }));
  };

  const handleKeyBlur = () => {
    onChange((c) => ({ ...c, key: slugifyResourceKey(c.key) }));
  };

  return (
    <div className="min-h-0 flex-1 space-y-6 overflow-y-auto px-1 py-1">
      <div className="grid gap-4 md:grid-cols-[minmax(0,1.4fr)_minmax(220px,0.9fr)] md:items-start">
        <div className="space-y-2">
          <Label htmlFor="feature-name">{t('fields.name')}</Label>
          <Input
            id="feature-name"
            value={draft.name}
            onChange={(e) => handleNameChange(e.target.value)}
            placeholder={t('fields.name')}
          />
        </div>

        {isEditing ? (
          <div className="space-y-2">
            <Label>{t('fields.key')}</Label>
            <div className="text-muted-foreground border-border/70 bg-muted/35 flex h-10 items-center rounded-md border px-3 font-mono text-sm">
              {draft.key}
            </div>
          </div>
        ) : (
          <div className="space-y-2">
            <Label htmlFor="feature-key">{t('fields.key')}</Label>
            <Input
              id="feature-key"
              value={autoSlug ? derivedKey : draft.key}
              onChange={(e) => handleKeyChange(e.target.value)}
              onBlur={handleKeyBlur}
              placeholder="my_feature"
              className="font-mono"
            />
            <p className="text-muted-foreground min-h-4 text-xs">{t('form.keyPattern')}</p>
          </div>
        )}
      </div>

      <div className="space-y-2">
        <Label htmlFor="feature-description">{t('fields.description')}</Label>
        <Input
          id="feature-description"
          value={draft.description}
          onChange={(e) => onChange((c) => ({ ...c, description: e.target.value }))}
        />
      </div>

      <div className="space-y-2">
        <Label>{t('fields.accessPolicy')}</Label>
        <Select
          value={draft.accessPolicy}
          onValueChange={(v) =>
            onChange((c) => ({ ...c, accessPolicy: v as FeatureAccessPolicy }))
          }
        >
          <SelectTrigger>
            <SelectValue />
          </SelectTrigger>
          <SelectContent>
            <SelectItem value="public">{t('accessPolicy.public')}</SelectItem>
            <SelectItem value="optional">{t('accessPolicy.optional')}</SelectItem>
            <SelectItem value="required">{t('accessPolicy.required')}</SelectItem>
          </SelectContent>
        </Select>
        <p className="text-muted-foreground text-xs">
          {t(`accessPolicyHelp.${draft.accessPolicy}`)}
        </p>
      </div>

      {draft.accessPolicy !== 'public' ? (
        <div className="space-y-2">
          <Label>Auth Profile</Label>
          <Select
            value={draft.authProfileKey || '__none__'}
            onValueChange={(v) =>
              onChange((c) => ({ ...c, authProfileKey: v === '__none__' ? '' : v }))
            }
          >
            <SelectTrigger>
              <SelectValue placeholder="Selecciona un Auth Profile activo" />
            </SelectTrigger>
            <SelectContent>
              <SelectItem value="__none__">Sin Auth Profile</SelectItem>
              {activeProfiles.map((profile) => (
                <SelectItem key={profile.key} value={profile.key}>
                  {profile.name}
                </SelectItem>
              ))}
            </SelectContent>
          </Select>
          <p className="text-muted-foreground text-xs">Solo se listan Auth Profiles activos.</p>
        </div>
      ) : null}
    </div>
  );
}
