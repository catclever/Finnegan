import { describe, it } from 'node:test';
import assert from 'node:assert/strict';
// @ts-ignore
import { extractHashTitle, isChatLogFormat, parseDialogue } from './chatlog.ts';

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
});
