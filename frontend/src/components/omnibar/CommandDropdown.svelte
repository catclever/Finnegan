<!--
  ======================================================================================
  FINNEGAN COSMIC STAGE - CommandDropdown Subcomponent
  Task: modularize_backend_and_frontend_architecture_074
  Renders the command selection suggestions dropdown menu.
  ======================================================================================
-->
<script lang="ts">
  import type { CommandDef } from '../../types/card';

  export let matchedCommands: CommandDef[] = [];
  export let highlightedCmdIndex: number = 0;
  export let onSelect: (cmd: CommandDef) => void = () => {};
  export let onHighlight: (index: number) => void = () => {};
</script>

{#if matchedCommands.length > 0}
  <div class="dropdown-menu command-dropdown" on:wheel|stopPropagation>
    {#each matchedCommands as cmd, index}
      <!-- svelte-ignore a11y-click-events-have-key-events -->
      <!-- svelte-ignore a11y-no-static-element-interactions -->
      <div 
        class="dropdown-item" 
        class:active={index === highlightedCmdIndex}
        on:mouseenter={() => onHighlight(index)}
        on:mousedown|preventDefault={() => onSelect(cmd)}
      >
        <div class="cmd-item-header">
          <span class="cmd-icon">🔭</span>
          <span class="cmd-trigger">/{cmd.trigger.replace(/^\//, '')}</span>
        </div>
      </div>
    {/each}
  </div>
{/if}

<style>
  .dropdown-menu {
    width: 100%;
    margin-top: 8px;
    background: rgba(255, 255, 255, 0.96);
    backdrop-filter: blur(28px);
    -webkit-backdrop-filter: blur(28px);
    border: 1px solid rgba(255, 255, 255, 0.8);
    border-radius: 20px;
    box-shadow: 0 16px 40px rgba(0, 0, 0, 0.16);
    overflow: hidden;
    max-height: 280px;
    overflow-y: auto;
    z-index: 60;
    display: flex;
    flex-direction: column;
    animation: dropdown-slide 0.15s ease-out;
    box-sizing: border-box;
  }

  @keyframes dropdown-slide {
    from { opacity: 0; transform: translateY(-8px); }
    to { opacity: 1; transform: translateY(0); }
  }

  .dropdown-item {
    padding: 10px 16px;
    display: flex;
    flex-direction: column;
    gap: 3px;
    cursor: pointer;
    transition: background 0.15s ease;
    border-bottom: 1px solid rgba(0, 0, 0, 0.04);
  }

  .dropdown-item:last-child {
    border-bottom: none;
  }

  .dropdown-item.active {
    background: rgba(59, 130, 246, 0.1);
  }

  .cmd-item-header {
    display: flex;
    align-items: center;
    gap: 8px;
  }

  .cmd-trigger {
    font-weight: 700;
    color: #2563eb;
    font-size: 0.95rem;
  }
</style>
