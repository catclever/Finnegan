<script lang="ts">
  import { WindowToggleMaximise } from '../../wailsjs/runtime/runtime.js';
</script>

<!-- svelte-ignore a11y-no-static-element-interactions -->
<div class="window-titlebar" on:dblclick={WindowToggleMaximise}>
  <!-- 左侧预留 76px 给 macOS 原生红绿灯按钮，pointer-events: none 确保原生按钮完全可点可响应 -->
  <div class="traffic-light-guard"></div>
  <!-- 其余区域均为窗口拖拽区 -->
  <div class="drag-region"></div>
</div>

<style>
  .window-titlebar {
    position: fixed;
    top: 0;
    left: 0;
    right: 0;
    height: 38px;
    z-index: 9999; /* 保证高于全屏遮罩(500)与浮动卡片 */
    display: flex;
    align-items: center;
    pointer-events: none; /* 穿透未占用部分 */
  }

  .traffic-light-guard {
    width: 76px;
    height: 100%;
    flex-shrink: 0;
    pointer-events: none;
    --wails-draggable: no-drag;
    -webkit-app-region: no-drag;
  }

  .drag-region {
    flex: 1;
    height: 100%;
    pointer-events: auto;
    --wails-draggable: drag;
    -webkit-app-region: drag;
    user-select: none;
    -webkit-user-select: none;
    background: transparent;
  }
</style>
