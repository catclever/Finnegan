<!--
  ======================================================================================
  FINNEGAN COSMIC STAGE - WritingPaper Component (Writing Mode Sheet)
  Task: writing_paper_physical_texture_and_silhouette_086
  
  Renders an authentic, tactile physical paper writing surface for long-form thoughts and notes:
  - Folded dog-ear corner (top-right) with under-flap shadow & realistic lighting
  - Manuscript ruled lines matching textarea line-height (repeating-linear-gradient with local attachment)
  - Warm ivory paper gradient with layered floating drop shadows and subtle edge highlight
  - Top manuscript margin with binding pin-hole (zero redundant explanatory badges)
  - Faint graphite pencil statistics (character / line count) at bottom
  - Integrated rounded feather quill submit button with dashed writing lines
  ======================================================================================
-->
<script lang="ts">
  import { onMount, tick } from 'svelte';
  import AttachmentTray from './AttachmentTray.svelte';

  export let omniText: string = '';
  export let isNoteFormat: boolean = false;
  export let noteContent: string = '';
  export let explicitNoteTitle: string = '';
  export let attachedImages: string[] = [];
  export let deleteHighlightIndex: number = -1;
  export let canSubmit: boolean = false;
  export let capturePending: boolean = false;

  export let onSubmit: () => void = () => {};
  export let onRemoveImage: (index: number) => void = () => {};
  export let onKeydown: (e: KeyboardEvent) => void = () => {};
  export let onPaste: (e: ClipboardEvent) => void = () => {};
  export let onInput: (e: Event) => void = () => {};
  export let onWheel: (e: WheelEvent) => void = () => {};

  let paperTextareaEl: HTMLTextAreaElement | null = null;

  $: activeText = isNoteFormat ? noteContent : omniText;
  $: charCount = activeText.trim().length;
  $: lineCount = activeText ? activeText.split('\n').length : 1;

  export function focusTextarea() {
    if (paperTextareaEl) {
      paperTextareaEl.focus();
    }
  }

  onMount(async () => {
    await tick();
    if (paperTextareaEl) {
      paperTextareaEl.focus();
      // Position cursor at end of text
      const len = paperTextareaEl.value.length;
      paperTextareaEl.setSelectionRange(len, len);
    }
  });

  function handleInternalInput(e: Event) {
    if (onInput) onInput(e);
  }
</script>

<div class="writing-paper-box" class:note-theme={isNoteFormat}>
  <!-- Right-top Dog-Ear Corner (Folded Paper) -->
  <div class="dog-ear-cutout">
    <div class="dog-ear-shadow"></div>
    <div class="dog-ear-flap"></div>
  </div>

  <!-- Paper Top Margin with Pin Hole (zero literal badges) -->
  <div class="paper-top-margin">
    <div class="paper-pin-hole" title="纸张装订孔"></div>
    {#if isNoteFormat && explicitNoteTitle}
      <div class="paper-note-title" title="便签标题">
        <span class="note-hash">#</span>
        <span class="note-heading">{explicitNoteTitle}</span>
      </div>
    {/if}
  </div>

  <!-- Paper Writing Area with Ruled Lines -->
  <div class="paper-body">
    <textarea
      bind:this={paperTextareaEl}
      class="paper-textarea"
      bind:value={omniText}
      on:input={handleInternalInput}
      on:keydown={onKeydown}
      on:paste={onPaste}
      on:wheel={onWheel}
      placeholder={isNoteFormat ? "记录确定性备忘..." : "尽情倾注您的长篇思绪..."}
      disabled={capturePending}
      rows="6"
    ></textarea>
  </div>

  <!-- Paper Footer: Stats, Attachment tray (if any) & Right Quill Submit Button -->
  <div class="paper-footer">
    <div class="footer-left">
      {#if attachedImages.length > 0}
        <AttachmentTray 
          images={attachedImages} 
          deleteHighlightIndex={deleteHighlightIndex} 
          on:remove={(e) => onRemoveImage(e.detail)} 
          onRemove={onRemoveImage}
        />
      {:else}
        <span class="paper-pencil-stat">
          {charCount} 字符 · {lineCount} 行
        </span>
      {/if}
    </div>

    <div class="footer-right">
      {#if attachedImages.length > 0}
        <span class="paper-pencil-stat attached-stat">
          {charCount} 字符 · {lineCount} 行
        </span>
      {/if}
      <span class="shortcut-tip">⌘↵</span>
      <!-- The Rounded Feather Quill with Dashed Writing Lines -->
      <button 
        type="button"
        class="omnibar-submit-btn paper-quill-btn" 
        class:active={canSubmit && !capturePending}
        class:note-mode={isNoteFormat}
        disabled={!canSubmit || capturePending}
        on:mousedown|preventDefault
        on:click|stopPropagation={onSubmit}
        title={isNoteFormat ? "保存备忘 (⌘↵)" : "录入思绪 (⌘↵)"}
      >
        <svg class="quill-icon" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-linecap="round" stroke-linejoin="round">
          <!-- Feather Plume: Soft rounded organic feather vane with subtle translucent fill -->
          <path 
            d="M19.5 3.5c-3.5.2-6.8 1.8-8.8 4.5-1 1.2-1.8 2-3 2.4-1.3.4-2.4 1.4-3 2.6-.7 1.2-.8 2.2-1.8 3.2-.8.8-1.5 1.4-1.7 1.8.4.3 1 .3 1.8-.2 1-.7 2-1.6 3.2-2.2 1.2-.6 2.2-.6 3.3-1.4 1.6-1 2.8-2.4 4-4.2 1.4-2.2 2.6-4.2 4-5.2 1-.7 1.8-1 2-1.3Z" 
            fill="currentColor" 
            fill-opacity="0.14" 
            stroke-width="1.6"
          />
          <!-- Feather Shaft: Sweeping central rachis from plume tip to nib -->
          <path d="M19.5 3.5C14.5 9 9.5 13.5 4 18" stroke-width="1.8" />
          <!-- Feather Barb Detail: Soft inner barb split -->
          <path d="M13.5 9c-1.8 1.2-3 2.8-3.8 4.5" stroke-width="1.3" opacity="0.75" />
          <!-- Written Text: Two lines of rounded dashed handwriting -->
          <path d="M8 18.5h11" stroke-dasharray="2 3" stroke-width="2" />
          <path d="M4 21.5h11" stroke-dasharray="2 3" stroke-width="2" />
        </svg>
      </button>
    </div>
  </div>
</div>

<style>
  .writing-paper-box {
    position: relative;
    width: 100%;
    display: flex;
    flex-direction: column;
    min-height: 270px;
    max-height: 58vh;
    border-radius: 12px 2px 14px 14px;
    background: linear-gradient(178deg, #ffffff 0%, #fdfdfa 32%, #f7f5ed 100%);
    border: 1px solid rgba(0, 0, 0, 0.08);
    box-shadow: 
      inset 0 1px 0 rgba(255, 255, 255, 1),
      inset 0 0 0 1px rgba(255, 255, 255, 0.6),
      0 20px 48px rgba(0, 0, 0, 0.22),
      0 4px 14px rgba(0, 0, 0, 0.06);
    box-sizing: border-box;
    transition: background 0.3s ease, border 0.3s ease, box-shadow 0.3s ease;
    animation: paper-unfold 0.28s cubic-bezier(0.16, 1, 0.3, 1);
    overflow: hidden;
  }

  @keyframes paper-unfold {
    from {
      opacity: 0.8;
      transform: scale(0.97) translateY(-6px);
    }
    to {
      opacity: 1;
      transform: scale(1) translateY(0);
    }
  }

  .writing-paper-box.note-theme {
    background: linear-gradient(178deg, #ffffff 0%, #fdfbfe 32%, #f7f3fb 100%);
    border-color: rgba(220, 210, 242, 0.8);
  }

  /* Right-top Dog-Ear Corner (Folded Paper) */
  .dog-ear-cutout {
    position: absolute;
    top: 0;
    right: 0;
    width: 28px;
    height: 28px;
    pointer-events: none;
    z-index: 10;
  }
  .dog-ear-shadow {
    position: absolute;
    top: 0;
    right: 0;
    width: 0;
    height: 0;
    border-style: solid;
    border-width: 0 28px 28px 0;
    border-color: transparent transparent rgba(0, 0, 0, 0.12) transparent;
    filter: drop-shadow(-2px 3px 3px rgba(0, 0, 0, 0.22));
  }
  .dog-ear-flap {
    position: absolute;
    top: 0;
    right: 0;
    width: 0;
    height: 0;
    border-style: solid;
    border-width: 0 28px 28px 0;
    border-color: transparent transparent #ece7dc transparent;
  }
  .dog-ear-flap::after {
    content: "";
    position: absolute;
    top: 0;
    right: -28px;
    width: 28px;
    height: 28px;
    background: linear-gradient(135deg, rgba(255, 255, 255, 0.95) 0%, rgba(215, 206, 192, 0.8) 100%);
    clip-path: polygon(0 0, 100% 100%, 0 100%);
  }

  .writing-paper-box.note-theme .dog-ear-flap {
    border-color: transparent transparent #eae4f4 transparent;
  }
  .writing-paper-box.note-theme .dog-ear-flap::after {
    background: linear-gradient(135deg, rgba(255, 255, 255, 0.95) 0%, rgba(210, 198, 235, 0.8) 100%);
  }

  /* Paper Top Margin / Pin-Hole Header */
  .paper-top-margin {
    height: 26px;
    border-bottom: 1px solid rgba(225, 135, 135, 0.25); /* Subtle pinkish/coral manuscript margin line */
    display: flex;
    align-items: center;
    padding: 0 22px;
    user-select: none;
    box-sizing: border-box;
  }

  .writing-paper-box.note-theme .paper-top-margin {
    border-bottom-color: rgba(147, 51, 234, 0.2);
  }

  .paper-pin-hole {
    width: 6px;
    height: 6px;
    border-radius: 50%;
    background: #11141c;
    box-shadow: inset 0 1px 2px rgba(0, 0, 0, 0.8), 0 1px 0 rgba(255, 255, 255, 0.9);
    margin-right: auto;
    opacity: 0.65;
  }

  .paper-note-title {
    display: flex;
    align-items: center;
    gap: 4px;
    font-size: 0.78rem;
    font-weight: 600;
    color: #6b21a8;
    letter-spacing: 0.02em;
    margin-right: 28px; /* avoid overlap with dog-ear */
  }

  .note-hash {
    color: #a855f7;
    font-weight: 700;
  }

  .note-heading {
    max-width: 320px;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }

  /* Paper Body & Ruled Lines */
  .paper-body {
    flex: 1;
    position: relative;
    display: flex;
    flex-direction: column;
  }

  /* Paper Textarea */
  .paper-textarea {
    width: 100%;
    min-height: 180px;
    max-height: 42vh;
    padding: 8px 22px 12px 22px;
    border: none;
    background: transparent;
    font-size: 15px;
    line-height: 32px; /* Precisely matches repeating-linear-gradient */
    color: #1e293b;
    outline: none;
    text-align: left;
    resize: none;
    box-sizing: border-box;
    font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, "PingFang SC", "Hiragino Sans GB", "Microsoft YaHei", sans-serif;
    letter-spacing: 0.015em;
    overflow-y: auto;
    background-image: repeating-linear-gradient(
      to bottom,
      transparent,
      transparent 31px,
      rgba(60, 80, 120, 0.07) 31px,
      rgba(60, 80, 120, 0.07) 32px
    );
    background-attachment: local;
    background-position: 0 7px;
  }

  .writing-paper-box.note-theme .paper-textarea {
    background-image: repeating-linear-gradient(
      to bottom,
      transparent,
      transparent 31px,
      rgba(124, 58, 237, 0.065) 31px,
      rgba(124, 58, 237, 0.065) 32px
    );
  }

  .paper-textarea::placeholder {
    color: rgba(100, 116, 139, 0.45);
    font-style: italic;
    font-weight: 400;
  }

  .paper-textarea::-webkit-scrollbar {
    width: 5px;
  }
  .paper-textarea::-webkit-scrollbar-thumb {
    background: rgba(100, 116, 139, 0.2);
    border-radius: 4px;
  }
  .paper-textarea::-webkit-scrollbar-track {
    background: transparent;
  }

  /* Paper Footer */
  .paper-footer {
    display: flex;
    align-items: flex-end;
    justify-content: space-between;
    padding: 8px 18px 12px 22px;
    border-top: 1px dashed rgba(0, 0, 0, 0.06);
    user-select: none;
  }

  .footer-left {
    flex: 1;
    min-width: 0;
    display: flex;
    align-items: center;
  }

  .footer-right {
    display: flex;
    align-items: center;
    gap: 10px;
    flex-shrink: 0;
  }

  /* Graphite Pencil Metrics Annotation */
  .paper-pencil-stat {
    font-size: 0.72rem;
    color: rgba(100, 116, 139, 0.55);
    font-feature-settings: "tnum";
    font-variant-numeric: tabular-nums;
    font-family: ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, monospace;
    user-select: none;
    letter-spacing: 0.02em;
  }

  .paper-pencil-stat.attached-stat {
    margin-right: 2px;
  }

  .shortcut-tip {
    font-size: 0.72rem;
    color: rgba(100, 116, 139, 0.45);
    user-select: none;
    font-family: ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, monospace;
  }

  /* Quill Button in Paper Mode */
  .paper-quill-btn {
    display: inline-flex;
    align-items: center;
    justify-content: center;
    width: 34px;
    height: 34px;
    border-radius: 50%;
    border: 1px solid rgba(0, 0, 0, 0.08);
    background: rgba(0, 0, 0, 0.04);
    color: rgba(0, 0, 0, 0.28);
    cursor: not-allowed;
    flex-shrink: 0;
    user-select: none;
    transition: all 0.2s cubic-bezier(0.16, 1, 0.3, 1);
    outline: none;
    padding: 0;
  }

  .quill-icon {
    width: 18px;
    height: 18px;
    display: block;
    transition: transform 0.25s cubic-bezier(0.34, 1.56, 0.64, 1);
    transform-origin: 20% 80%;
  }

  .paper-quill-btn.active {
    background: #18181b;
    color: #f4f4f5;
    border-color: rgba(255, 255, 255, 0.15);
    cursor: pointer;
    box-shadow: 0 2px 10px rgba(0, 0, 0, 0.18), 0 0 0 1px rgba(255, 255, 255, 0.1);
  }

  .paper-quill-btn.active.note-mode {
    background: #5b21b6;
    color: #f5f3ff;
    border-color: rgba(167, 139, 250, 0.3);
    box-shadow: 0 2px 12px rgba(91, 33, 182, 0.35);
  }

  .paper-quill-btn.active:hover {
    background: #09090b;
    transform: translateY(-1.5px) scale(1.06);
    box-shadow: 0 4px 16px rgba(0, 0, 0, 0.25);
  }

  .paper-quill-btn.active.note-mode:hover {
    background: #4c1d95;
    box-shadow: 0 4px 18px rgba(91, 33, 182, 0.45);
  }

  .paper-quill-btn.active:hover .quill-icon {
    transform: rotate(-7deg) translateY(-0.5px);
  }

  .paper-quill-btn.active:active {
    transform: translateY(0) scale(0.95);
    box-shadow: 0 1px 4px rgba(0, 0, 0, 0.15);
  }
</style>
