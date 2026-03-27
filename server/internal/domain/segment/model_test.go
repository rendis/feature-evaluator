package segment

import "testing"

func TestSegmentNormalizeCacheConfig(t *testing.T) {
	t.Parallel()

	seg := &Segment{
		MembershipCacheEnabled:    true,
		MembershipCacheTTLSeconds: 1,
		RecordCacheEnabled:        true,
		RecordCacheTTLSeconds:     0,
	}

	seg.NormalizeCacheConfig()

	if seg.MembershipCacheTTLSeconds != minCacheTTLSeconds {
		t.Fatalf("MembershipCacheTTLSeconds = %d, want %d", seg.MembershipCacheTTLSeconds, minCacheTTLSeconds)
	}
	if seg.RecordCacheTTLSeconds != defaultSegmentCacheTTLSeconds {
		t.Fatalf("RecordCacheTTLSeconds = %d, want %d", seg.RecordCacheTTLSeconds, defaultSegmentCacheTTLSeconds)
	}

	seg.MembershipCacheEnabled = false
	seg.MembershipCacheTTLSeconds = 120
	seg.RecordCacheEnabled = false
	seg.RecordCacheTTLSeconds = 120
	seg.NormalizeCacheConfig()

	if seg.MembershipCacheTTLSeconds != 0 || seg.RecordCacheTTLSeconds != 0 {
		t.Fatalf("expected disabled cache TTLs to be reset to zero, got membership=%d record=%d", seg.MembershipCacheTTLSeconds, seg.RecordCacheTTLSeconds)
	}
}
