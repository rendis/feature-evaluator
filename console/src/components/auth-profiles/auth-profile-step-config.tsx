import type { DraftState } from './auth-profile-builder-utils';
import { APIKeyEditor } from './auth-profile-api-key-editor';
import { CustomEditor } from './auth-profile-custom-editor';
import { OIDCEditor } from './auth-profile-oidc-editor';

interface StepConfigProps {
  draft: DraftState;
  onChange: React.Dispatch<React.SetStateAction<DraftState>>;
}

export function StepConfig({ draft, onChange }: StepConfigProps) {
  return (
    <div className="min-h-0 flex-1 overflow-y-auto px-1">
      {draft.type === 'api_key' ? <APIKeyEditor draft={draft} onChange={onChange} /> : null}
      {draft.type === 'oidc_standard' ? <OIDCEditor draft={draft} onChange={onChange} /> : null}
      {draft.type === 'custom' ? <CustomEditor draft={draft} onChange={onChange} /> : null}
    </div>
  );
}
