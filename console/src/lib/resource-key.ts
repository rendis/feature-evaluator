import { z } from 'zod';

import type {
  FieldPath,
  FieldValues,
  PathValue,
  UseFormRegisterReturn,
  UseFormRegister,
  UseFormSetValue,
} from 'react-hook-form';

export const RESOURCE_KEY_PATTERN = /^[a-z][a-z0-9_]{1,127}$/;

export function normalizeResourceKey(
  value: string,
  { trimTrailingSeparators = false }: { trimTrailingSeparators?: boolean } = {},
): string {
  const normalized = value
    .normalize('NFD')
    .replace(/[\u0300-\u036f]/g, '')
    .toLowerCase()
    .replace(/[^a-z0-9]+/g, '_')
    .replace(/_+/g, '_')
    .replace(/^_+/, '')
    .slice(0, 128);

  if (!normalized) {
    return '';
  }

  const withPrefix = /^[a-z]/.test(normalized) ? normalized : `k_${normalized}`;
  if (!trimTrailingSeparators) {
    return withPrefix.slice(0, 128);
  }

  return withPrefix.slice(0, 128).replace(/_+$/g, '');
}

export function slugifyResourceKey(value: string): string {
  return normalizeResourceKey(value, { trimTrailingSeparators: true });
}

export const resourceKeySchema = z
  .string()
  .transform((value) => slugifyResourceKey(value))
  .pipe(z.string().min(2).max(128).regex(RESOURCE_KEY_PATTERN));

interface NormalizedKeyFieldOptions<
  TFieldValues extends FieldValues,
  TName extends FieldPath<TFieldValues>,
> {
  name: TName;
  onChangeNormalized?: (value: string) => void;
  register: UseFormRegister<TFieldValues>;
  setValue: UseFormSetValue<TFieldValues>;
}

export function buildNormalizedKeyFieldProps<
  TFieldValues extends FieldValues,
  TName extends FieldPath<TFieldValues>,
>({
  name,
  onChangeNormalized,
  register,
  setValue,
}: NormalizedKeyFieldOptions<TFieldValues, TName>) {
  const field = register(name);
  const onBlur: UseFormRegisterReturn<TName>['onBlur'] = (event) => {
    const normalizedValue = slugifyResourceKey(event.target.value);
    event.target.value = normalizedValue;
    setValue(name, normalizedValue as PathValue<TFieldValues, TName>, {
      shouldDirty: true,
      shouldTouch: true,
      shouldValidate: true,
    });
    return field.onBlur(event);
  };
  const onChange: UseFormRegisterReturn<TName>['onChange'] = (event) => {
    const normalizedValue = normalizeResourceKey(event.target.value);
    event.target.value = normalizedValue;
    const result = field.onChange(event);
    onChangeNormalized?.(normalizedValue);
    return result;
  };

  return {
    ...field,
    onBlur,
    onChange,
  };
}
