import { getLocaleCatalogs } from '@/test/locale-resources';

function flattenCatalog(
  value: unknown,
  prefix = '',
  output: Record<string, string> = {},
): Record<string, string> {
  if (typeof value === 'string') {
    output[prefix] = value;
    return output;
  }

  if (!value || typeof value !== 'object' || Array.isArray(value)) {
    return output;
  }

  for (const [key, child] of Object.entries(value)) {
    const nextPrefix = prefix ? `${prefix}.${key}` : key;
    flattenCatalog(child, nextPrefix, output);
  }

  return output;
}

describe('locale catalogs', () => {
  it('keeps identical namespace files and translation keys for es and en', () => {
    const catalogs = getLocaleCatalogs();
    const esNamespaces = Object.keys(catalogs.es ?? {}).sort();
    const enNamespaces = Object.keys(catalogs.en ?? {}).sort();

    expect(esNamespaces).toEqual(enNamespaces);

    for (const namespace of esNamespaces) {
      const esFlat = flattenCatalog(catalogs.es?.[namespace]);
      const enFlat = flattenCatalog(catalogs.en?.[namespace]);

      expect(Object.keys(esFlat).sort()).toEqual(Object.keys(enFlat).sort());

      for (const [key, value] of Object.entries(esFlat)) {
        expect(value.trim(), `es:${namespace}.${key}`).not.toHaveLength(0);
      }

      for (const [key, value] of Object.entries(enFlat)) {
        expect(value.trim(), `en:${namespace}.${key}`).not.toHaveLength(0);
      }
    }
  });
});
