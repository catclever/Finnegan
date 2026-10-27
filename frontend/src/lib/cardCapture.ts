import type { CosmicCard } from '../types/card';
import { toCosmicCard, stageCandidate, isProtectedCard, BASE_STAGE_DECAY_MS } from './stageDecay';
import { extractHashTitle, isChatLogFormat } from './chatlog';

export interface CreateOptimisticChatLogParams {
  text: string;
  images?: string[];
  cards: CosmicCard[];
  memoryPool: CosmicCard[];
  localVaultPool: CosmicCard[];
  scrollExpansion: number;
  sourceCardId?: string;
  latestManualCardId: string | null;
}

export function createOptimisticChatLogCard(params: CreateOptimisticChatLogParams) {
  const { text, images, cards, memoryPool, localVaultPool, scrollExpansion, sourceCardId } = params;
  let { latestManualCardId } = params;
  const tempId = 'optimistic_chatlog_' + Date.now() + '_' + Math.random().toString(36).slice(2, 7);
  const { title: userTitle, cleanText } = extractHashTitle(text);
  const hasUserTitle = Boolean(userTitle);

  let currentCards = cards;
  if (latestManualCardId) {
    const prevVIP = currentCards.find(c => c.id === latestManualCardId);
    if (prevVIP && !isProtectedCard({ ...prevVIP, isVIP: false }, currentCards, sourceCardId)) {
      currentCards = currentCards.map(c =>
        c.id === latestManualCardId
          ? { ...c, isVIP: false, decayDuration: BASE_STAGE_DECAY_MS }
          : c
      );
    }
  }
  latestManualCardId = tempId;

  const baseCard = toCosmicCard({
    id: tempId,
    title: userTitle || '正在提炼主题...',
    type: 'chat_log',
    content: cleanText || text,
    images: images || [],
    isEnriching: !hasUserTitle,
  }, 'chat_log', 'manual');
  baseCard.isEnriching = !hasUserTitle;

  const { stagedCard, updatedActiveCards, updatedMemoryPool } = stageCandidate(
    baseCard,
    true,
    scrollExpansion,
    currentCards,
    memoryPool,
    sourceCardId
  );
  const nextMemoryPool = updatedMemoryPool || memoryPool;
  const newCard = {
    ...stagedCard,
    images: images || stagedCard.images,
    isVIP: true,
    decayDuration: 12 * 3600 * 1000,
    z: stagedCard.z ?? 3,
    isEnriching: !hasUserTitle,
  };

  const nextLocalVaultPool = [newCard, ...localVaultPool.filter(c => c.id !== newCard.id)];
  const nextCards = [...updatedActiveCards.filter(c => c.id !== newCard.id), newCard];

  return {
    tempId,
    cards: nextCards,
    memoryPool: nextMemoryPool,
    localVaultPool: nextLocalVaultPool,
    latestManualCardId,
  };
}

export interface ReconcileChatLogParams {
  tempId: string;
  res: any;
  originalImages?: string[];
  cards: CosmicCard[];
  localVaultPool: CosmicCard[];
  latestManualCardId: string | null;
}

export function reconcileChatLogCard(params: ReconcileChatLogParams) {
  const { tempId, res, originalImages, cards, localVaultPool } = params;
  let { latestManualCardId } = params;

  const realId = res?.docId || res?.DocID;
  const realTitle = res?.title || res?.Title;
  const realParticipants = res?.participants || res?.Participants;
  const realStartedAt = res?.startedAt || res?.StartedAt;
  const realEndedAt = res?.endedAt || res?.EndedAt;
  const realImages = res?.images || res?.Images || originalImages;

  if (latestManualCardId === tempId && realId) {
    latestManualCardId = realId;
  }

  const updateCard = (c: CosmicCard): CosmicCard => {
    if (c.id === tempId || (realId && c.id === realId)) {
      return {
        ...c,
        id: realId || c.id,
        title: realTitle || c.title,
        participants: realParticipants || c.participants,
        startedAt: realStartedAt || c.startedAt,
        endedAt: realEndedAt || c.endedAt,
        images: realImages || c.images,
        isEnriching: false,
      };
    }
    return c;
  };

  return {
    cards: cards.map(updateCard),
    localVaultPool: localVaultPool.map(updateCard),
    latestManualCardId,
  };
}

export function rollbackOptimisticCard(params: {
  tempId: string;
  cards: CosmicCard[];
  localVaultPool: CosmicCard[];
  latestManualCardId: string | null;
}) {
  const { tempId, cards, localVaultPool, latestManualCardId } = params;
  return {
    cards: cards.filter(c => c.id !== tempId),
    localVaultPool: localVaultPool.filter(c => c.id !== tempId),
    latestManualCardId: latestManualCardId === tempId ? null : latestManualCardId,
  };
}

export interface CreateNoteCardParams {
  docId: string;
  text: string;
  cards: CosmicCard[];
  memoryPool: CosmicCard[];
  localVaultPool: CosmicCard[];
  scrollExpansion: number;
  sourceCardId?: string;
  latestManualCardId?: string | null;
}

export function createNoteCard(params: CreateNoteCardParams) {
  const { docId, text, memoryPool, localVaultPool, scrollExpansion, sourceCardId } = params;
  let { cards, latestManualCardId } = params;

  if (latestManualCardId) {
    const prevVIP = cards.find(c => c.id === latestManualCardId);
    if (prevVIP && !isProtectedCard({ ...prevVIP, isVIP: false }, cards, sourceCardId)) {
      cards = cards.map(c =>
        c.id === latestManualCardId
          ? { ...c, isVIP: false, decayDuration: BASE_STAGE_DECAY_MS }
          : c
      );
    }
  }
  latestManualCardId = docId;

  const { title, cleanText } = extractHashTitle(text);
  const noteBody = (title && cleanText.trim()) ? cleanText.trim() : text;
  const noteTitle = title || (noteBody.length > 20 ? noteBody.slice(0, 20) + '...' : noteBody);
  const noteCard = toCosmicCard({
    id: docId,
    title: noteTitle,
    type: 'note',
    content: noteBody,
    createdAt: Date.now()
  }, 'note', 'manual');

  const { stagedCard, updatedActiveCards, updatedMemoryPool } = stageCandidate(
    noteCard,
    true,
    scrollExpansion,
    cards,
    memoryPool,
    sourceCardId
  );
  const nextMemoryPool = updatedMemoryPool || memoryPool;
  const newCard = {
    ...stagedCard,
    isVIP: true,
    decayDuration: 12 * 3600 * 1000,
    z: stagedCard.z ?? 3,
  };
  const nextCards = [...updatedActiveCards.filter(c => c.id !== newCard.id), newCard];
  const nextLocalVaultPool = [newCard, ...localVaultPool.filter(c => c.id !== docId)];

  return {
    cards: nextCards,
    memoryPool: nextMemoryPool,
    localVaultPool: nextLocalVaultPool,
    latestManualCardId,
  };
}

export interface MergeCapturedThoughtParams {
  data: any;
  cards: CosmicCard[];
  memoryPool: CosmicCard[];
  localVaultPool: CosmicCard[];
  scrollExpansion: number;
  sourceCardId?: string;
  latestManualCardId: string | null;
}

export function mergeCapturedThought(params: MergeCapturedThoughtParams) {
  const { data, memoryPool, scrollExpansion, sourceCardId } = params;
  let { cards, localVaultPool, latestManualCardId } = params;

  const cardId = data.DocID || ('thought_' + Date.now() + '_' + Math.random().toString(36).slice(2, 7));
  const images = data.Images || data.images;
  const isChat = data.Type === 'chat_log' || isChatLogFormat(data.Text);
  const isNote = data.Type === 'note';
  const cardTitle = data.Title || data.title;

  if (isNote) {
    const noteCard = toCosmicCard({
      id: cardId,
      title: cardTitle || (data.Text ? (data.Text.length > 20 ? data.Text.slice(0, 20) + '...' : data.Text) : 'Note'),
      type: 'note',
      content: data.Text,
      createdAt: data.CapturedAt || Date.now()
    }, 'note', 'manual');

    const existingCard = cards.find(c => c.id === cardId);
    if (!existingCard) {
      const { stagedCard, updatedActiveCards, updatedMemoryPool } = stageCandidate(
        noteCard,
        true,
        scrollExpansion,
        cards,
        memoryPool,
        sourceCardId
      );
      return {
        cards: [...updatedActiveCards.filter(c => c.id !== stagedCard.id), stagedCard],
        memoryPool: updatedMemoryPool || memoryPool,
        localVaultPool: [stagedCard, ...localVaultPool.filter(c => c.id !== cardId)],
        latestManualCardId,
      };
    } else {
      return {
        cards,
        memoryPool,
        localVaultPool: [existingCard, ...localVaultPool.filter(c => c.id !== cardId)],
        latestManualCardId,
      };
    }
  }

  // Check if this card was optimistically staged or already exists on stage
  const existingCard = cards.find(c => c.id === cardId || (isChat && c.isEnriching));
  if (existingCard) {
    const updateCard = (c: CosmicCard): CosmicCard => {
      if (c.id === existingCard.id || c.id === cardId) {
        return {
          ...c,
          id: cardId,
          title: cardTitle || (c.title === '正在提炼主题...' ? cardTitle || c.title : c.title),
          type: isChat ? 'chat_log' : c.type,
          content: data.Text || c.content,
          images: images || c.images,
          participants: data.Participants || data.participants || c.participants,
          startedAt: data.StartedAt || data.startedAt || c.startedAt,
          endedAt: data.EndedAt || data.endedAt || c.endedAt,
          tags: data.Tags || data.tags || c.tags,
          keywords: data.Keywords || data.keywords || c.keywords,
          isEnriching: false,
        };
      }
      return c;
    };

    return {
      cards: cards.map(updateCard),
      memoryPool,
      localVaultPool: localVaultPool.map(updateCard),
      latestManualCardId: latestManualCardId === existingCard.id ? cardId : latestManualCardId,
    };
  }

  let currentCards = cards;
  if (latestManualCardId) {
    const prevVIP = currentCards.find(c => c.id === latestManualCardId);
    if (prevVIP && !isProtectedCard({ ...prevVIP, isVIP: false }, currentCards, sourceCardId)) {
      currentCards = currentCards.map(c =>
        c.id === latestManualCardId
          ? { ...c, isVIP: false, decayDuration: BASE_STAGE_DECAY_MS }
          : c
      );
    }
  }
  latestManualCardId = cardId;
  const baseCard = toCosmicCard({
    id: cardId,
    title: cardTitle,
    type: isChat ? 'chat_log' : 'thought',
    content: data.Text,
    images,
    participants: data.Participants || data.participants,
    startedAt: data.StartedAt || data.startedAt,
    endedAt: data.EndedAt || data.endedAt,
    tags: data.Tags || data.tags,
    keywords: data.Keywords || data.keywords,
  }, isChat ? 'chat_log' : 'thought', 'manual');

  const { stagedCard, updatedActiveCards, updatedMemoryPool } = stageCandidate(
    baseCard,
    true,
    scrollExpansion,
    currentCards,
    memoryPool,
    sourceCardId
  );
  const newCard = {
    ...stagedCard,
    images: data.Images || data.images || stagedCard.images,
    isVIP: true,
    decayDuration: 12 * 3600 * 1000,
    z: stagedCard.z ?? 3
  };

  return {
    cards: [...updatedActiveCards.filter(c => c.id !== newCard.id), newCard],
    memoryPool: updatedMemoryPool || memoryPool,
    localVaultPool: [newCard, ...localVaultPool.filter(c => c.id !== newCard.id)],
    latestManualCardId,
  };
}
