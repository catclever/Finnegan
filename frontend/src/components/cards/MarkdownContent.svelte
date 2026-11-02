<script lang="ts">
  import { Marked } from 'marked';
  import DOMPurify from 'dompurify';

  export let content: string = '';
  export let isFocused: boolean = false;
  export let threads: Array<{ id: string; title: string }> | undefined = undefined;

  function createParser(activeThreads?: Array<{ id: string; title: string }>) {
    const m = new Marked();
    m.use({
      breaks: true,
      extensions: [{
        name: 'themeMention',
        level: 'inline',
        start(src: string) { return src.indexOf('@'); },
        tokenizer(src: string) {
          if (activeThreads && activeThreads.length > 0) {
            for (const th of activeThreads) {
              if (!th.title) continue;
              const prefix = '@' + th.title;
              if (src.startsWith(prefix)) {
                return {
                  type: 'themeMention',
                  raw: prefix,
                  text: th.title,
                };
              }
            }
          }
          const match = /^@([^\s@#,\n<>:，。！？]+)/u.exec(src);
          if (match) {
            return {
              type: 'themeMention',
              raw: match[0],
              text: match[1],
            };
          }
        },
        renderer(token: any) {
          return `<span class="theme-mention">@${token.text}</span>`;
        }
      }]
    });
    return m;
  }

  function stripLeadingThemeMentions(raw: string, activeThreads?: Array<{ id: string; title: string }>): string {
    if (!raw || !activeThreads || activeThreads.length === 0) return raw;
    let res = raw.trim();
    for (const th of activeThreads) {
      if (!th.title) continue;
      const target = '@' + th.title;
      if (res.startsWith(target)) {
        res = res.slice(target.length).replace(/^[\s\r\n-_:，。！？,.]+/, '');
      }
    }
    return res || raw;
  }

  $: sanitizedContent = stripLeadingThemeMentions(content || '', threads);
  $: parser = createParser(threads);
  $: parsedHtml = DOMPurify.sanitize(parser.parse(sanitizedContent) as string);
</script>

<div class="markdown-content" class:focused={isFocused}>
  {@html parsedHtml}
</div>

<style>
  .markdown-content {
    font-family: inherit;
    line-height: 1.5;
    color: var(--card-text-color, #1e293b);
    word-break: break-word;
  }

  /* Shared styles for both focused and scattered */
  :global(.markdown-content strong) {
    font-weight: 600;
    color: var(--card-text-color, #0f172a);
  }

  :global(.markdown-content em) {
    font-style: italic;
  }

  :global(.markdown-content code) {
    font-family: 'JetBrains Mono', 'Fira Code', Consolas, monospace;
    background: rgba(0, 0, 0, 0.06);
    color: #0f172a;
    padding: 2px 4px;
    border-radius: 4px;
    font-size: 0.9em;
  }

  /* Focused mode styles */
  .markdown-content.focused {
    font-size: 14px;
    color: var(--card-text-color, #1e293b);
  }

  :global(.markdown-content.focused h1),
  :global(.markdown-content.focused h2),
  :global(.markdown-content.focused h3),
  :global(.markdown-content.focused h4),
  :global(.markdown-content.focused h5),
  :global(.markdown-content.focused h6) {
    font-weight: 600;
    margin: 1em 0 0.5em;
    line-height: 1.3;
    color: var(--card-text-color, #0f172a);
  }

  :global(.markdown-content.focused h1) {
    font-size: 1.5em;
  }

  :global(.markdown-content.focused h2) {
    font-size: 1.3em;
  }

  :global(.markdown-content.focused h3) {
    font-size: 1.1em;
  }

  :global(.markdown-content.focused h4),
  :global(.markdown-content.focused h5),
  :global(.markdown-content.focused h6) {
    font-size: 1em;
  }

  :global(.markdown-content.focused p) {
    margin: 0 0 1em;
    color: var(--card-text-color, #1e293b);
  }

  :global(.markdown-content.focused p:last-child) {
    margin-bottom: 0;
  }

  :global(.markdown-content.focused a) {
    color: #2563eb;
    text-decoration: none;
  }

  :global(.markdown-content.focused a:hover) {
    text-decoration: underline;
  }

  :global(.markdown-content.focused ul),
  :global(.markdown-content.focused ol) {
    margin: 0 0 1em;
    padding-left: 20px;
    color: var(--card-text-color, #1e293b);
  }

  :global(.markdown-content.focused li) {
    margin-bottom: 0.25em;
  }

  :global(.markdown-content.focused blockquote) {
    margin: 0 0 1em;
    padding: 0.5em 1em;
    border-left: 3px solid rgba(0, 0, 0, 0.18);
    background: rgba(0, 0, 0, 0.03);
    border-radius: 0 4px 4px 0;
    color: rgba(30, 30, 30, 0.85);
  }

  :global(.markdown-content.focused blockquote p:last-child) {
    margin-bottom: 0;
  }

  :global(.markdown-content.focused pre) {
    background: rgba(0, 0, 0, 0.04);
    padding: 1em;
    border-radius: 6px;
    overflow-x: auto;
    margin: 0 0 1em;
    color: #1e293b;
  }

  :global(.markdown-content.focused pre code) {
    background: transparent;
    padding: 0;
    font-size: 0.9em;
    color: inherit;
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
    width: 100%;
  }

  :global(.markdown-content:not(.focused) *) {
    margin: 0 !important;
    padding: 0 !important;
  }

  :global(.markdown-content:not(.focused) p) {
    display: block;
    margin: 0 0 3px !important;
  }

  :global(.markdown-content:not(.focused) p:last-child) {
    margin-bottom: 0 !important;
  }

  :global(.markdown-content:not(.focused) ul),
  :global(.markdown-content:not(.focused) ol) {
    display: block;
    margin: 0 0 2px !important;
  }

  :global(.markdown-content:not(.focused) ul:last-child),
  :global(.markdown-content:not(.focused) ol:last-child) {
    margin-bottom: 0 !important;
  }

  :global(.markdown-content:not(.focused) li) {
    display: inline;
  }

  :global(.markdown-content:not(.focused) li::before) {
    content: "• ";
    margin-left: 4px;
  }

  :global(.markdown-content:not(.focused) li:first-child::before) {
    margin-left: 0;
  }

  :global(.markdown-content:not(.focused) h1),
  :global(.markdown-content:not(.focused) h2),
  :global(.markdown-content:not(.focused) h3),
  :global(.markdown-content:not(.focused) h4),
  :global(.markdown-content:not(.focused) h5),
  :global(.markdown-content:not(.focused) h6) {
    display: block;
    font-size: 1.05em;
    font-weight: 600;
    margin: 0 0 2px !important;
  }

  :global(.markdown-content:not(.focused) blockquote) {
    display: block;
    font-style: italic;
    color: rgba(30, 30, 30, 0.75);
    margin: 0 0 2px !important;
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
    display: block;
    background: transparent;
    white-space: pre-wrap;
    word-break: break-all;
    margin: 0 0 2px !important;
  }

  :global(.markdown-content:not(.focused) pre code) {
    background: rgba(0, 0, 0, 0.06);
    color: #0f172a;
    padding: 1px 3px;
    border-radius: 3px;
  }

  :global(.markdown-content:not(.focused) br) {
    display: inline;
  }

  /* Inline Theme Mention styling */
  :global(.theme-mention) {
    display: inline-flex;
    align-items: center;
    gap: 2px;
    padding: 1px 7px 2px 6px;
    margin: 0 2px;
    font-size: 0.92em;
    font-weight: 600;
    line-height: 1.35;
    vertical-align: baseline;
    border-radius: 9999px;
    background: linear-gradient(135deg, rgba(56, 189, 248, 0.15), rgba(168, 85, 247, 0.18));
    border: 1px solid rgba(168, 85, 247, 0.32);
    color: #0284c7;
    box-shadow: 0 1px 4px rgba(56, 189, 248, 0.12);
    user-select: text;
    transition: all 0.2s cubic-bezier(0.16, 1, 0.3, 1);
  }

  :global(.theme-mention:hover) {
    background: linear-gradient(135deg, rgba(56, 189, 248, 0.25), rgba(168, 85, 247, 0.28));
    border-color: rgba(168, 85, 247, 0.5);
    color: #0369a1;
    transform: translateY(-0.5px);
    box-shadow: 0 2px 8px rgba(168, 85, 247, 0.22);
  }
</style>
