package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestCardManager_Expiration(t *testing.T) {
	cm := NewCardManager()
	defer cm.Close()

	now := time.Now()
	freshCard := Card{
		ID:        "fresh-1",
		Type:      CardTypeBabel,
		Title:     "Fresh Card",
		CreatedAt: now,
	}
	expiredCard := Card{
		ID:        "expired-1",
		Type:      CardTypeBabel,
		Title:     "Expired Card",
		CreatedAt: now.Add(-25 * time.Hour),
	}

	cm.InjectBabelCards([]Card{freshCard, expiredCard})

	// GetTempCards should lazily filter and purge expired cards
	active := cm.GetTempCards()
	if len(active) != 1 {
		t.Fatalf("expected 1 active card, got %d", len(active))
	}
	if active[0].ID != "fresh-1" {
		t.Fatalf("expected fresh-1 to remain, got %s", active[0].ID)
	}

	// Verify the expired card was purged from internal storage
	cm.mutex.RLock()
	_, exists := cm.tempCards["expired-1"]
	cm.mutex.RUnlock()
	if exists {
		t.Fatal("expected expired-1 to be deleted from tempCards map")
	}
}

func TestCardManager_CleanExpiredCards(t *testing.T) {
	cm := NewCardManager()
	defer cm.Close()

	now := time.Now()
	cards := []Card{
		{ID: "c1", Type: CardTypeBabel, CreatedAt: now.Add(-26 * time.Hour)},
		{ID: "c2", Type: CardTypeBabel, CreatedAt: now.Add(-2 * time.Hour)},
		{ID: "c3", Type: CardTypeChatLog, CreatedAt: now.Add(-30 * time.Hour)},
	}

	cm.InjectBabelCards([]Card{cards[0], cards[1]})
	cm.InjectChatLogCard(cards[2])

	removed := cm.CleanExpiredCards()
	if removed != 2 {
		t.Fatalf("expected 2 removed cards, got %d", removed)
	}

	active := cm.GetTempCards()
	if len(active) != 1 || active[0].ID != "c2" {
		t.Fatalf("expected only c2 to remain, got %+v", active)
	}
}

func TestCardManager_EvictTempCard(t *testing.T) {
	cm := NewCardManager()
	defer cm.Close()

	cm.InjectBabelCards([]Card{{ID: "target-1", Type: CardTypeBabel, CreatedAt: time.Now()}})

	if !cm.EvictTempCard("target-1") {
		t.Fatal("expected EvictTempCard to return true for existing card")
	}
	if cm.EvictTempCard("target-1") {
		t.Fatal("expected EvictTempCard to return false for already deleted card")
	}
	if cm.EvictTempCard("non-existent") {
		t.Fatal("expected EvictTempCard to return false for non-existent card")
	}

	if len(cm.GetTempCards()) != 0 {
		t.Fatal("expected tempCards to be empty after eviction")
	}
}

func TestExtractHashtags(t *testing.T) {
	text := "今天测试了 #WebKitGTK 4.1 和 #跨平台UI, 以及 #v1.0.0。顺便体验了 #松弛感！"
	tags := ExtractHashtags(text)

	expected := []string{"WebKitGTK", "跨平台UI", "v1.0.0", "松弛感"}
	if len(tags) != len(expected) {
		t.Fatalf("expected %d tags, got %d: %v", len(expected), len(tags), tags)
	}
	for i, exp := range expected {
		if tags[i] != exp {
			t.Errorf("expected tag %d to be %q, got %q", i, exp, tags[i])
		}
	}
}

func TestStripAndExtractHashtags_UnifiedKeywords(t *testing.T) {
	text := "今天测试了 #WebKitGTK 4.1 和 #跨平台UI。\n\n#松弛感 #治愈"
	cleanBody, tags := StripAndExtractHashtags(text)

	if cleanBody != "今天测试了 WebKitGTK 4.1 和 跨平台UI。" {
		t.Fatalf("unexpected cleanBody: %q", cleanBody)
	}

	expectedTags := []string{"WebKitGTK", "跨平台UI", "松弛感", "治愈"}
	if len(tags) != len(expectedTags) {
		t.Fatalf("expected %d tags, got %d: %v", len(expectedTags), len(tags), tags)
	}
	for i, exp := range expectedTags {
		if tags[i] != exp {
			t.Errorf("expected tag %d to be %q, got %q", i, exp, tags[i])
		}
	}
}

func TestCardManager_PromoteToLocal_LosslessPreservation(t *testing.T) {
	cm := NewCardManager()
	defer cm.Close()

	tmpHome := t.TempDir()
	now := time.Now()

	// 1. Test Babel Card Lossless Promotion
	babelCard := Card{
		ID:             "temp-babel-001",
		Type:           CardTypeBabel,
		Origin:         "atproto",
		Title:          "Bluesky Intelligence Insight",
		Content:        "Fascinating distributed network pulse",
		AuthorName:     "quantum.bsky.social",
		AuthorAvatar:   "https://cdn.bsky.app/avatar.jpg",
		PublishedAt:    "2026-10-30T10:00:00Z",
		SourceURL:      "https://bsky.app/profile/quantum/post/123",
		EchoOf:         "mother-card-xyz",
		MatchedAnchor:  "分布式架构",
		EchoReason:     "共鸣命中分布式主题",
		ReplyCount:     5,
		RepostCount:    12,
		LikeCount:      42,
		CreatedAt:      now,
	}

	cm.InjectBabelCards([]Card{babelCard})

	promoted, err := cm.PromoteToLocal("temp-babel-001", tmpHome)
	if err != nil {
		t.Fatalf("PromoteToLocal failed: %v", err)
	}

	// In-memory verification
	if promoted.Type != CardTypeBabel {
		t.Errorf("expected promoted card type %q, got %q", CardTypeBabel, promoted.Type)
	}
	if promoted.Origin != "atproto" {
		t.Errorf("expected promoted card origin %q, got %q", "atproto", promoted.Origin)
	}
	if promoted.AuthorName != "quantum.bsky.social" {
		t.Errorf("expected AuthorName %q, got %q", "quantum.bsky.social", promoted.AuthorName)
	}
	if promoted.AuthorAvatar != "https://cdn.bsky.app/avatar.jpg" {
		t.Errorf("expected AuthorAvatar %q, got %q", "https://cdn.bsky.app/avatar.jpg", promoted.AuthorAvatar)
	}
	if promoted.SourceURL != "https://bsky.app/profile/quantum/post/123" {
		t.Errorf("expected SourceURL %q, got %q", "https://bsky.app/profile/quantum/post/123", promoted.SourceURL)
	}
	if promoted.FilePath == "" {
		t.Fatal("expected non-empty FilePath")
	}

	// Verify file content on disk
	contentBytes, err := os.ReadFile(promoted.FilePath)
	if err != nil {
		t.Fatalf("failed to read written file: %v", err)
	}
	contentStr := string(contentBytes)
	if !filepath.IsAbs(promoted.FilePath) {
		t.Errorf("expected absolute FilePath, got %s", promoted.FilePath)
	}

	// 2. Parse file back via ParseCardFile (simulating LoadLocalCards)
	loaded, err := ParseCardFile(promoted.FilePath)
	if err != nil {
		t.Fatalf("ParseCardFile failed: %v", err)
	}

	if loaded.Type != CardTypeBabel {
		t.Errorf("loaded card expected type %q, got %q", CardTypeBabel, loaded.Type)
	}
	if loaded.Origin != "atproto" {
		t.Errorf("loaded card expected origin %q, got %q", "atproto", loaded.Origin)
	}
	if loaded.AuthorName != "quantum.bsky.social" {
		t.Errorf("loaded card expected AuthorName %q, got %q", "quantum.bsky.social", loaded.AuthorName)
	}
	if loaded.AuthorAvatar != "https://cdn.bsky.app/avatar.jpg" {
		t.Errorf("loaded card expected AuthorAvatar %q, got %q", "https://cdn.bsky.app/avatar.jpg", loaded.AuthorAvatar)
	}
	if loaded.PublishedAt != "2026-10-30T10:00:00Z" {
		t.Errorf("loaded card expected PublishedAt %q, got %q", "2026-10-30T10:00:00Z", loaded.PublishedAt)
	}
	if loaded.SourceURL != "https://bsky.app/profile/quantum/post/123" {
		t.Errorf("loaded card expected SourceURL %q, got %q", "https://bsky.app/profile/quantum/post/123", loaded.SourceURL)
	}
	if loaded.EchoOf != "mother-card-xyz" {
		t.Errorf("loaded card expected EchoOf %q, got %q", "mother-card-xyz", loaded.EchoOf)
	}
	if loaded.MatchedAnchor != "分布式架构" {
		t.Errorf("loaded card expected MatchedAnchor %q, got %q", "分布式架构", loaded.MatchedAnchor)
	}
	if loaded.EchoReason != "共鸣命中分布式主题" {
		t.Errorf("loaded card expected EchoReason %q, got %q", "共鸣命中分布式主题", loaded.EchoReason)
	}
	if loaded.ReplyCount != 5 || loaded.RepostCount != 12 || loaded.LikeCount != 42 {
		t.Errorf("loaded counts mismatch: reply=%d repost=%d like=%d", loaded.ReplyCount, loaded.RepostCount, loaded.LikeCount)
	}

	// 3. Test Curio Card Lossless Promotion
	curioCard := Card{
		ID:           "temp-curio-002",
		Type:         CardTypeCurio,
		Title:        "Vintage Camera Specimen",
		Content:      "Found rare Leica M3",
		Brand:        "Leica",
		Platform:     "eBay",
		Price:        "¥12000",
		Intent:       "收藏鉴赏",
		Availability: "在售",
		SnapshotPath: "/tmp/snapshot.png",
		CreatedAt:    now,
	}

	cm.mutex.Lock()
	cm.tempCards[curioCard.ID] = curioCard
	cm.mutex.Unlock()

	promotedCurio, err := cm.PromoteToLocal("temp-curio-002", tmpHome)
	if err != nil {
		t.Fatalf("PromoteToLocal curio failed: %v", err)
	}
	if promotedCurio.Type != CardTypeCurio {
		t.Errorf("expected curio card type %q, got %q", CardTypeCurio, promotedCurio.Type)
	}

	loadedCurio, err := ParseCardFile(promotedCurio.FilePath)
	if err != nil {
		t.Fatalf("ParseCardFile curio failed: %v", err)
	}
	if loadedCurio.Type != CardTypeCurio {
		t.Errorf("loaded curio expected type %q, got %q", CardTypeCurio, loadedCurio.Type)
	}
	if loadedCurio.Brand != "Leica" || loadedCurio.Platform != "eBay" || loadedCurio.Price != "¥12000" {
		t.Errorf("loaded curio metadata mismatch: %+v", loadedCurio)
	}
	_ = contentStr
}

func TestCardManager_PromoteToLocalWithOpts_UserCommentAndEchoOf(t *testing.T) {
	cm := NewCardManager()
	defer cm.Close()

	tmpHome := t.TempDir()
	now := time.Now()

	// 1. Test Comment and EchoOf override promotion
	card1 := Card{
		ID:          "temp-comment-001",
		Type:        CardTypeBabel,
		Origin:      "bugu",
		Title:       "Distributed Systems Paper",
		Content:     "Consensus mechanisms in decentralized networks",
		EchoOf:      "mother-orig-999",
		CreatedAt:   now,
	}

	cm.mutex.Lock()
	cm.tempCards[card1.ID] = card1
	cm.mutex.Unlock()

	promoted1, err := cm.PromoteToLocalWithOpts("temp-comment-001", tmpHome, "这是一个精妙的灵感补充批注", "mother-target-888")
	if err != nil {
		t.Fatalf("PromoteToLocalWithOpts failed: %v", err)
	}

	if promoted1.UserComment != "这是一个精妙的灵感补充批注" {
		t.Errorf("expected UserComment %q, got %q", "这是一个精妙的灵感补充批注", promoted1.UserComment)
	}
	if promoted1.EchoOf != "mother-target-888" {
		t.Errorf("expected EchoOf %q, got %q", "mother-target-888", promoted1.EchoOf)
	}

	// Verify file content written to disk
	bytes1, err := os.ReadFile(promoted1.FilePath)
	if err != nil {
		t.Fatalf("failed to read written file: %v", err)
	}
	str1 := string(bytes1)
	if !strings.Contains(str1, "user_comment: \"这是一个精妙的灵感补充批注\"") {
		t.Errorf("expected frontmatter to contain user_comment, got:\n%s", str1)
	}
	if !strings.Contains(str1, "echo_of: mother-target-888") {
		t.Errorf("expected frontmatter to contain echo_of, got:\n%s", str1)
	}

	// Verify ParseCardFile deserialization
	loaded1, err := ParseCardFile(promoted1.FilePath)
	if err != nil {
		t.Fatalf("ParseCardFile failed: %v", err)
	}
	if loaded1.UserComment != "这是一个精妙的灵感补充批注" {
		t.Errorf("loaded UserComment mismatch: expected %q, got %q", "这是一个精妙的灵感补充批注", loaded1.UserComment)
	}
	if loaded1.EchoOf != "mother-target-888" {
		t.Errorf("loaded EchoOf mismatch: expected %q, got %q", "mother-target-888", loaded1.EchoOf)
	}

	// 2. Test explicit detachment via "NONE"
	card2 := Card{
		ID:        "temp-detach-002",
		Type:      CardTypeBabel,
		Origin:    "bugu",
		Title:     "Standalone Candidate",
		Content:   "Decoupled from mother card",
		EchoOf:    "mother-orig-999",
		CreatedAt: now,
	}

	cm.mutex.Lock()
	cm.tempCards[card2.ID] = card2
	cm.mutex.Unlock()

	promoted2, err := cm.PromoteToLocalWithOpts("temp-detach-002", tmpHome, "", "NONE")
	if err != nil {
		t.Fatalf("PromoteToLocalWithOpts with NONE failed: %v", err)
	}

	if promoted2.EchoOf != "" {
		t.Errorf("expected empty EchoOf on detached card, got %q", promoted2.EchoOf)
	}

	bytes2, err := os.ReadFile(promoted2.FilePath)
	if err != nil {
		t.Fatalf("failed to read file: %v", err)
	}
	str2 := string(bytes2)
	if strings.Contains(str2, "echo_of:") {
		t.Errorf("detached card should not contain echo_of in frontmatter:\n%s", str2)
	}

	loaded2, err := ParseCardFile(promoted2.FilePath)
	if err != nil {
		t.Fatalf("ParseCardFile failed: %v", err)
	}
	if loaded2.EchoOf != "" {
		t.Errorf("expected loaded detached card EchoOf to be empty, got %q", loaded2.EchoOf)
	}
}

