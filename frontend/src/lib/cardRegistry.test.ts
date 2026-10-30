import { describe, it, beforeEach } from 'node:test';
import assert from 'node:assert/strict';
// @ts-ignore
import { CardRegistry, createCardRegistry, normalizeCardType, isCoreCardType, getCardDefaultTitle, CORE_CARD_TYPES, SHARED_CARD_TYPES, BUILTIN_CARD_DEFINITIONS, DEFAULT_CARD_TYPE } from './cardRegistry.ts';
import type { CardTypeDefinition } from '../types/card';

describe('Card Type Registry & Morphology Decoupling Engine', () => {
  describe('Built-in Type Definitions & Core/Shared Categorization', () => {
    it('defines thought as the default core native type', () => {
      assert.equal(DEFAULT_CARD_TYPE, 'thought');
      assert.equal(BUILTIN_CARD_DEFINITIONS.thought.isDefault, true);
      assert.equal(BUILTIN_CARD_DEFINITIONS.thought.defaultTitle, 'Thought');
    });

    it('categorizes thought, note, and chat_log as zero-dependency core base types', () => {
      assert.deepEqual(CORE_CARD_TYPES, ['thought', 'note', 'chat_log']);
      assert.equal(isCoreCardType('thought'), true);
      assert.equal(isCoreCardType('note'), true);
      assert.equal(isCoreCardType('chat_log'), true);
    });

    it('categorizes babel as a generic shared shape decoupled from bugu', () => {
      assert.deepEqual(SHARED_CARD_TYPES, ['babel']);
      assert.equal(isCoreCardType('babel'), false);
      assert.equal(BUILTIN_CARD_DEFINITIONS.babel.defaultTitle, 'Babel Echo');
      assert.equal(BUILTIN_CARD_DEFINITIONS.babel.name, '巴别塔');
    });
  });

  describe('Card Type Normalization & Title Helpers', () => {
    it('normalizes empty, null, undefined, or whitespace types to thought', () => {
      assert.equal(normalizeCardType(undefined), 'thought');
      assert.equal(normalizeCardType(null), 'thought');
      assert.equal(normalizeCardType(''), 'thought');
      assert.equal(normalizeCardType('   '), 'thought');
      assert.equal(normalizeCardType('thought'), 'thought');
      assert.equal(normalizeCardType('  note  '), 'note');
      assert.equal(normalizeCardType('babel'), 'babel');
    });

    it('retrieves default titles for builtin and unknown types', () => {
      assert.equal(getCardDefaultTitle('thought'), 'Thought');
      assert.equal(getCardDefaultTitle('note'), 'Note');
      assert.equal(getCardDefaultTitle('chat_log'), 'Chat Log');
      assert.equal(getCardDefaultTitle('babel'), 'Babel Echo');
      // Unknown or empty types gracefully fallback to Thought
      assert.equal(getCardDefaultTitle('unknown_morph'), 'Thought');
      assert.equal(getCardDefaultTitle(undefined), 'Thought');
      assert.equal(getCardDefaultTitle(null), 'Thought');
      assert.equal(getCardDefaultTitle(''), 'Thought');
    });
  });

  describe('CardRegistry Dynamic Dispatch & Fallback Mechanics', () => {
    const MockThoughtBody = { name: 'ThoughtCardBody' };
    const MockNoteBody = { name: 'NoteCardBody' };
    const MockChatLogBody = { name: 'ChatLogCardBody' };
    const MockBabelBody = { name: 'BabelCardBody' };

    let registry: CardRegistry<any>;

    beforeEach(() => {
      registry = createCardRegistry(MockThoughtBody);
      registry.register('thought', MockThoughtBody);
      registry.register('note', MockNoteBody);
      registry.register('chat_log', MockChatLogBody);
      registry.register('babel', MockBabelBody);
    });

    it('retrieves registered components for core types', () => {
      assert.equal(registry.get('thought'), MockThoughtBody);
      assert.equal(registry.get('note'), MockNoteBody);
      assert.equal(registry.get('chat_log'), MockChatLogBody);
    });

    it('retrieves registered component for shared babel type', () => {
      assert.equal(registry.get('babel'), MockBabelBody);
    });

    it('gracefully falls back to ThoughtCardBody when type is missing or empty', () => {
      assert.equal(registry.get(undefined), MockThoughtBody);
      assert.equal(registry.get(null), MockThoughtBody);
      assert.equal(registry.get(''), MockThoughtBody);
      assert.equal(registry.get('   '), MockThoughtBody);
    });

    it('gracefully falls back to ThoughtCardBody when type is unknown or unregistered', () => {
      assert.equal(registry.get('unknown_type'), MockThoughtBody);
      assert.equal(registry.get('some_random_future_card'), MockThoughtBody);
    });

    it('tolerates leading and trailing whitespace in type lookups', () => {
      assert.equal(registry.get('  note  '), MockNoteBody);
      assert.equal(registry.get('  babel '), MockBabelBody);
    });

    it('supports dynamic registration of modular extension types (e.g. curio)', () => {
      const MockCurioBody = { name: 'CurioCardBody' };
      assert.equal(registry.has('curio'), false);

      registry.register('curio', MockCurioBody, {
        name: '奇物',
        description: '未来模块扩展：探索与收藏奇物卡片',
        defaultTitle: 'Curio'
      });

      assert.equal(registry.has('curio'), true);
      assert.equal(registry.get('curio'), MockCurioBody);

      const def = registry.getDefinition('curio');
      assert.equal(def.type, 'curio');
      assert.equal(def.name, '奇物');
      assert.equal(def.defaultTitle, 'Curio');
    });

    it('tests "curio" type registration and lookup specifically', () => {
      const CurioComponent = { name: 'CurioComponent' };
      registry.register('curio', CurioComponent);
      assert.equal(registry.has('curio'), true);
      assert.equal(registry.get('curio'), CurioComponent);
    });

    it('allows updating an existing registered component', () => {
      const CustomNoteBody = { name: 'EnhancedNoteCardBody' };
      registry.register('note', CustomNoteBody);
      assert.equal(registry.get('note'), CustomNoteBody);
    });

    it('retrieves all card type definitions including custom extensions', () => {
      const initialDefs = registry.getAllDefinitions();
      assert.ok(initialDefs.length >= 4);

      registry.register('todo', { name: 'TodoBody' }, {
        name: '待办',
        defaultTitle: 'Todo List'
      });

      const updatedDefs = registry.getAllDefinitions();
      assert.equal(updatedDefs.length, initialDefs.length + 1);
      assert.ok(updatedDefs.some((d: CardTypeDefinition) => d.type === 'todo' && d.defaultTitle === 'Todo List'));
    });

    it('clears custom registrations and resets definitions to built-in presets', () => {
      registry.register('custom_temp', { name: 'Temp' });
      assert.equal(registry.has('custom_temp'), true);

      registry.clear();
      assert.equal(registry.has('custom_temp'), false);
      assert.equal(registry.has('thought'), false); // components cleared
      assert.equal(registry.getAllDefinitions().length, 4); // definitions restored
    });
  });
});
