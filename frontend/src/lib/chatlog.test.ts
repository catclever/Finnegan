import { describe, it } from 'node:test';
import assert from 'node:assert/strict';
// @ts-ignore
import { extractHashTitle, isChatLogFormat, parseDialogue, createInlineDialogueParser, renderInlineDialogue, fixMalformedBold } from './chatlog.ts';

describe('ChatLog Parser & Hash Title Engine', () => {
  const sampleDialogue = `巧先生
2026年10月14日  8:15
体验下 看看有什么区别~~

枫
2026年10月14日  9:08
说话语气正常了hhhh`;

  describe('extractHashTitle', () => {
    it('extracts title from head (# Title on first line)', () => {
      const input = `# 架构复盘讨论
${sampleDialogue}`;
      const { title, cleanText } = extractHashTitle(input);
      assert.equal(title, '架构复盘讨论');
      assert.equal(cleanText.includes('# 架构复盘讨论'), false);
      assert.equal(cleanText.includes('巧先生'), true);
      assert.equal(cleanText.includes('说话语气正常了hhhh'), true);
    });

    it('extracts title with leading empty lines', () => {
      const input = `

  # 项目进度汇报  

${sampleDialogue}`;
      const { title, cleanText } = extractHashTitle(input);
      assert.equal(title, '项目进度汇报');
      assert.equal(cleanText.includes('项目进度汇报'), false);
      assert.equal(cleanText.includes('巧先生'), true);
    });

    it('extracts title from tail (# Title on last line)', () => {
      const input = `${sampleDialogue}

# 尾部讨论纪要
`;
      const { title, cleanText } = extractHashTitle(input);
      assert.equal(title, '尾部讨论纪要');
      assert.equal(cleanText.includes('# 尾部讨论纪要'), false);
      assert.equal(cleanText.includes('巧先生'), true);
      assert.equal(cleanText.includes('说话语气正常了hhhh'), true);
    });

    it('extracts title without space (#Title or #Redis启动)', () => {
      const input = `#Redis启动
${sampleDialogue}`;
      const { title, cleanText } = extractHashTitle(input);
      assert.equal(title, 'Redis启动');
      assert.equal(cleanText.includes('#Redis启动'), false);
      assert.equal(cleanText.includes('巧先生'), true);
    });

    it('does not extract middle # line as title', () => {
      const input = `巧先生
2026年10月14日  8:15
# 这里是对话内部的一句话
枫
2026年10月14日  9:08
收到`;
      const { title, cleanText } = extractHashTitle(input);
      assert.equal(title, '');
      assert.equal(cleanText, input);
    });
  });

  describe('isChatLogFormat with hash title', () => {
    it('recognizes dialogue with head hash title', () => {
      const input = `# 架构复盘讨论\n${sampleDialogue}`;
      assert.equal(isChatLogFormat(input), true);
    });

    it('recognizes dialogue with tail hash title', () => {
      const input = `${sampleDialogue}\n\n# 架构复盘讨论`;
      assert.equal(isChatLogFormat(input), true);
    });

    it('rejects single colon headers without timestamps (common notes)', () => {
      assert.equal(isChatLogFormat('注意:\n明天下午开会'), false);
      assert.equal(isChatLogFormat('提示：\n这是一个普通笔记'), false);
      assert.equal(isChatLogFormat('TODO:\n1. 修复bug\n2. 提交代码'), false);
    });

    it('accepts multi-speaker colon dialogues without timestamps', () => {
      const multi = `张三:\n你好\n李四:\n你好呀`;
      assert.equal(isChatLogFormat(multi), true);
    });
  });

  describe('parseDialogue with hash title', () => {
    it('removes head hash title so it does not become a turn', () => {
      const input = `# 架构复盘讨论\n${sampleDialogue}`;
      const turns = parseDialogue(input);
      assert.equal(turns.length, 2);
      assert.equal(turns[0].speaker, '巧先生');
      assert.equal(turns[0].lines.includes('# 架构复盘讨论'), false);
      assert.equal(turns[1].speaker, '枫');
    });

    it('removes tail hash title so it does not become a turn', () => {
      const input = `${sampleDialogue}\n\n# 架构复盘讨论`;
      const turns = parseDialogue(input);
      assert.equal(turns.length, 2);
      assert.equal(turns[0].speaker, '巧先生');
      assert.equal(turns[1].speaker, '枫');
      assert.equal(turns[1].lines.includes('# 架构复盘讨论'), false);
    });
  });

  describe('renderInlineDialogue & createInlineDialogueParser', () => {
    it('returns empty string for empty input', () => {
      assert.equal(renderInlineDialogue(''), '');
      assert.equal(renderInlineDialogue(null as any), '');
      assert.equal(renderInlineDialogue(undefined as any), '');
    });

    it('preserves plain text without changes', () => {
      assert.equal(renderInlineDialogue('今天天气真好'), '今天天气真好');
    });

    it('renders inline code, bold, italic, and strikethrough', () => {
      const input = '运行 `git status` 看看 **重要结果** 以及 *注意事项* 和 ~~过期内容~~';
      const output = renderInlineDialogue(input);
      assert.ok(output.includes('<code>git status</code>'));
      assert.ok(output.includes('<strong>重要结果</strong>'));
      assert.ok(output.includes('<em>注意事项</em>'));
      assert.ok(output.includes('<del>过期内容</del>'));
    });

    it('renders markdown links', () => {
      const input = '点击查看 [官方文档](https://example.com/docs)';
      const output = renderInlineDialogue(input);
      assert.ok(output.includes('<a href="https://example.com/docs">官方文档</a>'));
    });

    it('does not produce block headers or lists that disrupt screenplay layout', () => {
      assert.equal(renderInlineDialogue('# 一级标题'), '# 一级标题');
      assert.equal(renderInlineDialogue('### 三级标题'), '### 三级标题');
      assert.equal(renderInlineDialogue('- 列表项1'), '- 列表项1');
      assert.equal(renderInlineDialogue('> 引用内容'), '&gt; 引用内容');
    });

    it('renders mentions using themeMention extension', () => {
      const output = renderInlineDialogue('请问 @Alice 你觉得呢？');
      assert.ok(output.includes('<span class="theme-mention">@Alice</span>'));
    });

    it('prioritizes active threads for theme mentions', () => {
      const parser = createInlineDialogueParser([{ id: 'th1', title: 'Finnegan架构' }]);
      const output = renderInlineDialogue('关于 @Finnegan架构 的讨论', parser);
      assert.ok(output.includes('<span class="theme-mention">@Finnegan架构</span>'));
    });

    it('fixes and balances malformed bold delimiters from LLMs', () => {
      const raw = '这是一份“AI 研发全流程时序图”**，从**需求提出到验收上线。';
      const output = renderInlineDialogue(raw);
      assert.ok(output.includes('<strong>“AI 研发全流程时序图”</strong>') || output.includes('“<strong>AI 研发全流程时序图</strong>”'));
      assert.ok(output.includes('<strong>需求提出到验收上线</strong>'));
    });
  });

  describe('parseDialogue with participants and speaker filtering', () => {
    it('restricts turns strictly to known participants when provided', () => {
      const raw = `User:
理解一下这个图上说的内容？

Gemini:
以下是核心分析：
1. **业务方 (\`BIZ\`)**：负责最初提出业务需求。
2. **项目管理结对**：作为全局中枢。
* **状态机驱动**：两阶段提交。`;

      const turns = parseDialogue(raw, ['User', 'Gemini']);
      assert.equal(turns.length, 2);
      assert.equal(turns[0].speaker, 'User');
      assert.equal(turns[1].speaker, 'Gemini');
      // Verify list items with colons remain inside Gemini lines
      assert.ok(turns[1].lines.some(l => l.includes('1. **业务方 (`BIZ`)**：负责最初提出业务需求。')));
      assert.ok(turns[1].lines.some(l => l.includes('2. **项目管理结对**：作为全局中枢。')));
      assert.ok(turns[1].lines.some(l => l.includes('* **状态机驱动**：两阶段提交。')));
    });

    it('rejects markdown list markers and prose phrases from becoming speakers even without participants list', () => {
      const raw = `User:
你好
Gemini:
想一下：
1. **业务方 (\`BIZ\`)**：负责最初提出需求。
* **重要特性**：这是列表项。
### 核心模块：说明`;

      const turns = parseDialogue(raw);
      assert.equal(turns.length, 2);
      assert.equal(turns[0].speaker, 'User');
      assert.equal(turns[1].speaker, 'Gemini');
      assert.ok(turns[1].lines.some(l => l.includes('1. **业务方 (`BIZ`)**：负责最初提出需求。')));
    });
  });
});
