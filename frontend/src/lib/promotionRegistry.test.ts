import test from 'node:test';
import assert from 'node:assert/strict';
import {
  hydrateModuleManifests,
  getCardPromotionStrategy,
  getModuleThemeColor,
  getModuleBeaconType,
  resetPromotionStrategies,
  DEFAULT_PROMOTION_STRATEGY,
} from './promotionRegistry.ts';
import type { ModuleManifest } from '../types/card.ts';

test('promotionRegistry: built-in presets for bugu and atproto work out of the box', () => {
  resetPromotionStrategies();

  const buguStrat = getCardPromotionStrategy({ origin: 'bugu', type: 'babel' });
  assert.equal(buguStrat.icon, 'anchor_comment');
  assert.equal(buguStrat.bindMotherCard, true);

  const atprotoStrat = getCardPromotionStrategy({ origin: 'atproto', type: 'babel' });
  assert.equal(atprotoStrat.icon, 'anchor_comment');
  assert.equal(atprotoStrat.bindMotherCard, true);

  const defaultStrat = getCardPromotionStrategy({ origin: 'local', type: 'thought' });
  assert.deepEqual(defaultStrat, DEFAULT_PROMOTION_STRATEGY);

  assert.equal(getModuleThemeColor('bugu'), '#f59e0b');
  assert.equal(getModuleThemeColor('atproto'), '#38bdf8');
  assert.equal(getModuleBeaconType('bugu:radar'), 'radar');
  assert.equal(getModuleBeaconType('atproto:echo'), 'butterfly');
});

test('promotionRegistry: hydrateModuleManifests registers external plugin strategy, theme, and beacon', () => {
  resetPromotionStrategies();

  const manifests: ModuleManifest[] = [
    {
      name: 'taohuo',
      version: '1.0.0',
      themeColor: '#ec4899',
      beaconType: 'radar',
      isExternal: true,
      promotionStrategy: {
        icon: 'default_book',
        tooltip: '收录好物卡片',
        hasCommentPrompt: false,
        placeholder: '',
        bindMotherCard: false,
      },
      cardTypes: [{ type: 'taohuo_item', uiEntry: '/api/plugins/taohuo/ui.js' }],
      commands: [{ id: 'taohuo:hunt', name: '淘货寻宝', trigger: 'taohuo', description: '寻宝', namespace: 'taohuo' }],
    },
  ];

  hydrateModuleManifests(manifests);

  // Lookup by card origin
  const byOrigin = getCardPromotionStrategy({ origin: 'taohuo', type: 'babel' });
  assert.equal(byOrigin.tooltip, '收录好物卡片');
  assert.equal(byOrigin.bindMotherCard, false);

  // Lookup by custom cardType
  const byType = getCardPromotionStrategy({ origin: 'manual', type: 'taohuo_item' });
  assert.equal(byType.tooltip, '收录好物卡片');

  // Theme color & beacon type
  assert.equal(getModuleThemeColor('taohuo'), '#ec4899');
  assert.equal(getModuleBeaconType('taohuo'), 'radar');
  assert.equal(getModuleBeaconType('taohuo:hunt'), 'radar');
  assert.equal(getModuleBeaconType('taohuo:subcmd'), 'radar');
});
