<!--
  ======================================================================================
  FINNEGAN COSMIC STAGE - ChatLogCardBody Component
  Task: chat_log_card_frontend_032

  Renders the dialogue body of chat log cards in:
  - Scattering state (200x190) with screenplay dialogue snippet preview & gradient fade
  - Focused state (420x420) with full screenplay typography, metadata dateline, and natural scroll stream
  - Pure screenplay typography: character name cues, indented dialogue, elegant spacing
  - Strictly without cheesy badges or special copy buttons
  ======================================================================================
-->
<script lang="ts">
  import type { CosmicCard } from '../../types/card';
  import { parseDialogue, formatChatLogTime } from '../../lib/chatlog';

  export let card: CosmicCard;
  export let isFocused: boolean = false;
  export let isHovered: boolean = false;
  export let thoughtTime: string = '';

  $: turns = parseDialogue(card?.content || '');
  $: title = card?.title && card.title !== 'Thought' && card.title !== 'Card' && card.title !== 'Babel Echo' ? card.title : '';
  $: participants = Array.isArray(card?.participants) ? card.participants : [];
  $: timeStr = formatChatLogTime(card?.startedAt, card?.endedAt) || thoughtTime;
</script>

<div class="chat-log-card-body" class:focused={isFocused} class:hovered={isHovered}>
  {#if isFocused}
    <!-- Focused mode: 420x420 Screenplay Layout -->
    <header class="screenplay-header">
      {#if title}
        <h2 class="screenplay-title">{title}</h2>
      {/if}
      <div class="screenplay-meta">
        {#if participants.length > 0}
          <span class="meta-participants">{participants.join(' · ')}</span>
        {/if}
        {#if timeStr}
          <span class="meta-time">{timeStr}</span>
        {/if}
      </div>
      <div class="screenplay-divider" aria-hidden="true"></div>
    </header>

    <div class="screenplay-stream">
      {#each turns as turn, i (turn.speaker + i)}
        <div class="screenplay-turn">
          {#if turn.speaker}
            <div class="screenplay-character">{turn.speaker}</div>
          {/if}
          <div class="screenplay-dialogue">
            {#each turn.lines as line}
              <p class="dialogue-line">{line}</p>
            {/each}
          </div>
        </div>
      {/each}
    </div>
  {:else}
    <!-- Scattering mode: 200x190 Screenplay Preview -->
    <div class="screenplay-preview">
      {#if title}
        <div class="preview-title" title={title}>{title}</div>
      {/if}
      <div class="preview-turns">
        {#each turns.slice(0, 3) as turn, i (turn.speaker + i)}
          <div class="preview-turn">
            {#if turn.speaker}
              <span class="preview-character">{turn.speaker}</span>
            {/if}
            {#each turn.lines.slice(0, 2) as line}
              <p class="preview-line">{line}</p>
            {/each}
          </div>
        {/each}
      </div>
    </div>
  {/if}
</div>

<style>
  .chat-log-card-body {
    display: flex;
    flex-direction: column;
    width: 100%;
    height: 100%;
    position: relative;
    z-index: 1;
    overflow: hidden;
    text-align: left;
    box-sizing: border-box;
  }

  /* Focused mode: natural scrollable stream */
  .chat-log-card-body.focused {
    flex: 1;
    min-height: 0;
    height: 100%;
    max-height: 100%;
    overflow: hidden;
  }

  /* --- Focused Screenplay Typography --- */
  .screenplay-header {
    flex-shrink: 0;
    margin-bottom: 2px;
  }

  .screenplay-title {
    margin: 0 0 6px 0;
    font-size: 1.15rem;
    font-weight: 700;
    line-height: 1.35;
    letter-spacing: -0.015em;
    color: var(--card-text-color, rgba(22, 22, 22, 0.96));
    word-break: break-word;
  }

  .screenplay-meta {
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: 12px;
    font-size: 0.74rem;
    color: rgba(0, 0, 0, 0.46);
    font-variant-numeric: tabular-nums;
  }

  .meta-participants {
    font-weight: 550;
    letter-spacing: 0.03em;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }

  .meta-time {
    flex-shrink: 0;
    white-space: nowrap;
    letter-spacing: 0.02em;
  }

  .screenplay-divider {
    width: 100%;
    height: 1px;
    background: linear-gradient(to right, rgba(0, 0, 0, 0.09), rgba(0, 0, 0, 0.02));
    margin-top: 10px;
    margin-bottom: 12px;
  }

  .screenplay-stream {
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
    padding-right: 4px;
  }

  .screenplay-stream::-webkit-scrollbar {
    display: none;
    width: 0;
    height: 0;
  }

  .screenplay-turn {
    margin-bottom: 14px;
  }

  .screenplay-turn:last-child {
    margin-bottom: 6px;
  }

  .screenplay-character {
    font-size: 0.76rem;
    font-weight: 700;
    letter-spacing: 0.08em;
    text-transform: uppercase;
    color: rgba(0, 0, 0, 0.72);
    margin-bottom: 4px;
    font-family: ui-monospace, SFMono-Regular, "SF Mono", Menlo, Consolas, monospace;
  }

  .screenplay-dialogue {
    margin-left: 10px;
    border-left: 2px solid rgba(0, 0, 0, 0.07);
    padding-left: 10px;
  }

  .dialogue-line {
    margin: 0 0 6px 0;
    font-size: 0.91rem;
    line-height: 1.6;
    color: var(--card-text-color, rgba(32, 32, 32, 0.95));
    font-weight: 450;
    word-break: break-word;
  }

  .dialogue-line:last-child {
    margin-bottom: 0;
  }

  /* --- Scattering Screenplay Typography (200x190) --- */
  .screenplay-preview {
    width: 100%;
    height: 100%;
    position: relative;
    overflow: hidden;
    display: flex;
    flex-direction: column;
    text-align: left;
  }

  .preview-title {
    font-size: 0.82rem;
    font-weight: 650;
    line-height: 1.3;
    color: var(--card-text-color, rgba(28, 28, 28, 0.95));
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
    margin-bottom: 6px;
    letter-spacing: -0.01em;
    flex-shrink: 0;
  }

  .preview-turns {
    flex: 1;
    min-height: 0;
    display: flex;
    flex-direction: column;
    gap: 6px;
    overflow: hidden;
    mask-image: linear-gradient(to bottom, black 70%, transparent 100%);
    -webkit-mask-image: linear-gradient(to bottom, black 70%, transparent 100%);
  }

  .preview-turn {
    display: flex;
    flex-direction: column;
    gap: 2px;
  }

  .preview-character {
    font-size: 0.67rem;
    font-weight: 700;
    letter-spacing: 0.06em;
    text-transform: uppercase;
    color: rgba(0, 0, 0, 0.55);
    font-family: ui-monospace, SFMono-Regular, "SF Mono", Menlo, Consolas, monospace;
    display: block;
    line-height: 1.2;
    transition: color 0.15s ease;
  }

  .chat-log-card-body.hovered .preview-character {
    color: rgba(0, 0, 0, 0.75);
  }

  .preview-line {
    font-size: 0.76rem;
    line-height: 1.38;
    color: var(--card-text-color, rgba(38, 38, 38, 0.92));
    margin: 0;
    padding-left: 6px;
    border-left: 1.5px solid rgba(0, 0, 0, 0.08);
    overflow: hidden;
    text-overflow: ellipsis;
    display: -webkit-box;
    -webkit-line-clamp: 2;
    -webkit-box-orient: vertical;
  }
</style>
