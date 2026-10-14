/**
 * Finnegan Cosmic Stage - Chat Log Domain Utilities
 * Task: chat_log_card_frontend_032
 * 
 * Provides pure parsing, format detection, and time formatting for chat logs.
 */

import type { DialogueTurn } from '../types/card';

const reWeChatDate = /^(\d{4})年(\d{1,2})月(\d{1,2})日\s+(\d{1,2}):(\d{2})(?::(\d{2}))?$/;
const reStandardDate = /^(\d{4})[-/](\d{1,2})[-/](\d{1,2})\s+(\d{1,2}):(\d{2})(?::(\d{2}))?$/;
const reMonthDayTime = /^(\d{1,2})月(\d{1,2})日\s+(\d{1,2}):(\d{2})(?::(\d{2}))?$/;
const reSingleLineWeChat = /^([^\n\d]+?)\s+(\d{4}年\d{1,2}月\d{1,2}日\s+\d{1,2}:\d{2}(?::\d{2})?)$/;
const reSingleLineStandard = /^([^\n\d]+?)\s+(\d{4}[-/]\d{1,2}[-/]\d{1,2}\s+\d{1,2}:\d{2}(?::\d{2})?)$/;
const reBracketTimePrefix = /^\[(\d{4}[-/]\d{1,2}[-/]\d{1,2}\s+\d{1,2}:\d{2}(?::\d{2})?)\]\s*(.+?):?$/;
const reBracketTimeSuffix = /^(.+?)\s*\[(\d{4}[-/]\d{1,2}[-/]\d{1,2}\s+\d{1,2}:\d{2}(?::\d{2})?)\]:?$/;
const reColonSpeaker = /^([^\n:：]{1,60})[:：]\s*$/;

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
  const clean = text.replace(/\r\n/g, '\n').replace(/\r/g, '\n');
  const lines = clean.split('\n');
  if (lines.length < 2) return false;

  let matchCount = 0;
  for (let i = 0; i < lines.length; i++) {
    const line = lines[i].trim();
    if (!line) continue;

    // Check WeChat 2-line header pattern
    if (isTimestamp(line) && i > 0) {
      const prev = lines[i - 1].trim();
      if (prev && prev.length <= 60 && !prev.startsWith('#') && !prev.startsWith('---')) {
        matchCount++;
      }
    }

    // Check single-line timestamp header
    if (
      reSingleLineWeChat.test(line) ||
      reSingleLineStandard.test(line) ||
      reBracketTimePrefix.test(line) ||
      reBracketTimeSuffix.test(line)
    ) {
      matchCount++;
    }

    // Check colon speaker pattern
    if (reColonSpeaker.test(line)) {
      matchCount++;
    }
  }

  return matchCount >= 1;
}

/**
 * Parses dialogue text into structured speaker turns.
 */
export function parseDialogue(text: string): DialogueTurn[] {
  if (!text) return [];
  const clean = text.replace(/\r\n/g, '\n').replace(/\r/g, '\n');
  const lines = clean.split('\n');

  const turns: DialogueTurn[] = [];
  let currentTurn: DialogueTurn | null = null;

  const colonRegex = /^([^\n:：]{1,60})[:：]\s*(.*)$/;
  const timestampRegex = /^(\d{4}[年\-\/].+|\d{1,2}月\d{1,2}日.+|\d{1,2}:\d{2}(?::\d{2})?)/;

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
    if (colonMatch && !line.startsWith('http://') && !line.startsWith('https://')) {
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
      line.length <= 60 &&
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
