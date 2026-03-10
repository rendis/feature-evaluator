import { normalizeResourceKey, slugifyResourceKey } from './resource-key';

describe('resource key helpers', () => {
  it('collapses consecutive separators while preserving a trailing underscore during typing', () => {
    expect(normalizeResourceKey('qw-w-w------')).toBe('qw_w_w_');
  });

  it('applies the final normalization used on blur and schema preprocessing', () => {
    expect(slugifyResourceKey('Mi Flag.Á_Test@2026---')).toBe('mi_flag_a_test_2026');
  });

  it('prefixes keys that do not start with a letter', () => {
    expect(slugifyResourceKey('2026 feature')).toBe('k_2026_feature');
  });
});
