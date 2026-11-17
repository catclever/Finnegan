/**
 * Finnegan Cosmic Stage - Chat Log Domain Utilities
 * Task: chat_log_card_frontend_032
 * 
 * Provides pure parsing, format detection, and time formatting for chat logs.
 */

import type { DialogueTurn } from '../types/card';
import { Marked } from 'marked';
import DOMPurify from 'dompurify';

const reWeChatDate = /^(\d{4})年(\d{1,2})月(\d{1,2})日\s+(\d{1,2}):(\d{2})(?::(\d{2}))?$/;
const reStandardDate = /^(\d{4})[-/](\d{1,2})[-/](\d{1,2})\s+(\d{1,2}):(\d{2})(?::(\d{2}))?$/;
const reMonthDayTime = /^(\d{1,2})月(\d{1,2})日\s+(\d{1,2}):(\d{2})(?::(\d{2}))?$/;
const reSingleLineWeChat = /^([^\n\d]+?)\s+(\d{4}年\d{1,2}月\d{1,2}日\s+\d{1,2}:\d{2}(?::\d{2})?)$/;
const reSingleLineStandard = /^([^\n\d]+?)\s+(\d{4}[-/]\d{1,2}[-/]\d{1,2}\s+\d{1,2}:\d{2}(?::\d{2})?)$/;
const reBracketTimePrefix = /^\[(\d{4}[-/]\d{1,2}[-/]\d{1,2}\s+\d{1,2}:\d{2}(?::\d{2})?)\]\s*(.+?):?$/;
const reBracketTimeSuffix = /^(.+?)\s*\[(\d{4}[-/]\d{1,2}[-/]\d{1,2}\s+\d{1,2}:\d{2}(?::\d{2})?)\]:?$/;
const reColonSpeaker = /^([^\n:：]{1,60})[:：]\s*$/;
const reHashTitle = /^\s*#\s*(.+)$/;

/**
 * Scans the first or last non-empty line of text for standard Markdown H1 syntax ("# <title>").
 * If found, returns the extracted title and dialogue text with that title line removed.
 */
export function extractHashTitle(text: string): { title: string; cleanText: string } {
  if (!text) return { title: '', cleanText: '' };
  const clean = text.replace(/\r\n/g, '\n').replace(/\r/g, '\n');
  const lines = clean.split('\n');

  let firstIdx = -1;
  for (let i = 0; i < lines.length; i++) {
    if (lines[i].trim() !== '') {
      firstIdx = i;
      break;
    }
  }

  let lastIdx = -1;
  for (let i = lines.length - 1; i >= 0; i--) {
    if (lines[i].trim() !== '') {
      lastIdx = i;
      break;
    }
  }

  if (firstIdx !== -1) {
    const firstLine = lines[firstIdx].trim();
    const match = firstLine.match(reHashTitle);
    if (match && match[1]?.trim()) {
      const title = match[1].trim();
      const newLines = [...lines.slice(0, firstIdx), ...lines.slice(firstIdx + 1)];
      return { title, cleanText: newLines.join('\n') };
    }
  }

  if (lastIdx !== -1 && lastIdx !== firstIdx) {
    const lastLine = lines[lastIdx].trim();
    const match = lastLine.match(reHashTitle);
    if (match && match[1]?.trim()) {
      const title = match[1].trim();
      const newLines = [...lines.slice(0, lastIdx), ...lines.slice(lastIdx + 1)];
      return { title, cleanText: newLines.join('\n') };
    }
  }

  return { title: '', cleanText: text };
}

function isTimestamp(s: string): boolean {
  const t = s.trim();
  if (!t) return false;
  if (reWeChatDate.test(t) || reStandardDate.test(t) || reMonthDayTime.test(t)) return true;
  if (!isNaN(Date.parse(t)) && (t.includes('T') || t.includes('-') || t.includes('/'))) return true;
  return false;
}

/**
 * Returns true if the raw input text represents a dialogue or chat log format.
 */
export function isChatLogFormat(text: string): boolean {
  if (!text) return false;
  const { cleanText } = extractHashTitle(text);
  const clean = cleanText.replace(/\r\n/g, '\n').replace(/\r/g, '\n');
  const lines = clean.split('\n');
  if (lines.length < 2) return false;

  let timestampMatches = 0;
  let colonMatches = 0;
  for (let i = 0; i < lines.length; i++) {
    const line = lines[i].trim();
    if (!line) continue;

    // Check WeChat 2-line header pattern
    if (isTimestamp(line) && i > 0) {
      const prev = lines[i - 1].trim();
      if (prev && prev.length <= 60 && !prev.startsWith('#') && !prev.startsWith('---')) {
        timestampMatches++;
      }
    }

    // Check single-line timestamp header
    if (
      reSingleLineWeChat.test(line) ||
      reSingleLineStandard.test(line) ||
      reBracketTimePrefix.test(line) ||
      reBracketTimeSuffix.test(line)
    ) {
      timestampMatches++;
    }

    // Check colon speaker pattern
    if (reColonSpeaker.test(line)) {
      colonMatches++;
    }
  }

  if (timestampMatches >= 1) return true;
  return colonMatches >= 2;
}

const NON_SPEAKER_WORDS = new Set([
  '想一下', '注意', '提示', '例如', '比如', '总结', '分析', '说明', '备注', 'ps', 'todo', '参考', '补充', '重点'
]);

function isSpeakerCandidate(cand: string, rawLine: string, validParticipants?: string[] | null): boolean {
  if (!cand) return false;
  const cleanCand = cand.trim();
  const lower = cleanCand.toLowerCase();
  if (validParticipants && validParticipants.length > 0) {
    return validParticipants.includes(lower);
  }
  // Reject markdown list items: "1. ", "2) ", "* ", "- ", "+ "
  if (/^\s*(?:\d+[\.\)]|[\*\-\+])\s+/.test(rawLine)) return false;
  // Reject headings, blockquotes, tables
  if (/^\s*#{1,6}\s+/.test(rawLine)) return false;
  if (/^\s*>/.test(rawLine)) return false;
  if (/^\s*\|/.test(rawLine)) return false;
  // Reject URLs
  if (cleanCand.startsWith('http://') || cleanCand.startsWith('https://')) return false;
  // Reject inline code
  if (cleanCand.includes('`')) return false;
  // Reject sentence punctuation
  if (/[，。！？；、“”‘’（）《》【】]/.test(cleanCand)) return false;
  // Reject overly long candidates (names are almost never > 20 chars)
  if (cleanCand.length > 20) return false;
  // Reject common prose markers
  if (NON_SPEAKER_WORDS.has(lower)) return false;
  return true;
}

/**
 * Parses dialogue text into structured speaker turns.
 * When knownParticipants is provided (e.g. ['User', 'Gemini']), turn boundaries strictly match those speakers,
 * preventing embedded colons (like list items or prompt explanations) from splitting the conversation.
 */
export function parseDialogue(text: string, knownParticipants?: string[]): DialogueTurn[] {
  if (!text) return [];
  const { cleanText } = extractHashTitle(text);
  const clean = cleanText.replace(/\r\n/g, '\n').replace(/\r/g, '\n');
  const lines = clean.split('\n');

  const turns: DialogueTurn[] = [];
  let currentTurn: DialogueTurn | null = null;

  const colonRegex = /^([^\n:：]{1,60})[:：]\s*(.*)$/;
  const timestampRegex = /^(\d{4}[年\-\/].+|\d{1,2}月\d{1,2}日.+|\d{1,2}:\d{2}(?::\d{2})?)/;

  const validParticipants = Array.isArray(knownParticipants) && knownParticipants.length > 0
    ? knownParticipants.map(p => p.trim().toLowerCase()).filter(Boolean)
    : null;

  for (let i = 0; i < lines.length; i++) {
    const rawLine = lines[i];
    const line = rawLine.trim();

    if (!line) {
      continue;
    }

    // Check if this line is a timestamp line following a speaker name (WeChat raw style)
    if (i > 0 && timestampRegex.test(line) && currentTurn && currentTurn.lines.length === 0) {
      continue;
    }

    // Check if line is Speaker: or Speaker: Content
    const colonMatch = line.match(colonRegex);
    if (colonMatch && isSpeakerCandidate(colonMatch[1].trim(), rawLine, validParticipants)) {
      const speaker = colonMatch[1].trim();
      const firstLine = colonMatch[2].trim();

      currentTurn = {
        speaker,
        lines: firstLine ? [firstLine] : []
      };
      turns.push(currentTurn);
      continue;
    }

    // Check if this line is a WeChat style standalone speaker (next line is timestamp)
    if (
      i + 1 < lines.length &&
      timestampRegex.test(lines[i + 1].trim()) &&
      isSpeakerCandidate(line, rawLine, validParticipants) &&
      !line.startsWith('#') &&
      !line.startsWith('---')
    ) {
      currentTurn = {
        speaker: line,
        lines: []
      };
      turns.push(currentTurn);
      continue;
    }

    // Regular dialogue line
    if (currentTurn) {
      currentTurn.lines.push(line);
    } else {
      currentTurn = {
        speaker: '',
        lines: [line]
      };
      turns.push(currentTurn);
    }
  }

  return turns;
}

/**
 * Formats start and end timestamps into a screenplay dateline range.
 */
export function formatChatLogTime(startedAt?: string, endedAt?: string): string {
  if (!startedAt && !endedAt) return '';
  const dStart = startedAt ? new Date(startedAt) : null;
  const dEnd = endedAt ? new Date(endedAt) : null;

  if (dStart && isNaN(dStart.getTime())) return '';
  if (dEnd && isNaN(dEnd.getTime())) return '';

  const pad = (n: number) => String(n).padStart(2, '0');

  if (dStart && dEnd) {
    const sameDay =
      dStart.getFullYear() === dEnd.getFullYear() &&
      dStart.getMonth() === dEnd.getMonth() &&
      dStart.getDate() === dEnd.getDate();

    const startH = pad(dStart.getHours());
    const startM = pad(dStart.getMinutes());
    const endH = pad(dEnd.getHours());
    const endM = pad(dEnd.getMinutes());
    const m = dStart.getMonth() + 1;
    const d = dStart.getDate();

    if (sameDay) {
      return `${m}/${d} ${startH}:${startM} – ${endH}:${endM}`;
    } else {
      const endMonth = dEnd.getMonth() + 1;
      const endDay = dEnd.getDate();
      return `${m}/${d} ${startH}:${startM} – ${endMonth}/${endDay} ${endH}:${endM}`;
    }
  }

  const d = dStart || dEnd;
  if (!d) return '';
  const m = d.getMonth() + 1;
  const day = d.getDate();
  const h = pad(d.getHours());
  const min = pad(d.getMinutes());
  return `${m}/${day} ${h}:${min}`;
}

/**
 * Creates an inline-only Marked parser that supports inline markdown syntax (bold, italic, code, del, links)
 * and optional @theme mentions, avoiding block-level headers/tables from breaking screenplay layout.
 */
export function createInlineDialogueParser(activeThreads?: Array<{ id: string; title: string }>): Marked {
  const m = new Marked();
  m.use({
    extensions: [{
      name: 'themeMention',
      level: 'inline',
      start(src: string) { return src.indexOf('@'); },
      tokenizer(src: string) {
        if (activeThreads && activeThreads.length > 0) {
          for (const th of activeThreads) {
            if (!th.title) continue;
            const prefix = '@' + th.title;
            if (src.startsWith(prefix)) {
              return {
                type: 'themeMention',
                raw: prefix,
                text: th.title,
              };
            }
          }
        }
        const match = /^@([^\s@#,\n<>:，。！？]+)/u.exec(src);
        if (match) {
          return {
            type: 'themeMention',
            raw: match[0],
            text: match[1],
          };
        }
      },
      renderer(token: any) {
        return `<span class="theme-mention">@${token.text}</span>`;
      }
    }]
  });
  return m;
}

/**
 * Heuristically corrects common LLM markdown formatting flaws like:
 * - Quotes followed by bold delimiters: “xxx”** -> “**xxx**”
 * - Unbalanced bold delimiters within a single dialogue line
 */
export function fixMalformedBold(text: string): string {
  if (!text) return '';
  let s = text;
  // Turn “xxx”** into “**xxx**”
  s = s.replace(/(?<!\*\*)“([^”\n]+)”\*\*/g, '“**$1**”');
  s = s.replace(/(?<!\*\*)「([^」\n]+)」\*\*/g, '「**$1**」');
  s = s.replace(/(?<!\*\*)《([^》\n]+)》\*\*/g, '《**$1**》');
  // Turn **“xxx” into “**xxx**” if unclosed
  s = s.replace(/\*\*“([^”\n]+)”(?!\*\*)/g, '“**$1**”');

  // Balance odd count of **
  const matches = s.match(/\*\*/g);
  if (matches && matches.length % 2 !== 0) {
    if (/[。！？；]$/.test(s)) {
      s = s.slice(0, -1) + '**' + s.slice(-1);
    } else {
      s = s + '**';
    }
  }
  return s;
}

/**
 * Renders a single line of dialogue into sanitized inline HTML.
 */
export function renderInlineDialogue(text: string, parser?: Marked): string {
  if (!text) return '';
  const p = parser || createInlineDialogueParser();
  try {
    const cleaned = fixMalformedBold(text);
    const parsed = p.parseInline(cleaned) as string;
    return typeof DOMPurify !== 'undefined' && typeof (DOMPurify as any).sanitize === 'function'
      ? (DOMPurify as any).sanitize(parsed)
      : parsed;
  } catch {
    return text;
  }
}

