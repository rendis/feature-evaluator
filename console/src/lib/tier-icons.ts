export interface TierDef {
  key: string;
  name: string;
  color: string;
  spriteId: string;
  category: string;
}

export const TIER_CATEGORIES = [
  'Entry / Access',
  'Growth',
  'Advanced',
  'Top Level',
  'Specials',
  'Technical / Staff',
] as const;

export const TIERS: TierDef[] = [
  // Entry / Access
  { key: 'free', name: 'Free', color: '#6B7280', spriteId: 'tier-free', category: 'Entry / Access' },
  { key: 'starter', name: 'Starter', color: '#64748B', spriteId: 'tier-starter', category: 'Entry / Access' },
  { key: 'hobby', name: 'Hobby', color: '#78716C', spriteId: 'tier-hobby', category: 'Entry / Access' },
  { key: 'community', name: 'Community', color: '#71717A', spriteId: 'tier-community', category: 'Entry / Access' },
  { key: 'lite', name: 'Lite', color: '#9CA3AF', spriteId: 'tier-lite', category: 'Entry / Access' },
  // Growth
  { key: 'basic', name: 'Basic', color: '#3B82F6', spriteId: 'tier-basic', category: 'Growth' },
  { key: 'standard', name: 'Standard', color: '#2563EB', spriteId: 'tier-standard', category: 'Growth' },
  { key: 'plus', name: 'Plus', color: '#0EA5E9', spriteId: 'tier-plus', category: 'Growth' },
  { key: 'growth', name: 'Growth', color: '#06B6D4', spriteId: 'tier-growth', category: 'Growth' },
  // Advanced
  { key: 'pro', name: 'Pro', color: '#8B5CF6', spriteId: 'tier-pro', category: 'Advanced' },
  { key: 'professional', name: 'Professional', color: '#7C3AED', spriteId: 'tier-professional', category: 'Advanced' },
  { key: 'business', name: 'Business', color: '#6366F1', spriteId: 'tier-business', category: 'Advanced' },
  { key: 'premium', name: 'Premium', color: '#A855F7', spriteId: 'tier-premium', category: 'Advanced' },
  // Top Level
  { key: 'enterprise', name: 'Enterprise', color: '#F59E0B', spriteId: 'tier-enterprise', category: 'Top Level' },
  { key: 'ultimate', name: 'Ultimate', color: '#EAB308', spriteId: 'tier-ultimate', category: 'Top Level' },
  { key: 'elite', name: 'Elite', color: '#D97706', spriteId: 'tier-elite', category: 'Top Level' },
  { key: 'corporate', name: 'Corporate', color: '#CA8A04', spriteId: 'tier-corporate', category: 'Top Level' },
  // Specials
  { key: 'founder', name: 'Founder', color: '#10B981', spriteId: 'tier-founder', category: 'Specials' },
  { key: 'early-adopter', name: 'Early Adopter', color: '#14B8A6', spriteId: 'tier-early-adopter', category: 'Specials' },
  { key: 'legacy', name: 'Legacy', color: '#059669', spriteId: 'tier-legacy', category: 'Specials' },
  { key: 'non-profit', name: 'Non-Profit', color: '#F472B6', spriteId: 'tier-non-profit', category: 'Specials' },
  // Technical / Staff
  { key: 'internal', name: 'Internal', color: '#EF4444', spriteId: 'tier-internal', category: 'Technical / Staff' },
  { key: 'test', name: 'Test', color: '#F97316', spriteId: 'tier-test', category: 'Technical / Staff' },
  { key: 'beta', name: 'Beta', color: '#FB923C', spriteId: 'tier-beta', category: 'Technical / Staff' },
  { key: 'alpha', name: 'Alpha', color: '#FBBF24', spriteId: 'tier-alpha', category: 'Technical / Staff' },
  { key: 'staging', name: 'Staging', color: '#A3E635', spriteId: 'tier-staging', category: 'Technical / Staff' },
  { key: 'admin', name: 'Admin', color: '#DC2626', spriteId: 'tier-admin', category: 'Technical / Staff' },
];

export function findTier(key: string): TierDef | undefined {
  return TIERS.find((t) => t.key === key);
}

export function getTiersByCategory(category: string): TierDef[] {
  return TIERS.filter((t) => t.category === category);
}
