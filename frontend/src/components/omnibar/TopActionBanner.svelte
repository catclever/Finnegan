<!--
  ======================================================================================
  FINNEGAN COSMIC STAGE - TopActionBanner Subcomponent
  Task: modularize_backend_and_frontend_architecture_074
  Renders the omnibar top banner across all interaction modes.
  ======================================================================================
-->
<script lang="ts">
  import type { CosmicCard, SearchSession } from '../../types/card';

  export let showTopBanner: boolean = false;
  export let commandParamMode: boolean = false;
  export let lockedCard: CosmicCard | null = null;
  export let commandSelectMode: boolean = false;
  export let searchSession: SearchSession;
  export let isNoteFormat: boolean = false;
  export let explicitNoteTitle: string = '';
  export let isDialogueFormat: boolean = false;
  export let explicitUserTitle: string = '';

  export let onExecuteCommand: () => void = () => {};
  export let onUnlockCard: () => void = () => {};
  export let onSolidifySearch: () => void = () => {};
  export let onRevertSearch: () => void = () => {};
  export let onRestoreSearch: () => void = () => {};
</script>

{#if showTopBanner}
  <div class="omnibar-top-banner">
    {#if commandParamMode}
      {#if lockedCard}
        <span class="banner-badge ready-badge">🚀 准备就绪</span>
        <!-- svelte-ignore a11y-click-events-have-key-events -->
        <!-- svelte-ignore a11y-no-static-element-interactions -->
        <button type="button" class="banner-clickable-action" on:click={onExecuteCommand} title="点击或按 Enter 执行命令">
          <span class="banner-key">↵</span>
          <span class="banner-action">执行命令</span>
        </button>
        <span class="banner-sep">·</span>
        <!-- svelte-ignore a11y-click-events-have-key-events -->
        <!-- svelte-ignore a11y-no-static-element-interactions -->
        <button type="button" class="banner-clickable-action" on:click={onUnlockCard} title="点击或按 Esc 重新选择">
          <span class="banner-key">Esc</span>
          <span class="banner-action">重新选择</span>
        </button>
      {:else}
        <span class="banner-badge param-badge">📄 观点参数</span>
        <span class="banner-key">↑ ↓</span>
        <span class="banner-action">移动</span>
        <span class="banner-sep">·</span>
        <span class="banner-key">↵</span>
        <span class="banner-action">锁定卡片</span>
        <span class="banner-sep">·</span>
        <span class="banner-key">Esc</span>
        <span class="banner-action">退出</span>
      {/if}
    {:else if commandSelectMode}
      <span class="banner-badge cmd-badge">🔭 模块命令</span>
      <span class="banner-key">↑ ↓</span>
      <span class="banner-action">选择</span>
      <span class="banner-sep">·</span>
      <span class="banner-key">↵</span>
      <span class="banner-action">确认</span>
      <span class="banner-sep">·</span>
      <span class="banner-key">Esc</span>
      <span class="banner-action">取消</span>
    {:else if searchSession.isActive && !searchSession.isConfirmed}
      <span class="banner-badge search-badge" style="--glow-color: {searchSession.color || 'hsl(180, 95%, 62%)'}">
        <span class="banner-dot"></span> 探针共鸣中
      </span>
      {#if !searchSession.isConfirmed}
        <button type="button" class="banner-clickable-action" on:click={onSolidifySearch} title="按 Shift+Enter 保留搜索结果">
          <span class="banner-key">⇧ ↵</span>
          <span class="banner-action">保留搜索结果</span>
        </button>
        <span class="banner-sep">·</span>
      {/if}
      <button type="button" class="banner-clickable-action" on:click={searchSession.isConfirmed ? onRestoreSearch : onRevertSearch} title="按 Esc 还原全部卡片">
        <span class="banner-key">Esc</span>
        <span class="banner-action">{searchSession.isConfirmed ? '恢复全部' : '清除'}</span>
      </button>
    {:else if isNoteFormat}
      <span class="banner-badge note-badge">📝 {explicitNoteTitle ? `备忘「${explicitNoteTitle}」` : '备忘'}</span>
      <span class="banner-key">↵</span>
      <span class="banner-action">换行</span>
      <span class="banner-sep">·</span>
      <span class="banner-key">Esc</span>
      <span class="banner-action">取消</span>
    {:else if isDialogueFormat}
      {#if explicitUserTitle}
        <span class="banner-badge dialogue-badge">💬 对话录「{explicitUserTitle}」</span>
      {/if}
      <span class="banner-key">↵</span>
      <span class="banner-action">换行</span>
      <span class="banner-sep">·</span>
      <span class="banner-key">/</span>
      <span class="banner-action">命令</span>
    {:else}
      <span class="banner-key">↵</span>
      <span class="banner-action">换行</span>
      <span class="banner-sep">·</span>
      <span class="banner-key">/</span>
      <span class="banner-action">命令</span>
    {/if}
  </div>
{/if}

<style>
  .omnibar-top-banner {
    align-self: flex-start;
    margin-left: 22px;
    margin-bottom: 6px;
    display: inline-flex;
    align-items: center;
    gap: 6px;
    padding: 0;
    background: transparent;
    backdrop-filter: none;
    -webkit-backdrop-filter: none;
    border: none;
    border-radius: 0;
    box-shadow: none;
    font-size: 0.72rem;
    color: rgba(50, 50, 50, 0.85);
    user-select: none;
    animation: banner-pop 0.18s cubic-bezier(0.16, 1, 0.3, 1);
  }

  @keyframes banner-pop {
    from {
      opacity: 0;
      transform: translateY(3px);
    }
    to {
      opacity: 1;
      transform: translateY(0);
    }
  }

  .banner-badge {
    display: inline-flex;
    align-items: center;
    gap: 3px;
    padding: 1px 6px;
    border-radius: 4px;
    font-weight: 600;
    font-size: 0.68rem;
    letter-spacing: 0.02em;
  }

  .banner-badge.cmd-badge {
    background: rgba(59, 130, 246, 0.12);
    color: #2563eb;
  }

  .banner-badge.note-badge {
    background: rgba(139, 92, 246, 0.12);
    color: #7c3aed;
  }

  .banner-badge.dialogue-badge {
    background: rgba(16, 185, 129, 0.12);
    color: #059669;
  }

  .banner-badge.param-badge {
    background: rgba(16, 185, 129, 0.12);
    color: #059669;
  }

  .banner-badge.ready-badge {
    background: rgba(245, 158, 11, 0.12);
    color: #d97706;
  }

  .banner-badge.search-badge {
    background: rgba(147, 51, 234, 0.1);
    color: #7e22ce;
  }

  .banner-dot {
    width: 5px;
    height: 5px;
    border-radius: 50%;
    background: var(--glow-color, #7e22ce);
    box-shadow: 0 0 5px var(--glow-color, #7e22ce);
    display: inline-block;
  }

  .banner-key {
    display: inline-flex;
    align-items: center;
    justify-content: center;
    padding: 1px 5px;
    background: rgba(255, 255, 255, 0.65);
    border: 1px solid rgba(255, 255, 255, 0.8);
    border-radius: 4px;
    font-size: 0.66rem;
    font-weight: 600;
    color: #374151;
    line-height: 1.25;
    font-family: inherit;
    box-shadow: 0 1px 2px rgba(0, 0, 0, 0.04);
  }

  .banner-action {
    color: rgba(55, 65, 81, 0.85);
    font-weight: 500;
    font-size: 0.7rem;
    text-shadow: 0 1px 2px rgba(255, 255, 255, 0.6);
  }

  .banner-sep {
    color: rgba(0, 0, 0, 0.2);
    font-weight: 600;
    font-size: 0.65rem;
    margin: 0 1px;
  }

  .banner-clickable-action {
    display: inline-flex;
    align-items: center;
    gap: 4px;
    background: transparent;
    border: none;
    padding: 0;
    margin: 0;
    cursor: pointer;
    font-family: inherit;
    border-radius: 4px;
    transition: transform 0.15s ease, opacity 0.15s ease;
  }

  .banner-clickable-action:hover {
    opacity: 0.8;
    transform: translateY(-1px);
  }

  .banner-clickable-action:active {
    transform: translateY(0);
  }
</style>
