<script lang="ts">
  import { marked } from 'marked';
  import DOMPurify from 'dompurify';

  export let content: string = '';
  export let isFocused: boolean = false;

  $: parsedHtml = DOMPurify.sanitize(marked.parse(content || '') as string);
</script>

<div class="markdown-content" class:focused={isFocused}>
  {@html parsedHtml}
</div>

<style>
  .markdown-content {
    font-family: inherit;
    line-height: 1.5;
    color: inherit;
    word-break: break-word;
  }

  /* Shared styles for both focused and scattered */
  :global(.markdown-content strong) {
    font-weight: 600;
  }

  :global(.markdown-content em) {
    font-style: italic;
  }

  :global(.markdown-content code) {
    font-family: 'JetBrains Mono', 'Fira Code', Consolas, monospace;
    background: rgba(255, 255, 255, 0.1);
    padding: 2px 4px;
    border-radius: 4px;
    font-size: 0.9em;
  }

  /* Focused mode styles */
  .markdown-content.focused {
    font-size: 14px;
  }

  :global(.markdown-content.focused h1) {
    font-size: 1.5em;
    font-weight: 600;
    margin: 1em 0 0.5em;
    line-height: 1.3;
  }

  :global(.markdown-content.focused h2) {
    font-size: 1.3em;
    font-weight: 600;
    margin: 1em 0 0.5em;
    line-height: 1.3;
  }

  :global(.markdown-content.focused h3) {
    font-size: 1.1em;
    font-weight: 600;
    margin: 1em 0 0.5em;
    line-height: 1.3;
  }

  :global(.markdown-content.focused h4),
  :global(.markdown-content.focused h5),
  :global(.markdown-content.focused h6) {
    font-size: 1em;
    font-weight: 600;
    margin: 1em 0 0.5em;
    line-height: 1.3;
  }

  :global(.markdown-content.focused p) {
    margin: 0 0 1em;
  }

  :global(.markdown-content.focused p:last-child) {
    margin-bottom: 0;
  }

  :global(.markdown-content.focused a) {
    color: #8ab4f8;
    text-decoration: none;
  }

  :global(.markdown-content.focused a:hover) {
    text-decoration: underline;
  }

  :global(.markdown-content.focused ul),
  :global(.markdown-content.focused ol) {
    margin: 0 0 1em;
    padding-left: 20px;
  }

  :global(.markdown-content.focused li) {
    margin-bottom: 0.25em;
  }

  :global(.markdown-content.focused blockquote) {
    margin: 0 0 1em;
    padding: 0.5em 1em;
    border-left: 4px solid rgba(255, 255, 255, 0.2);
    background: rgba(255, 255, 255, 0.05);
    border-radius: 0 4px 4px 0;
    color: rgba(255, 255, 255, 0.8);
  }

  :global(.markdown-content.focused blockquote p:last-child) {
    margin-bottom: 0;
  }

  :global(.markdown-content.focused pre) {
    background: rgba(0, 0, 0, 0.2);
    padding: 1em;
    border-radius: 6px;
    overflow-x: auto;
    margin: 0 0 1em;
  }

  :global(.markdown-content.focused pre code) {
    background: transparent;
    padding: 0;
    font-size: 0.9em;
  }

  /* Scattered mode (compact) styles */
  .markdown-content:not(.focused) {
    font-size: 13px;
    line-height: 1.4;
    display: -webkit-box;
    -webkit-box-orient: vertical;
    -webkit-line-clamp: 5;
    overflow: hidden;
    text-overflow: ellipsis;
    max-height: 100%;
  }

  :global(.markdown-content:not(.focused) *) {
    margin: 0 !important;
    padding: 0 !important;
  }

  :global(.markdown-content:not(.focused) p),
  :global(.markdown-content:not(.focused) ul),
  :global(.markdown-content:not(.focused) ol) {
    display: inline;
  }

  :global(.markdown-content:not(.focused) li) {
    display: inline;
  }

  :global(.markdown-content:not(.focused) li::before) {
    content: "• ";
    margin-left: 4px;
  }

  :global(.markdown-content:not(.focused) h1),
  :global(.markdown-content:not(.focused) h2),
  :global(.markdown-content:not(.focused) h3),
  :global(.markdown-content:not(.focused) h4),
  :global(.markdown-content:not(.focused) h5),
  :global(.markdown-content:not(.focused) h6) {
    display: inline;
    font-size: 1em;
    font-weight: 600;
    margin-right: 4px !important;
  }

  :global(.markdown-content:not(.focused) blockquote) {
    display: inline;
    font-style: italic;
    color: rgba(255, 255, 255, 0.7);
    border-left: none;
    background: transparent;
  }

  :global(.markdown-content:not(.focused) blockquote::before) {
    content: '"';
  }

  :global(.markdown-content:not(.focused) blockquote::after) {
    content: '" ';
  }

  :global(.markdown-content:not(.focused) pre) {
    display: inline;
    background: transparent;
    white-space: normal;
  }

  :global(.markdown-content:not(.focused) pre code) {
    background: rgba(255, 255, 255, 0.1);
    padding: 2px 4px !important;
  }

  :global(.markdown-content:not(.focused) br) {
    display: none;
  }
</style>
