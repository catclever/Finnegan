import { describe, it } from 'node:test';
import assert from 'node:assert/strict';
import {
  collectAllAvailableTags,
  filterMatchedTags,
  detectTagTriggerAtCursor,
  applyTagSelectionToText,
} from './tagAutocomplete.ts';

describe('Tag Autocomplete Engine', () => {
  it('collectAllAvailableTags extracts and deduplicates tags from cards and localVaultPool', () => {
    const cards: any[] = [
      { id: '1', tags: ['ideas', 'design*'] },
      { id: '2', keywords: { dimension: ['system', 'design*'] } },
    ];
    const localVault: any[] = [
      { id: '3', tags: ['architecture', 'ideas'] },
    ];

    const tags = collectAllAvailableTags(cards, localVault);
    assert.deepEqual(tags, ['architecture', 'design', 'ideas', 'system']);
  });

  it('filterMatchedTags returns top N matches or all up to limit when query is empty', () => {
    const allTags = ['apple', 'application', 'banana', 'cherry', 'grape'];
    const emptyQueryMatches = filterMatchedTags(allTags, '', 3);
    assert.deepEqual(emptyQueryMatches, ['apple', 'application', 'banana']);

    const queryMatches = filterMatchedTags(allTags, 'app', 10);
    assert.deepEqual(queryMatches, ['apple', 'application']);
  });

  it('detectTagTriggerAtCursor detects hashtag triggers at cursor', () => {
    // 1. Trigger at start of line
    const r1 = detectTagTriggerAtCursor('#ide', 4);
    assert.equal(r1.isTriggered, true);
    assert.equal(r1.tagQuery, 'ide');
    assert.equal(r1.tagMatchStart, 0);

    // 2. Trigger with space before hash
    const r2 = detectTagTriggerAtCursor('Hello #sys', 10);
    assert.equal(r2.isTriggered, true);
    assert.equal(r2.tagQuery, 'sys');
    assert.equal(r2.tagMatchStart, 6);

    // 3. No trigger if cursor is not following a hash token
    const r3 = detectTagTriggerAtCursor('Hello world', 11);
    assert.equal(r3.isTriggered, false);

    // 4. No trigger if hash is attached to word without preceding whitespace: abc#def
    const r4 = detectTagTriggerAtCursor('abc#def', 7);
    assert.equal(r4.isTriggered, false);
  });

  it('applyTagSelectionToText formats tag and advances cursor past trailing space', () => {
    const text = 'Exploring #des today';
    const tagMatchStart = 10;
    const cursorPos = 14;
    const { newText, newCursor } = applyTagSelectionToText(text, tagMatchStart, cursorPos, 'design-system');
    assert.equal(newText, 'Exploring #design-system  today');
    assert.equal(newCursor, ('Exploring #design-system ').length);

    // Tag with space converted to hyphen
    const res = applyTagSelectionToText('Notes #mac', 6, 10, 'machine learning');
    assert.equal(res.newText, 'Notes #machine-learning ');
    assert.equal(res.newCursor, ('Notes #machine-learning ').length);
  });
});
