import { describe, it } from 'node:test';
import assert from 'node:assert/strict';
// @ts-ignore
import {
  normalizeTrigger,
  getCommandIcon,
  filterAndRankCommands,
  resolveCommandInput,
  matchExactCommand,
} from './commandEngine.ts';
import type { CommandDef } from '../types/card';

const sampleCommands: CommandDef[] = [
  {
    id: 'bugu:radar',
    name: '布谷深网雷达',
    trigger: 'bugu',
    description: '吾道不孤',
    namespace: 'bugu',
    isPrimary: true,
  },
  {
    id: 'atproto:echo',
    name: 'Bluesky 灵感回声',
    trigger: 'bsky',
    description: 'Bluesky 灵感回声',
    namespace: 'bsky',
    isPrimary: true,
  },
  {
    id: 'atproto:publish',
    name: '发布到 Bluesky',
    trigger: 'bsky-push',
    description: '发布到 Bluesky',
    namespace: 'bsky',
    isPrimary: false,
    subCommand: 'push',
  },
  {
    id: 'system:search',
    name: '全局搜索',
    trigger: 'search',
    description: '搜索所有本地卡片',
  },
];

describe('Finnegan Module Command Architecture & Command Engine', () => {
  describe('normalizeTrigger', () => {
    it('removes leading slash and trims', () => {
      assert.equal(normalizeTrigger('/bsky'), 'bsky');
      assert.equal(normalizeTrigger('//bugu '), 'bugu');
      assert.equal(normalizeTrigger('bsky-push'), 'bsky-push');
    });
  });

  describe('getCommandIcon', () => {
    it('returns butterfly for bsky module commands', () => {
      assert.equal(getCommandIcon(sampleCommands[1]), '🦋'); // bsky
      assert.equal(getCommandIcon(sampleCommands[2]), '🦋'); // bsky-push
    });

    it('returns dove for bugu module command', () => {
      assert.equal(getCommandIcon(sampleCommands[0]), '🕊️'); // bugu
    });

    it('returns telescope for other commands', () => {
      assert.equal(getCommandIcon(sampleCommands[3]), '🔭'); // search
      assert.equal(getCommandIcon(null), '🔭');
    });
  });

  describe('filterAndRankCommands', () => {
    it('ranks exact trigger match first', () => {
      const results = filterAndRankCommands(sampleCommands, 'bsky');
      assert.equal(results.length, 2);
      assert.equal(results[0].trigger, 'bsky');
      assert.equal(results[1].trigger, 'bsky-push');
    });

    it('ranks primary command before sub-commands when namespace is matched', () => {
      const results = filterAndRankCommands(sampleCommands, 'bsky');
      assert.equal(results[0].isPrimary, true);
      assert.equal(results[1].isPrimary, false);
      assert.equal(results[1].subCommand, 'push');
    });

    it('matches by subCommand keyword', () => {
      const results = filterAndRankCommands(sampleCommands, 'push');
      assert.equal(results.length, 1);
      assert.equal(results[0].id, 'atproto:publish');
      assert.equal(results[0].trigger, 'bsky-push');
    });

    it('returns all commands when query is empty', () => {
      const results = filterAndRankCommands(sampleCommands, '');
      assert.equal(results.length, 4);
    });

    it('matches by Chinese description or name', () => {
      const results = filterAndRankCommands(sampleCommands, '雷达');
      assert.equal(results.length, 1);
      assert.equal(results[0].trigger, 'bugu');
    });
  });

  describe('resolveCommandInput', () => {
    it('resolves standard /trigger with remainder text', () => {
      const res = resolveCommandInput('/bugu my search term', sampleCommands);
      assert.ok(res);
      assert.equal(res.command.trigger, 'bugu');
      assert.equal(res.restText, 'my search term');
    });

    it('resolves direct sub-command /bsky-push with remainder text', () => {
      const res = resolveCommandInput('/bsky-push my card title', sampleCommands);
      assert.ok(res);
      assert.equal(res.command.trigger, 'bsky-push');
      assert.equal(res.command.subCommand, 'push');
      assert.equal(res.restText, 'my card title');
    });

    it('resolves space-separated namespace and subcommand: /bsky push my card', () => {
      const res = resolveCommandInput('/bsky push my card', sampleCommands);
      assert.ok(res);
      assert.equal(res.command.id, 'atproto:publish');
      assert.equal(res.command.trigger, 'bsky-push');
      assert.equal(res.restText, 'my card');
    });

    it('resolves space-separated namespace and subcommand without rest text: /bsky push', () => {
      const res = resolveCommandInput('/bsky push', sampleCommands);
      assert.ok(res);
      assert.equal(res.command.id, 'atproto:publish');
      assert.equal(res.command.trigger, 'bsky-push');
      assert.equal(res.restText, '');
    });

    it('resolves primary command when second word is not a sub-command: /bsky note', () => {
      const res = resolveCommandInput('/bsky note', sampleCommands);
      assert.ok(res);
      assert.equal(res.command.trigger, 'bsky');
      assert.equal(res.command.isPrimary, true);
      assert.equal(res.restText, 'note');
    });

    it('returns null when there are no spaces in input', () => {
      const res = resolveCommandInput('/bsky', sampleCommands);
      assert.equal(res, null);
    });
  });

  describe('matchExactCommand', () => {
    it('matches direct trigger without slash', () => {
      const cmd = matchExactCommand('bsky', sampleCommands);
      assert.ok(cmd);
      assert.equal(cmd.id, 'atproto:echo');
    });

    it('matches direct trigger with slash', () => {
      const cmd = matchExactCommand('/bsky-push', sampleCommands);
      assert.ok(cmd);
      assert.equal(cmd.id, 'atproto:publish');
    });

    it('matches space-separated "namespace subcommand"', () => {
      const cmd = matchExactCommand('/bsky push', sampleCommands);
      assert.ok(cmd);
      assert.equal(cmd.id, 'atproto:publish');
      assert.equal(cmd.trigger, 'bsky-push');
    });

    it('returns null for unknown command', () => {
      const cmd = matchExactCommand('/unknown', sampleCommands);
      assert.equal(cmd, null);
    });
  });
});
