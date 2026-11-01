<!--
  ======================================================================================
  FINNEGAN COSMIC STAGE - CardCollectControl Component
  Extracted temporary card collection controls, comment input strip, and feedback toast.
  ======================================================================================
-->
<script lang="ts">
  import { createEventDispatcher } from 'svelte';
  import { getCardPromotionStrategy } from '../../lib/promotionRegistry';
  import { PromoteTempCard } from '../../../wailsjs/go/main/App.js';

  const dispatch = createEventDispatcher();

  export let card: any;
  export let isFocused: boolean = false;
  export let isHovered: boolean = false;
  export let hasAnyFocused: boolean = false;

  $: isTemporary = card.isTemporary === false ? false : Boolean(card.isTemporary || card.origin === 'bugu' || card.origin === 'atproto' || card.type === 'radar');
  $: promotionStrategy = getCardPromotionStrategy(card);

  let isPromoting = false;
  let collectPromptMessage: string | null = null;
  let collectPromptTimer: any = null;

  let isCommentStripOpen = false;
  let commentText = '';
  let commentInputEl: HTMLInputElement | null = null;
  let commentStripEl: HTMLDivElement | null = null;

  export function showCollectPrompt() {
    collectPromptMessage = '需先收集';
    if (collectPromptTimer) clearTimeout(collectPromptTimer);
    collectPromptTimer = setTimeout(() => {
      collectPromptMessage = null;
    }, 2400);
  }

  function openCommentStrip() {
    isCommentStripOpen = true;
    setTimeout(() => {
      commentInputEl?.focus();
    }, 50);
  }

  function closeCommentStrip() {
    isCommentStripOpen = false;
    commentText = '';
  }

  function handleCollectClick(e: MouseEvent) {
    e.stopPropagation();
    e.preventDefault();
    if (isPromoting) return;

    if (promotionStrategy.hasCommentPrompt) {
      if (isCommentStripOpen) {
        closeCommentStrip();
      } else {
        openCommentStrip();
      }
    } else {
      executePromote('', card.echoOf || '');
    }
  }

  function handleCommentKeydown(e: KeyboardEvent) {
    if (e.key === 'Enter') {
      e.preventDefault();
      e.stopPropagation();
      executePromote(commentText.trim(), card.echoOf || '');
    } else if (e.key === 'Escape') {
      e.preventDefault();
      e.stopPropagation();
      closeCommentStrip();
    }
  }

  function handleWindowClick(e: MouseEvent) {
    if (!isCommentStripOpen) return;
    if (commentStripEl && !commentStripEl.contains(e.target as Node)) {
      closeCommentStrip();
    }
  }

  async function executePromote(comment: string = '', targetEchoOf?: string) {
    if (isPromoting) return;
    isPromoting = true;
    const oldId = card.id;
    try {
      let promoted: any = null;
      const finalEchoOf = targetEchoOf !== undefined ? targetEchoOf : (card.echoOf || '');
      if (typeof PromoteTempCard === 'function') {
        try {
          promoted = await PromoteTempCard(card.id, comment, finalEchoOf);
        } catch (ipcErr) {
          console.warn('PromoteTempCard IPC error:', ipcErr);
        }
      }

      // 原地转正 (In-place promotion)
      card.isTemporary = false;
      if (promoted) {
        if (promoted.id) card.id = promoted.id;
        if (promoted.filePath) card.filePath = promoted.filePath;
        if (promoted.type && promoted.type !== 'local') {
          card.type = promoted.type;
        } else if (card.type === 'radar') {
          card.type = 'thought';
        }
        if (promoted.origin) card.origin = promoted.origin;
        if (promoted.createdAt) card.createdAt = promoted.createdAt;
        if (promoted.echoOf !== undefined) {
          card.echoOf = promoted.echoOf;
        } else if (targetEchoOf === 'NONE') {
          delete card.echoOf;
        }
        if (promoted.userComment !== undefined) {
          card.userComment = promoted.userComment;
        } else if (comment) {
          card.userComment = comment;
        }
      } else {
        if (card.type === 'radar') card.type = 'thought';
        if (targetEchoOf === 'NONE') {
          delete card.echoOf;
        } else if (targetEchoOf) {
          card.echoOf = targetEchoOf;
        }
        if (comment) card.userComment = comment;
      }
      delete card.macroExpiresAt;
      card = { ...card };

      closeCommentStrip();
      dispatch('cardPromoted', { originalId: oldId, card });
    } catch (err) {
      console.error('Failed to promote temp card:', err);
      card.isTemporary = false;
      if (card.type === 'radar') card.type = 'thought';
      if (targetEchoOf === 'NONE') {
        delete card.echoOf;
      }
      if (comment) card.userComment = comment;
      delete card.macroExpiresAt;
      card = { ...card };
      closeCommentStrip();
      dispatch('cardPromoted', { originalId: oldId, card });
    } finally {
      isPromoting = false;
    }
  }

  const handleCollect = handleCollectClick;
</script>

<svelte:window on:click={handleWindowClick} />

{#if collectPromptMessage}
  <div class="collect-prompt-toast">
    {collectPromptMessage}
  </div>
{/if}

{#if isTemporary && (isFocused || (isHovered && !hasAnyFocused))}
  <button
    type="button"
    class="collect-btn"
    class:focused={isFocused}
    class:scattered={!isFocused}
    class:has-comment-prompt={promotionStrategy.hasCommentPrompt}
    on:click={handleCollect}
    title={promotionStrategy.tooltip}
    aria-label={promotionStrategy.tooltip}
  >
    {#if promotionStrategy.icon === 'anchor_comment'}
      <svg
        class="collect-anchor-icon"
        viewBox="0 0 24 24"
        fill="none"
        stroke="currentColor"
        stroke-width="1.8"
        stroke-linecap="round"
        stroke-linejoin="round"
      >
        <!-- Speech / Note Bubble with Anchor Pin -->
        <path d="M21 11.5a8.38 8.38 0 0 1-.9 3.8 8.5 8.5 0 0 1-7.6 4.7 8.38 8.38 0 0 1-3.8-.9L3 21l1.9-5.7a8.38 8.38 0 0 1-.9-3.8 8.5 8.5 0 0 1 4.7-7.6 8.38 8.38 0 0 1 3.8-.9h.5a8.48 8.48 0 0 1 8 8v.5z" />
        <path d="M9 11h6" stroke-width="1.6" opacity="0.75" />
        <path d="M12 8v6" stroke-width="1.6" opacity="0.75" />
      </svg>
    {:else}
      <svg
        class="collect-book-icon"
        viewBox="0 0 24 24"
        fill="none"
        stroke="currentColor"
        stroke-width="1.8"
        stroke-linecap="round"
        stroke-linejoin="round"
      >
        <!-- Open Book Base (Spreads & Central Spine) -->
        <path
          class="book-spine-cover"
          d="M12 19.5C8 17.5 4.5 17.5 2 18V5.5C4.5 5 8 5 12 7M12 19.5C16 17.5 19.5 17.5 22 18V5.5C19.5 5 16 5 12 7M12 19.5V7"
        />
        <!-- Book Page Subtle Text Texture -->
        <path
          class="book-lines"
          d="M6 10.5h3M6 13.5h3M15 10.5h3M15 13.5h3"
          stroke-width="1.2"
          opacity="0.38"
        />
        <!-- Adding Page Animation (往里加页动效) -->
        <g class="adding-page">
          <path
            d="M12 7C14 5 17.5 5 19.5 5.5V16.5C17.5 16 14 16 12 18V7Z"
            fill="currentColor"
            fill-opacity="0.32"
            stroke-width="1.6"
          />
          <path
            d="M14 9.5h3.5M14 12.5h2.5"
            stroke-width="1"
            opacity="0.65"
          />
        </g>
      </svg>
    {/if}
  </button>
{/if}

{#if isTemporary && isCommentStripOpen}
  <!-- svelte-ignore a11y-click-events-have-key-events -->
  <!-- svelte-ignore a11y-no-static-element-interactions -->
  <div
    bind:this={commentStripEl}
    class="comment-input-strip"
    class:focused-mode={isFocused}
    on:click|stopPropagation
  >
    <input
      bind:this={commentInputEl}
      type="text"
      class="comment-input"
      placeholder={promotionStrategy.placeholder || '留下附着批注（回车直接收录）...'}
      bind:value={commentText}
      on:keydown={handleCommentKeydown}
    />
    <button
      type="button"
      class="strip-btn standalone-collect-btn"
      on:click|stopPropagation={() => executePromote('', 'NONE')}
      title="作为独立卡片收录（不附着母卡）"
      aria-label="作为独立卡片收录（不附着母卡）"
    >
      <svg viewBox="0 0 24 24" width="14" height="14" stroke="currentColor" stroke-width="2" fill="none">
        <path d="M18 13v6a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2V8a2 2 0 0 1 2-2h6"></path>
        <polyline points="15 3 21 3 21 9"></polyline>
        <line x1="10" y1="14" x2="21" y2="3"></line>
      </svg>
    </button>
    <button
      type="button"
      class="strip-btn commit-btn"
      on:click|stopPropagation={() => executePromote(commentText.trim(), card.echoOf || '')}
      title="确认附着收录 (Enter)"
      aria-label="确认附着收录"
    >
      <svg viewBox="0 0 24 24" width="14" height="14" stroke="currentColor" stroke-width="2.2" fill="none">
        <polyline points="20 6 9 17 4 12"></polyline>
      </svg>
    </button>
  </div>
{/if}

<style>
  /* Collect (转正) Button for Temporary Cards in Scattered & Focused States */
  .collect-btn {
    position: absolute;
    display: inline-flex;
    align-items: center;
    justify-content: center;
    border: none;
    cursor: pointer;
    user-select: none;
    z-index: 100;
    transition: all 0.25s cubic-bezier(0.16, 1, 0.3, 1);
  }

  .collect-btn.scattered {
    top: 10px;
    right: 10px;
    width: 26px;
    height: 26px;
    padding: 0;
    border-radius: 50%;
    background: rgba(255, 255, 255, 0.92);
    color: #0284c7;
    border: 1px solid rgba(56, 189, 248, 0.45);
    box-shadow: 0 2px 8px rgba(0, 0, 0, 0.08);
    backdrop-filter: blur(8px);
    -webkit-backdrop-filter: blur(8px);
  }

  .collect-btn.scattered:hover {
    background: #0284c7;
    color: #ffffff;
    border-color: #0284c7;
    box-shadow: 0 4px 14px rgba(2, 132, 199, 0.38);
    transform: scale(1.08);
  }

  .collect-btn.focused {
    top: 18px;
    right: 18px;
    width: 32px;
    height: 32px;
    padding: 0;
    border-radius: 50%;
    background: rgba(2, 132, 199, 0.12);
    color: #0284c7;
    border: 1px solid rgba(2, 132, 199, 0.35);
    box-shadow: 0 2px 8px rgba(2, 132, 199, 0.12);
    backdrop-filter: blur(8px);
    -webkit-backdrop-filter: blur(8px);
  }

  .collect-btn.focused:hover {
    background: #0284c7;
    color: #ffffff;
    border-color: #0284c7;
    box-shadow: 0 4px 16px rgba(2, 132, 199, 0.42);
    transform: scale(1.08);
  }

  .collect-btn:active {
    transform: scale(0.94);
  }

  .collect-book-icon {
    display: block;
    overflow: visible;
    flex-shrink: 0;
  }

  .collect-btn.scattered .collect-book-icon {
    width: 15px;
    height: 15px;
  }

  .collect-btn.focused .collect-book-icon {
    width: 18px;
    height: 18px;
  }

  .book-spine-cover {
    transform-origin: 12px 19.5px;
    transition: transform 0.25s cubic-bezier(0.16, 1, 0.3, 1);
  }

  .collect-btn:hover .book-spine-cover {
    transform: scale(1.04);
  }

  /* 往里加页动效 (Page-Adding Settle Animation) */
  .adding-page {
    transform-origin: 12px 19.5px;
    transition: transform 0.35s cubic-bezier(0.34, 1.45, 0.7, 1), opacity 0.25s ease;
    transform: translate(3.5px, -6px) rotate(16deg) scale(0.75);
    opacity: 0;
  }

  .collect-btn:hover .adding-page {
    transform: translate(0, 0) rotate(0deg) scale(1);
    opacity: 1;
    animation: page-settle 0.45s cubic-bezier(0.34, 1.45, 0.7, 1);
  }

  @keyframes page-settle {
    0% {
      transform: translate(4.5px, -7.5px) rotate(22deg) scale(0.7);
      opacity: 0;
    }
    65% {
      transform: translate(-0.6px, 0.5px) rotate(-3deg) scale(1.03);
      opacity: 1;
    }
    100% {
      transform: translate(0, 0) rotate(0deg) scale(1);
      opacity: 1;
    }
  }

  .collect-anchor-icon {
    display: block;
    overflow: visible;
    flex-shrink: 0;
  }

  .collect-btn.scattered .collect-anchor-icon {
    width: 15px;
    height: 15px;
  }

  .collect-btn.focused .collect-anchor-icon {
    width: 18px;
    height: 18px;
  }

  /* Frosted Glass In-place Comment Strip (Option B) */
  .comment-input-strip {
    position: absolute;
    bottom: 10px;
    left: 10px;
    right: 10px;
    z-index: 1005;
    display: flex;
    align-items: center;
    gap: 6px;
    padding: 4px 6px 4px 10px;
    background: rgba(255, 255, 255, 0.90);
    backdrop-filter: blur(20px);
    -webkit-backdrop-filter: blur(20px);
    border: 1px solid rgba(255, 255, 255, 0.95);
    border-radius: 12px;
    box-shadow: 0 8px 24px rgba(0, 0, 0, 0.16), 0 0 0 1px rgba(0, 0, 0, 0.05);
    animation: strip-slide-up 0.2s cubic-bezier(0.16, 1, 0.3, 1);
    pointer-events: auto;
  }

  .comment-input-strip.focused-mode {
    bottom: 20px;
    left: 24px;
    right: 24px;
    padding: 6px 8px 6px 14px;
    border-radius: 14px;
  }

  @keyframes strip-slide-up {
    from {
      opacity: 0;
      transform: translateY(8px) scale(0.96);
    }
    to {
      opacity: 1;
      transform: translateY(0) scale(1);
    }
  }

  .comment-input {
    flex: 1;
    border: none;
    background: transparent;
    outline: none;
    font-size: 11px;
    color: #1e293b;
    padding: 4px 0;
    min-width: 0;
    font-family: inherit;
  }

  .comment-input-strip.focused-mode .comment-input {
    font-size: 13px;
  }

  .comment-input::placeholder {
    color: #94a3b8;
    font-size: 10.5px;
  }

  .comment-input-strip.focused-mode .comment-input::placeholder {
    font-size: 12px;
  }

  .strip-btn {
    width: 24px;
    height: 24px;
    border-radius: 7px;
    border: none;
    display: inline-flex;
    align-items: center;
    justify-content: center;
    cursor: pointer;
    background: rgba(0, 0, 0, 0.04);
    color: #475569;
    transition: all 0.15s ease;
    padding: 0;
    flex-shrink: 0;
  }

  .strip-btn:hover {
    background: rgba(0, 0, 0, 0.09);
    color: #0f172a;
  }

  .strip-btn.standalone-collect-btn:hover {
    background: rgba(2, 132, 199, 0.15);
    color: #0284c7;
  }

  .strip-btn.commit-btn {
    background: #0284c7;
    color: #ffffff;
  }

  .strip-btn.commit-btn:hover {
    background: #0369a1;
  }

  /* Collect Prompt Floating Toast */
  .collect-prompt-toast {
    position: absolute;
    top: 12px;
    left: 50%;
    transform: translateX(-50%);
    background: rgba(15, 23, 42, 0.92);
    color: #f8fafc;
    font-size: 0.78rem;
    font-weight: 500;
    padding: 5px 12px;
    border-radius: 9999px;
    box-shadow: 0 4px 16px rgba(0, 0, 0, 0.2);
    backdrop-filter: blur(12px);
    -webkit-backdrop-filter: blur(12px);
    z-index: 1002;
    pointer-events: none;
    white-space: nowrap;
    animation: toast-pop 0.2s cubic-bezier(0.16, 1, 0.3, 1);
  }

  @keyframes toast-pop {
    from {
      opacity: 0;
      transform: translate(-50%, -6px) scale(0.96);
    }
    to {
      opacity: 1;
      transform: translate(-50%, 0) scale(1);
    }
  }
</style>
