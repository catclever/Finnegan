import { describe, it } from 'node:test';
import assert from 'node:assert/strict';
import {
  buildMotherChildRayDefs,
  buildSynapseRayDefs,
  assembleRenderedRays,
} from './cosmicRayEngine';

describe('Cosmic Ray Engine', () => {
  const motherCard: any = {
    id: 'mother-1',
    x: 0,
    y: 0,
    z: 10,
    tags: ['design'],
    isFadingOut: false,
  };

  const childCard: any = {
    id: 'child-1',
    origin: 'bugu',
    echoOf: 'mother-1',
    x: 100,
    y: 100,
    z: 20,
    matchedAnchor: 'Design Systems',
    echoReason: 'Shared design anchor',
    isFadingOut: false,
  };

  const synapseTargetCard: any = {
    id: 'other-1',
    x: 200,
    y: 50,
    z: 30,
    tags: ['design'],
    isFadingOut: false,
  };

  it('buildMotherChildRayDefs identifies unclustered valid child-to-mother connections', () => {
    const defs = buildMotherChildRayDefs([motherCard, childCard]);
    assert.equal(defs.length, 1);
    assert.equal(defs[0].id, 'mc:mother-1->child-1');
    assert.equal(defs[0].source.id, 'mother-1');
    assert.equal(defs[0].target.id, 'child-1');
    assert.equal(defs[0].matchedAnchor, 'Design Systems');
  });

  it('buildSynapseRayDefs connects hovered card with matching tag cards', () => {
    const synDefs = buildSynapseRayDefs(
      [motherCard, childCard, synapseTargetCard],
      'mother-1',
      null
    );
    assert.equal(synDefs.length, 1);
    assert.equal(synDefs[0].source.id, 'mother-1');
    assert.equal(synDefs[0].target.id, 'other-1');
    assert.equal(synDefs[0].tag, 'design');
  });

  it('buildSynapseRayDefs returns empty array when modal focus is active', () => {
    const synDefs = buildSynapseRayDefs(
      [motherCard, childCard, synapseTargetCard],
      'mother-1',
      'focused-card'
    );
    assert.equal(synDefs.length, 0);
  });

  it('assembleRenderedRays constructs rays with coordinates when child is hovered', () => {
    const motherChildDefs = buildMotherChildRayDefs([motherCard, childCard]);
    const rays = assembleRenderedRays({
      cards: [motherCard, childCard, synapseTargetCard],
      focusedCardId: null,
      hoveredCardId: 'child-1',
      motherChildRayDefs: motherChildDefs,
      synapseRayDefs: [],
      activeInstance: null,
    });

    assert.equal(rays.length, 1);
    assert.equal(rays[0].type, 'mother_child');
    assert.equal(rays[0].isPermanent, true);
    assert.ok(rays[0].coords.path.startsWith('M '));
    assert.equal(rays[0].badgeText, 'Design Systems');
  });

  it('assembleRenderedRays returns empty array when focusedCardId is present', () => {
    const motherChildDefs = buildMotherChildRayDefs([motherCard, childCard]);
    const rays = assembleRenderedRays({
      cards: [motherCard, childCard],
      focusedCardId: 'focused-1',
      hoveredCardId: 'child-1',
      motherChildRayDefs: motherChildDefs,
      synapseRayDefs: [],
      activeInstance: null,
    });
    assert.equal(rays.length, 0);
  });
});
