import { describe, it, beforeEach } from 'node:test';
import assert from 'node:assert/strict';
// @ts-ignore
import {
  CardRegistry,
  createCardRegistry,
  normalizeCardType,
  isCoreCardType,
  getCardDefaultTitle,
  CORE_CARD_TYPES,
  SHARED_CARD_TYPES,
  BUILTIN_CARD_DEFINITIONS,
  DEFAULT_CARD_TYPE,
  appendCacheBust,
  reconcileExternalCardTypes,
} from './cardRegistry.ts';
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

    it('defines selectable flags correctly for built-in card definitions', () => {
      assert.equal(BUILTIN_CARD_DEFINITIONS.thought.selectable, false);
      assert.equal(BUILTIN_CARD_DEFINITIONS.chat_log.selectable, false);
      assert.equal(BUILTIN_CARD_DEFINITIONS.babel.selectable, false);
      assert.equal(BUILTIN_CARD_DEFINITIONS.note.selectable, true);
      assert.equal(BUILTIN_CARD_DEFINITIONS.note.prefix, ':note');
      assert.equal(BUILTIN_CARD_DEFINITIONS.note.icon, '📝');
      assert.equal(BUILTIN_CARD_DEFINITIONS.book.selectable, true);
      assert.equal(BUILTIN_CARD_DEFINITIONS.book.prefix, ':book');
      assert.equal(BUILTIN_CARD_DEFINITIONS.book.icon, '📖');
      assert.equal(BUILTIN_CARD_DEFINITIONS.book.defaultTitle, 'Reading Note');
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
    const MockCurioBody = { name: 'CurioCardBody' };

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

    it('supports unregistering an extension type and gracefully falling back to default', () => {
      registry.register('curio', MockCurioBody);
      assert.equal(registry.has('curio'), true);
      assert.equal(registry.get('curio'), MockCurioBody);

      registry.unregister('curio');
      assert.equal(registry.has('curio'), false);
      assert.equal(registry.get('curio'), MockThoughtBody);
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
      assert.equal(registry.getAllDefinitions().length, 5); // definitions restored
    });

    it('retrieves only selectable definitions and updates dynamically when extensions register/unregister', () => {
      const selectableBefore = registry.getSelectableDefinitions();
      assert.equal(selectableBefore.length, 2);
      assert.ok(selectableBefore.some(d => d.type === 'note'));
      assert.ok(selectableBefore.some(d => d.type === 'book'));

      // Register non-selectable extension
      registry.register('audit_log', { name: 'AuditLog' }, {
        name: '审计日志',
        selectable: false
      });
      assert.equal(registry.getSelectableDefinitions().length, 2);

      // Register selectable extension (e.g. curio)
      registry.register('curio', MockCurioBody, {
        name: '奇物标本',
        prefix: ':curio',
        selectable: true
      });
      const selectableAfter = registry.getSelectableDefinitions();
      assert.equal(selectableAfter.length, 3);
      assert.ok(selectableAfter.some(d => d.type === 'note'));
      assert.ok(selectableAfter.some(d => d.type === 'book'));
      assert.ok(selectableAfter.some(d => d.type === 'curio'));

      // Unregister curio
      registry.unregister('curio');
      const selectableAfterUnreg = registry.getSelectableDefinitions();
      assert.equal(selectableAfterUnreg.length, 2);
      assert.ok(selectableAfterUnreg.some(d => d.type === 'note'));
      assert.ok(selectableAfterUnreg.some(d => d.type === 'book'));
    });

    it('reconciles external card manifests, appends cache-busting params, and falls back to Thought on unregister', () => {
      assert.equal(appendCacheBust('/api/plugins/taohuo/ui.js', 12345), '/api/plugins/taohuo/ui.js?t=12345');
      assert.equal(appendCacheBust('/api/plugins/taohuo/ui.js?v=1', 12345), '/api/plugins/taohuo/ui.js?v=1&t=12345');
      assert.equal(appendCacheBust('/api/plugins/taohuo/ui.js'), '/api/plugins/taohuo/ui.js');

      const MockExternalHost = { name: 'ExternalCardHost' };
      const trackedUITypes = new Set<string>();
      const trackedDefTypes = new Set<string>();

      const manifestsWithTaohuo: any[] = [
        {
          name: 'taohuo',
          description: '淘货模块',
          themeColor: '#ec4899',
          isExternal: true,
          cardTypes: [
            {
              type: 'taohuo',
              name: '淘货',
              icon: '🛍️',
              prefix: ':taohuo',
              submitLabel: '淘货入库',
              badgeColor: '#db2777',
              selectable: true,
              customElementTag: 'finnegan-taohuo-card',
              shellStyle: { borderColor: '#db2777' },
              uiEntry: '/api/plugins/taohuo/ui.js',
            },
          ],
        },
      ];

      const res1 = reconcileExternalCardTypes(registry, manifestsWithTaohuo, trackedUITypes, trackedDefTypes);
      assert.equal(res1.activeUISpecs.length, 1);
      assert.equal(res1.activeUISpecs[0].spec.type, 'taohuo');

      // Simulate component registration by loadExternalPluginCardComponents
      registry.register('taohuo', MockExternalHost, registry.getDefinition('taohuo'));
      trackedUITypes.add('taohuo');
      assert.equal(registry.get('taohuo'), MockExternalHost);
      assert.equal(registry.getDefinition('taohuo').moduleName, 'taohuo');
      assert.deepEqual(registry.getDefinition('taohuo').shellStyle, { borderColor: '#db2777' });
      assert.equal(registry.getDefinition('taohuo').customElementTag, 'finnegan-taohuo-card');

      // Delete plugin from manifests -> reconcile must unregister taohuo and fall back to MockThoughtBody
      const res2 = reconcileExternalCardTypes(registry, [], trackedUITypes, trackedDefTypes);
      assert.deepEqual(res2.removedUITypes, ['taohuo']);
      assert.equal(registry.has('taohuo'), false);
      assert.equal(registry.get('taohuo'), MockThoughtBody);
    });
  });
});
