<!--
  ======================================================================================
  FINNEGAN COSMIC STAGE - BabelCardBody Component
  Task: modularize_backend_and_frontend_architecture_074
  Renders the dedicated body for babel cards (card.type === 'babel').
  ======================================================================================
-->
<script lang="ts">
  import { BrowserOpenURL } from '../../../wailsjs/runtime/runtime.js';
  import { getCardDisplayTime } from '../../lib/time';

  export let card: any;
  export let isFocused: boolean = false;
  export let isHovered: boolean = false;
  export let babelTime: string = '';
  export let thoughtTime: string = '';
  export let displayTime: string = '';

  function getHostname(url: string | undefined): string {
    if (!url) return '';
    try {
      return new URL(url).hostname || 'Source';
    } catch {
      return 'Source';
    }
  }

  $: hostname = getHostname(card.sourceUrl);

  $: hasTranslation = Boolean(
    card.translatedText &&
    card.translatedText.trim() !== '' &&
    card.originalText &&
    card.translatedText.trim() !== card.originalText.trim()
  );

  function handleOpenSource(e: MouseEvent) {
    e.stopPropagation();
    e.preventDefault();
    if (card.sourceUrl) {
      BrowserOpenURL(card.sourceUrl);
    }
  }

  function handleAvatarError(e: Event) {
    const el = e.currentTarget as HTMLElement | null;
    if (el) el.style.display = 'none';
  }

  $: isBluesky = card.origin === 'atproto' || (typeof card.sourceUrl === 'string' && card.sourceUrl.includes('bsky.app'));
  $: resolvedDisplayTime = displayTime || babelTime || thoughtTime || getCardDisplayTime(card);
</script>

<div class="babel-card-body" class:focused={isFocused} class:hovered={isHovered}>
  <div class="babel-header">
    <div class="header-left">
      {#if card.sourceUrl}
        <a 
          href={card.sourceUrl} 
          target="_blank" 
          class="source-link" 
          on:click={handleOpenSource}
          title={card.sourceUrl}
        >
          {#if isBluesky}
            <span class="domain-icon bluesky-icon">🦋</span> @{card.authorName || 'bsky.app'}
          {:else}
            <span class="domain-icon">🌐</span> {hostname}
          {/if}
        </a>
      {/if}
    </div>
  </div>

  {#if hasTranslation}
    <div class="babel-content twinkle-container" class:focused={isFocused}>
      <div class="twinkle-text lang-trans">
        <p class="translated">{card.translatedText}</p>
      </div>
      <div class="twinkle-text lang-orig">
        <p class="original">{card.originalText}</p>
      </div>
    </div>
  {:else}
    <div class="babel-content single-container" class:focused={isFocused}>
      <p class="pure-original">{card.originalText || card.content}</p>
    </div>
  {/if}

  <div class="babel-footer">
    <div class="footer-left">
      {#if card.authorName || card.authorAvatar}
        <div class="author-pill" title={card.authorName || 'Author'}>
          {#if card.authorAvatar}
            <img src={card.authorAvatar} alt="" class="author-avatar" on:error={handleAvatarError} />
          {/if}
          {#if card.authorName}
            <span class="author-name">{card.authorName}</span>
          {/if}
        </div>
      {/if}
    </div>

    {#if isFocused && isBluesky}
      <div class="interaction-bar">
        <button class="interaction-btn reply-btn" on:click={(e) => { e.stopPropagation(); window.dispatchEvent(new CustomEvent('atproto:fetch_comments', { detail: { uri: card.sourceUrl || card.id } })); }}>
          <span class="icon">💬</span> {card.replyCount || card.commentsCount || ''}
        </button>
        <button class="interaction-btn">
          <span class="icon">🔁</span> {card.repostCount || ''}
        </button>
        <button class="interaction-btn">
          <span class="icon">♥</span> {card.likeCount || ''}
        </button>
      </div>
    {/if}

    {#if resolvedDisplayTime}
      <span class="card-time">{resolvedDisplayTime}</span>
    {/if}
  </div>
</div>

<style>
  .babel-card-body {
    position: relative;
    width: 100%;
    height: 100%;
    display: flex;
    flex-direction: column;
    z-index: 1;
  }

  .babel-header,
  .babel-content,
  .babel-footer {
    position: relative;
    z-index: 1;
  }

  .babel-header {
    display: flex;
    justify-content: space-between;
    align-items: center;
    gap: 8px;
    margin-bottom: 8px;
  }

  .babel-footer {
    display: flex;
    justify-content: space-between;
    align-items: center;
    gap: 8px;
    position: absolute;
    bottom: 0;
    left: 0;
    right: 0;
  }

  .babel-card-body.focused .babel-footer {
    position: relative;
    margin-top: auto;
    padding-top: 16px;
    bottom: auto;
    left: auto;
    right: auto;
  }

  .header-left,
  .footer-left {
    display: flex;
    align-items: center;
    gap: 6px;
    overflow: hidden;
    flex: 1;
    min-width: 0;
  }

  .author-pill {
    display: inline-flex;
    align-items: center;
    gap: 4px;
    font-size: 0.72rem;
    color: rgba(0, 0, 0, 0.65);
    background: rgba(255, 255, 255, 0.55);
    padding: 2px 7px;
    border-radius: 12px;
    max-width: 120px;
    overflow: hidden;
    white-space: nowrap;
    text-overflow: ellipsis;
    border: 1px solid rgba(255, 255, 255, 0.7);
  }

  .author-avatar {
    width: 14px;
    height: 14px;
    border-radius: 50%;
    object-fit: cover;
    flex-shrink: 0;
  }

  .author-name {
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }

  .source-link {
    font-size: 0.75rem;
    color: rgba(0, 0, 0, 0.5);
    text-decoration: none;
    display: flex;
    align-items: center;
    gap: 5px;
    background: rgba(255, 255, 255, 0.6);
    padding: 3px 8px;
    border-radius: 20px;
    transition: all 0.2s;
  }

  .source-link:hover {
    background: rgba(255, 255, 255, 0.9);
    color: rgba(0, 0, 0, 0.8);
  }

  .bluesky-icon {
    color: #0085ff;
  }

  .interaction-bar {
    display: flex;
    gap: 8px;
    align-items: center;
    background: rgba(255, 255, 255, 0.4);
    backdrop-filter: blur(4px);
    padding: 4px 8px;
    border-radius: 12px;
    border: 1px solid rgba(255, 255, 255, 0.6);
    box-shadow: 0 2px 4px rgba(0,0,0,0.05);
  }

  .interaction-btn {
    background: none;
    border: none;
    padding: 2px 6px;
    font-size: 0.75rem;
    color: rgba(0, 0, 0, 0.7);
    cursor: pointer;
    display: flex;
    align-items: center;
    gap: 4px;
    transition: color 0.2s, background 0.2s;
    border-radius: 8px;
  }

  .interaction-btn:hover {
    background: rgba(255, 255, 255, 0.8);
    color: rgba(0, 0, 0, 0.9);
  }

  :global(.card.temporary) .author-pill {
    background: rgba(255, 255, 255, 0.6);
    border: 1px solid rgba(226, 232, 240, 0.8);
  }

  :global(.card.temporary) .source-link {
    background: rgba(255, 255, 255, 0.6);
    border: 1px solid rgba(226, 232, 240, 0.75);
  }

  .card-time {
    font-size: 0.7rem;
    color: rgba(0, 0, 0, 0.42);
    font-variant-numeric: tabular-nums;
    letter-spacing: 0.02em;
    user-select: none;
    white-space: nowrap;
  }

  .single-container {
    position: relative;
    width: 100%;
    height: 105px;
    overflow: hidden;
    display: flex;
    flex-direction: column;
    justify-content: center;
  }

  .pure-original {
    font-size: 0.825rem;
    line-height: 1.55;
    color: var(--card-text-color, rgba(38, 38, 38, 0.95));
    font-weight: 450;
    display: -webkit-box;
    -webkit-line-clamp: 5;
    -webkit-box-orient: vertical;
    overflow: hidden;
    text-overflow: ellipsis;
    text-align: left;
    margin: 0;
  }

  /* Twinkling effect */
  .twinkle-container {
    position: relative;
    width: 100%;
    height: 95px;
  }

  .twinkle-text {
    position: absolute;
    top: 0;
    left: 0;
    width: 100%;
    height: 100%;
    overflow: hidden;
    display: flex;
    flex-direction: column;
    justify-content: center;
  }

  .lang-trans {
    animation: twinkle-a 8s infinite cubic-bezier(0.4, 0, 0.2, 1);
  }

  .lang-orig {
    animation: twinkle-b 8s infinite cubic-bezier(0.4, 0, 0.2, 1);
  }

  /* Stop animation completely when focused */
  .twinkle-container.focused .lang-trans,
  .twinkle-container.focused .lang-orig {
    animation: none !important;
    position: relative;
    opacity: 1;
    filter: none;
    height: auto;
    margin-bottom: 10px;
    display: block;
  }

  .twinkle-container.focused {
    height: auto;
    overflow: visible;
    max-height: none;
  }

  @keyframes twinkle-a {
    0%, 35% { opacity: 1; filter: blur(0px); z-index: 2; }
    45%, 85% { opacity: 0; filter: blur(8px); z-index: 1; }
    100% { opacity: 1; filter: blur(0px); z-index: 2; }
  }

  @keyframes twinkle-b {
    0%, 35% { opacity: 0; filter: blur(8px); z-index: 1; }
    45%, 85% { opacity: 1; filter: blur(0px); z-index: 2; }
    100% { opacity: 0; filter: blur(8px); z-index: 1; }
  }

  .babel-card-body.focused {
    flex: 1;
    min-height: 0;
  }

  .babel-card-body.focused .babel-content::-webkit-scrollbar {
    display: none;
    width: 0;
    height: 0;
  }

  .babel-card-body.focused .babel-content {
    justify-content: flex-start;
    flex: 1;
    min-height: 0;
    height: 100%;
    max-height: 100%;
    overflow-y: auto !important;
    overflow-x: hidden !important;
    overscroll-behavior: contain;
    -webkit-overflow-scrolling: touch;
    scrollbar-width: none;
    -ms-overflow-style: none;
    text-align: left;
  }

  .babel-card-body.focused .twinkle-container,
  .babel-card-body.focused .single-container {
    height: auto;
    min-height: 0;
    overflow-y: auto !important;
    overflow-x: hidden !important;
  }

  .babel-card-body.focused .original,
  .babel-card-body.focused .translated,
  .babel-card-body.focused .pure-original {
    -webkit-line-clamp: unset !important;
    display: block !important;
    overflow: visible !important;
    text-overflow: unset !important;
    max-height: none !important;
    padding-bottom: 18px;
  }

  .babel-card-body.focused .translated {
    font-size: 1.1rem;
    line-height: 1.75;
    color: rgba(30, 30, 30, 0.98);
    font-weight: 500;
  }

  .babel-card-body.focused .original {
    font-size: 0.95rem;
    line-height: 1.65;
    color: rgba(60, 60, 60, 0.85);
    margin-top: 16px;
    padding-top: 14px;
    border-top: 1px dashed rgba(0, 0, 0, 0.15);
  }

  .babel-card-body.focused .pure-original {
    font-size: 1.15rem;
    line-height: 1.75;
    color: rgba(30, 30, 30, 0.98);
  }

  p {
    margin: 0;
    font-size: 0.825rem;
    line-height: 1.5;
    white-space: pre-wrap;
    text-align: left;
  }

  .original {
    font-size: 0.775rem;
    color: var(--card-text-secondary, rgba(70, 70, 70, 0.9));
    font-style: italic;
    line-height: 1.4;
    display: -webkit-box;
    -webkit-line-clamp: 3;
    -webkit-box-orient: vertical;
    overflow: hidden;
    text-overflow: ellipsis;
    text-align: left;
  }

  .translated {
    font-size: 0.825rem;
    font-weight: 500;
    line-height: 1.4;
    display: -webkit-box;
    -webkit-line-clamp: 3;
    -webkit-box-orient: vertical;
    overflow: hidden;
    text-overflow: ellipsis;
    text-align: left;
    color: var(--card-text-color, rgba(38, 38, 38, 0.95));
  }
</style>
