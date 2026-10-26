package chatlog

import (
	"context"
	"strings"
	"testing"
	"time"

	"Finnegan/core/markdown"
	"Finnegan/core/module"
)

const exactWeChatSample = `巧先生
2026年10月14日  8:15
哥们 argon暂时体验一下就好~~我刚15分钟跑掉10% 就改了几个前端交互 简直了……

枫
2026年10月14日  9:08
啊[捂脸]这么吓人

枫
2026年10月14日  9:08
我就说刚体验了下，赶紧关了

巧先生
2026年10月14日  9:09
按这么算5小时额度只够用1个多小时的hhh

巧先生
2026年10月14日  9:09
体验下 看看有什么区别~~

巧先生
2026年10月14日  9:09
虽然我是没太看出来……

枫
2026年10月14日  9:10
说话语气正常了hhhh

枫
2026年10月14日  9:10
3.8flash动不动就是 您说的太对了！

巧先生
2026年10月14日  9:11
！！ 这种优化方向 有点东西啊~

巧先生
2026年10月14日  9:11
现在是说您说得对 还是 就不这么说了hhh~

巧先生
2026年10月14日  9:11
gpt那边就是总是 您是对的

枫
2026年10月14日  9:12
现在就不这么说了

枫
2026年10月14日  9:12
就是真的感觉是个正常人说话

巧先生
2026年10月14日  9:12
那还是有点东西的~

枫
2026年10月14日  9:12
gpt那个就别提了，全都是你说得对，感觉在阴阳我

巧先生
2026年10月14日  9:12
hhhh~

巧先生
2026年10月14日  9:13
优化代码优化傻了~ 微信聊天记录是这样的格式 看起来还行 但是就是如果有图片 就没法复制了 不过也没有办法了 微信的复制机制就是这样的……`

func TestParseExactWeChatSample(t *testing.T) {
	// 1. Verify detection
	if !IsChatLog(exactWeChatSample) {
		t.Fatalf("expected IsChatLog to be true for WeChat sample")
	}

	// 2. Parse chat log
	parsed, err := Parse(exactWeChatSample)
	if err != nil {
		t.Fatalf("Parse returned unexpected error: %v", err)
	}

	// 3. Verify total raw messages
	if len(parsed.Messages) != 17 {
		t.Fatalf("expected 17 messages, got %d", len(parsed.Messages))
	}

	// 4. Verify participants
	expectedParticipants := []string{"巧先生", "枫"}
	if len(parsed.Participants) != len(expectedParticipants) {
		t.Fatalf("expected %d participants, got %d: %v", len(expectedParticipants), len(parsed.Participants), parsed.Participants)
	}
	for i, p := range expectedParticipants {
		if parsed.Participants[i] != p {
			t.Errorf("participant[%d] expected %q, got %q", i, p, parsed.Participants[i])
		}
	}

	// 5. Verify timestamp lifting (started_at / ended_at)
	if parsed.StartedAt == nil {
		t.Fatalf("expected non-nil StartedAt")
	}
	if parsed.EndedAt == nil {
		t.Fatalf("expected non-nil EndedAt")
	}

	expectedStarted := "2026-10-14T08:15:00+08:00"
	actualStarted := parsed.StartedAt.Format(time.RFC3339)
	if actualStarted != expectedStarted {
		t.Errorf("StartedAt expected %s, got %s", expectedStarted, actualStarted)
	}

	expectedEnded := "2026-10-14T09:13:00+08:00"
	actualEnded := parsed.EndedAt.Format(time.RFC3339)
	if actualEnded != expectedEnded {
		t.Errorf("EndedAt expected %s, got %s", expectedEnded, actualEnded)
	}

	// 6. Verify coalescing structure
	expectedCoalesced := `巧先生:
哥们 argon暂时体验一下就好~~我刚15分钟跑掉10% 就改了几个前端交互 简直了……

枫:
啊[捂脸]这么吓人
我就说刚体验了下，赶紧关了

巧先生:
按这么算5小时额度只够用1个多小时的hhh
体验下 看看有什么区别~~
虽然我是没太看出来……

枫:
说话语气正常了hhhh
3.8flash动不动就是 您说的太对了！

巧先生:
！！ 这种优化方向 有点东西啊~
现在是说您说得对 还是 就不这么说了hhh~
gpt那边就是总是 您是对的

枫:
现在就不这么说了
就是真的感觉是个正常人说话

巧先生:
那还是有点东西的~

枫:
gpt那个就别提了，全都是你说得对，感觉在阴阳我

巧先生:
hhhh~
优化代码优化傻了~ 微信聊天记录是这样的格式 看起来还行 但是就是如果有图片 就没法复制了 不过也没有办法了 微信的复制机制就是这样的……`

	if parsed.Coalesced != expectedCoalesced {
		t.Errorf("coalesced text mismatch.\nExpected:\n%s\n\nActual:\n%s", expectedCoalesced, parsed.Coalesced)
	}

	// 7. Verify Markdown card generation
	cardMD := FormatCardMarkdown("01JAX9K2TEST", "吐槽 Argon 额度消耗与大模型说话语气", parsed)
	if !strings.Contains(cardMD, "type: chat_log") {
		t.Errorf("card markdown missing 'type: chat_log'")
	}
	if !strings.Contains(cardMD, "started_at: 2026-10-14T08:15:00+08:00") {
		t.Errorf("card markdown missing correct started_at")
	}
	if !strings.Contains(cardMD, "ended_at: 2026-10-14T09:13:00+08:00") {
		t.Errorf("card markdown missing correct ended_at")
	}
	if !strings.Contains(cardMD, "created_at: 2026-10-14T08:15:00+08:00") {
		t.Errorf("card markdown missing correct created_at aligned with started_at")
	}
	if !strings.Contains(cardMD, "participants: [巧先生, 枫]") {
		t.Errorf("card markdown missing correct participants")
	}
	if !strings.Contains(cardMD, expectedCoalesced) {
		t.Errorf("card markdown missing coalesced dialogue body")
	}
}

func TestDeriveFallbackTitle(t *testing.T) {
	parsed, err := Parse(exactWeChatSample)
	if err != nil {
		t.Fatalf("unexpected parse error: %v", err)
	}

	fallbackTitle := DeriveFallbackTitle(parsed.Coalesced, parsed.Participants)
	if fallbackTitle == "" {
		t.Fatalf("expected non-empty fallback title")
	}
	if !strings.HasPrefix(fallbackTitle, "巧先生与枫:") {
		t.Errorf("expected fallback title to contain speaker prefix, got %q", fallbackTitle)
	}
}

func TestDeriveTitleWithoutLLMKey(t *testing.T) {
	parsed, err := Parse(exactWeChatSample)
	if err != nil {
		t.Fatalf("unexpected parse error: %v", err)
	}

	res, err := DeriveTitle(context.Background(), module.LLMConfig{}, parsed.Coalesced, parsed.Participants)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.Title == "" {
		t.Errorf("expected non-empty title")
	}
	if len(res.Participants) != 2 {
		t.Errorf("expected 2 participants, got %v", res.Participants)
	}
}

func TestIsChatLogNonChat(t *testing.T) {
	nonChatCases := []string{
		"Hello world, this is a plain note.",
		"func main() {\n\tfmt.Println(\"test\")\n}",
		"# Heading\nSome markdown paragraphs without any chat structure.",
	}
	for _, nc := range nonChatCases {
		if IsChatLog(nc) {
			t.Errorf("expected IsChatLog(%q) to be false", nc)
		}
	}
}

func TestSingleMessageDialogue(t *testing.T) {
	single := `Alice
2026-10-14 10:00
Only one message here.`
	parsed, err := Parse(single)
	if err != nil {
		t.Fatalf("unexpected parse error: %v", err)
	}
	if len(parsed.Messages) != 1 {
		t.Fatalf("expected 1 message, got %d", len(parsed.Messages))
	}
	if parsed.StartedAt == nil || parsed.EndedAt == nil {
		t.Fatalf("expected timestamps to be lifted")
	}
	if !parsed.StartedAt.Equal(*parsed.EndedAt) {
		t.Errorf("single message started_at and ended_at should match")
	}
}

func TestColonDialogueFormat(t *testing.T) {
	input := `Alice:
Hello there!

Bob:
Hi Alice!

Alice:
How is Finnegan going?`
	parsed, err := Parse(input)
	if err != nil {
		t.Fatalf("unexpected parse error: %v", err)
	}
	if len(parsed.Messages) != 3 {
		t.Fatalf("expected 3 messages, got %d", len(parsed.Messages))
	}
	if len(parsed.Participants) != 2 || parsed.Participants[0] != "Alice" || parsed.Participants[1] != "Bob" {
		t.Errorf("unexpected participants: %v", parsed.Participants)
	}
}

func TestFormatCardMarkdownAndParseWithCoreMarkdown(t *testing.T) {
	parsed, err := Parse(exactWeChatSample)
	if err != nil {
		t.Fatalf("unexpected parse error: %v", err)
	}

	docID := "01JAX9K2ABCDEF"
	title := "吐槽 Argon 额度消耗与大模型说话语气"
	cardMD := FormatCardMarkdown(docID, title, parsed)

	doc, err := markdown.Parse("test.md", []byte(cardMD), time.Time{})
	if err != nil {
		t.Fatalf("markdown.Parse error: %v", err)
	}

	if doc.ID != docID {
		t.Errorf("expected doc.ID %s, got %s", docID, doc.ID)
	}
	if doc.Title != title {
		t.Errorf("expected doc.Title %s, got %s", title, doc.Title)
	}
	if doc.Meta["type"] != "chat_log" {
		t.Errorf("expected doc.Meta['type'] 'chat_log', got %v", doc.Meta["type"])
	}
	if doc.CreatedAt.Format(time.RFC3339) != "2026-10-14T08:15:00+08:00" {
		t.Errorf("expected doc.CreatedAt 2026-10-14T08:15:00+08:00, got %s", doc.CreatedAt.Format(time.RFC3339))
	}
	if strings.TrimSpace(doc.Body) != strings.TrimSpace(parsed.Coalesced) {
		t.Errorf("doc.Body does not match parsed.Coalesced")
	}
}

func TestFormatCardMarkdownWithImages(t *testing.T) {
	parsed, err := Parse(exactWeChatSample)
	if err != nil {
		t.Fatalf("unexpected parse error: %v", err)
	}

	docID := "01JAX9K2IMGTEST"
	title := "对白测试图片"
	imgs := []string{"1728863456789-img1.png", "1728863456790-img2.jpg"}

	// Test passing images as variadic arg
	cardMD := FormatCardMarkdown(docID, title, parsed, imgs)
	if !strings.Contains(cardMD, "images:\n  - 1728863456789-img1.png\n  - 1728863456790-img2.jpg") {
		t.Fatalf("expected cardMD to contain images frontmatter, got:\n%s", cardMD)
	}

	// Test via parsed.Images
	parsed.Images = imgs
	cardMD2 := FormatCardMarkdown(docID, title, parsed)
	if !strings.Contains(cardMD2, "images:\n  - 1728863456789-img1.png\n  - 1728863456790-img2.jpg") {
		t.Fatalf("expected cardMD2 to contain images frontmatter from parsed.Images, got:\n%s", cardMD2)
	}
}

func TestExtractHashTitle_Head(t *testing.T) {
	input := `# 项目复盘讨论
巧先生
2026年10月14日  8:15
体验下 看看有什么区别~~

枫
2026年10月14日  9:08
说话语气正常了hhhh
`
	if !IsChatLog(input) {
		t.Fatalf("expected IsChatLog to be true with header # Title")
	}

	parsed, err := Parse(input)
	if err != nil {
		t.Fatalf("Parse error: %v", err)
	}

	if parsed.UserTitle != "项目复盘讨论" {
		t.Errorf("expected UserTitle '项目复盘讨论', got %q", parsed.UserTitle)
	}

	if strings.Contains(parsed.Coalesced, "项目复盘讨论") || strings.Contains(parsed.Coalesced, "#") {
		t.Errorf("parsed.Coalesced should not contain title line: %s", parsed.Coalesced)
	}

	if len(parsed.Messages) != 2 {
		t.Errorf("expected 2 messages, got %d", len(parsed.Messages))
	}

	cardMD := FormatCardMarkdown("01TESTDOCID", "", parsed)
	if !strings.Contains(cardMD, "title: \"项目复盘讨论\"") && !strings.Contains(cardMD, "title: 项目复盘讨论") {
		t.Errorf("FormatCardMarkdown should use parsed.UserTitle when title is empty:\n%s", cardMD)
	}
}

func TestExtractHashTitle_Tail(t *testing.T) {
	input := `巧先生
2026年10月14日  8:15
体验下 看看有什么区别~~

枫
2026年10月14日  9:08
说话语气正常了hhhh

# 尾部项目总结
`
	if !IsChatLog(input) {
		t.Fatalf("expected IsChatLog to be true with footer # Title")
	}

	parsed, err := Parse(input)
	if err != nil {
		t.Fatalf("Parse error: %v", err)
	}

	if parsed.UserTitle != "尾部项目总结" {
		t.Errorf("expected UserTitle '尾部项目总结', got %q", parsed.UserTitle)
	}

	if strings.Contains(parsed.Coalesced, "尾部项目总结") || strings.Contains(parsed.Coalesced, "#") {
		t.Errorf("parsed.Coalesced should not contain title line: %s", parsed.Coalesced)
	}

	if len(parsed.Messages) != 2 {
		t.Errorf("expected 2 messages, got %d", len(parsed.Messages))
	}
}

func TestExtractHashTitle_RelaxedAndMiddleHash(t *testing.T) {
	// 1. Title without space (#Redis启动) is now supported under relaxed parsing
	inputNoSpace := `#Redis启动
巧先生:
你好

枫:
早上好
`
	parsed, err := Parse(inputNoSpace)
	if err != nil {
		t.Fatalf("Parse error: %v", err)
	}
	if parsed.UserTitle != "Redis启动" {
		t.Errorf("expected UserTitle 'Redis启动' for #Redis启动, got %q", parsed.UserTitle)
	}

	// 2. Title with leading indentation (   # 标题)
	inputIndented := `   # 标题
巧先生:
你好

枫:
早上好
`
	parsedIndented, err := Parse(inputIndented)
	if err != nil {
		t.Fatalf("Parse error: %v", err)
	}
	if parsedIndented.UserTitle != "标题" {
		t.Errorf("expected UserTitle '标题' for indented # 标题, got %q", parsedIndented.UserTitle)
	}

	// 3. Middle # should remain in dialogue content
	inputMiddle := `巧先生:
你好
# 这里是对话内部的一句话

枫:
知道了
`
	parsedMiddle, err := Parse(inputMiddle)
	if err != nil {
		t.Fatalf("Parse error: %v", err)
	}
	if parsedMiddle.UserTitle != "" {
		t.Errorf("expected empty UserTitle when # is in middle, got %q", parsedMiddle.UserTitle)
	}
	if !strings.Contains(parsedMiddle.Coalesced, "# 这里是对话内部的一句话") {
		t.Errorf("expected middle # line to remain in Coalesced content")
	}
}
