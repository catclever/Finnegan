const STYLE_ELEMENT_ID = 'finnegan-plugin-curio-styles';

const INTENT_COLORS = {
  wish: '#10b981',
  someday: '#f59e0b',
  archive: '#64748b',
  acquired: 'transparent',
};

const CURIO_STYLES = `
.finnegan-curio-card-body {
  display: flex;
  flex-direction: column;
  width: 100%;
  height: 100%;
  position: relative;
  z-index: 1;
  overflow: hidden;
  text-align: left;
  color: var(--card-text-color, #1e293b);
}

.finnegan-curio-card-body.focused {
  flex: 1;
  min-height: 0;
  overflow-y: auto !important;
  overflow-x: hidden !important;
  scrollbar-width: none;
}

.finnegan-curio-card-body.focused::-webkit-scrollbar {
  display: none;
}

.finnegan-curio-card-body .scattering-top-row,
.finnegan-curio-card-body .meta-row {
  display: flex;
  flex-wrap: wrap;
  gap: 4px;
  margin-bottom: 6px;
  align-items: center;
}

.finnegan-curio-card-body .badge {
  font-size: 10px;
  padding: 2px 6px;
  border-radius: 4px;
  color: white;
  font-weight: 500;
}

.finnegan-curio-card-body .platform-badge {
  background: #3b82f6;
}

.finnegan-curio-card-body .availability-badge {
  background: #ef4444;
}

.finnegan-curio-card-body .intent-badge.acquired {
  border: 1px solid #10b981;
  color: #10b981;
  background: transparent;
}

.finnegan-curio-card-body .price {
  font-size: 11px;
  font-weight: 600;
  color: #059669;
}

.finnegan-curio-card-body .price.large {
  font-size: 16px;
  margin: 4px 0;
}

.finnegan-curio-card-body .image-stage {
  position: relative;
  width: 100%;
  height: 92px;
  border-radius: 10px;
  background: rgba(0, 0, 0, 0.03);
  margin-bottom: 8px;
  overflow: hidden;
  flex-shrink: 0;
}

.finnegan-curio-card-body .album-glow-backdrop {
  position: absolute;
  inset: -12px;
  filter: blur(32px);
  opacity: 0.68;
  z-index: 1;
}

.finnegan-curio-card-body .backdrop-img,
.finnegan-curio-card-body .stage-img {
  position: absolute;
  top: 0;
  left: 0;
  width: 100%;
  height: 100%;
  opacity: 0;
  transition: opacity 0.6s ease;
}

.finnegan-curio-card-body .backdrop-img {
  object-fit: cover;
  transform: scale(1.2);
}

.finnegan-curio-card-body .stage-img {
  object-fit: contain;
}

.finnegan-curio-card-body .backdrop-img.active,
.finnegan-curio-card-body .stage-img.active {
  opacity: 1;
}

.finnegan-curio-card-body .stage-viewport {
  position: relative;
  height: 92px;
  z-index: 2;
}

.finnegan-curio-card-body .image-stage .indicator-dots {
  position: absolute;
  bottom: 5px;
  left: 50%;
  transform: translateX(-50%);
  display: flex;
  gap: 4px;
  z-index: 10;
}

.finnegan-curio-card-body .image-stage .indicator-dot {
  width: 4px;
  height: 4px;
  border-radius: 50%;
  background: rgba(255, 255, 255, 0.45);
  border: none;
  padding: 0;
  cursor: pointer;
}

.finnegan-curio-card-body .image-stage .indicator-dot.active {
  background: #fff;
  transform: scale(1.35);
}

.finnegan-curio-card-body .scattering-bottom {
  display: flex;
  flex-direction: column;
  flex: 1;
  min-height: 0;
}

.finnegan-curio-card-body .curio-title {
  font-weight: 600;
  font-size: 12px;
  margin-bottom: 4px;
}

.finnegan-curio-card-body .specimen-title {
  font-size: 18px;
  font-weight: 700;
  margin: 4px 0;
}

.finnegan-curio-card-body .brand {
  font-size: 12px;
  color: #64748b;
}

.finnegan-curio-card-body .content-fade {
  mask-image: linear-gradient(to bottom, black 60%, transparent 100%);
  -webkit-mask-image: linear-gradient(to bottom, black 60%, transparent 100%);
  max-height: 40px;
  overflow: hidden;
}

.finnegan-curio-card-body .archive-action-bar {
  display: flex;
  gap: 8px;
  margin: 12px 0;
  flex-wrap: wrap;
}

.finnegan-curio-card-body .archive-action-bar button {
  background: rgba(255, 255, 255, 0.8);
  border: 1px solid rgba(0, 0, 0, 0.1);
  border-radius: 6px;
  padding: 4px 8px;
  font-size: 12px;
  cursor: pointer;
  backdrop-filter: blur(4px);
  color: inherit;
}

.finnegan-curio-card-body .archive-action-bar button:hover {
  background: #fff;
}

.finnegan-curio-card-body .tags {
  display: flex;
  flex-wrap: wrap;
  gap: 4px;
  margin-top: 4px;
}

.finnegan-curio-card-body .tag {
  font-size: 10px;
  color: #0284c7;
  background: rgba(56, 189, 248, 0.12);
  padding: 1px 6px;
  border-radius: 4px;
  cursor: pointer;
}

.finnegan-curio-card-body .tag:hover {
  background: rgba(56, 189, 248, 0.24);
}

.finnegan-curio-card-body .time {
  font-size: 10px;
  color: rgba(0, 0, 0, 0.42);
  margin-top: auto;
  text-align: right;
}

/* Focused Image Gallery (reproducing CardImageGallery.svelte) */
.finnegan-curio-card-body .card-image-gallery {
  width: 100%;
  max-width: 100%;
  box-sizing: border-box;
  margin-bottom: 14px;
  flex-shrink: 0;
  position: relative;
  user-select: none;
}

.finnegan-curio-card-body .single-image-wrapper {
  width: 100%;
  height: auto;
  display: flex;
  justify-content: center;
  align-items: flex-start;
}

.finnegan-curio-card-body .gallery-wrapper {
  position: relative;
  width: 100%;
  height: auto;
}

.finnegan-curio-card-body .gallery-track {
  display: flex;
  flex-direction: row;
  align-items: flex-start;
  width: 100%;
  height: auto;
  overflow-x: auto;
  overflow-y: hidden;
  scroll-snap-type: x mandatory;
  scroll-behavior: smooth;
  -webkit-overflow-scrolling: touch;
  overscroll-behavior-x: contain;
  scrollbar-width: none;
  -ms-overflow-style: none;
  border-radius: 14px;
}

.finnegan-curio-card-body .gallery-track::-webkit-scrollbar {
  display: none;
  width: 0;
  height: 0;
}

.finnegan-curio-card-body .gallery-slide {
  flex: 0 0 100%;
  width: 100%;
  min-width: 100%;
  max-width: 100%;
  box-sizing: border-box;
  scroll-snap-align: start;
  scroll-snap-stop: always;
  display: flex;
  justify-content: center;
  align-items: flex-start;
  height: auto;
}

.finnegan-curio-card-body .gallery-image {
  width: 100%;
  height: auto;
  max-width: 100%;
  display: block;
  object-fit: contain;
  border-radius: 14px;
  box-shadow: 0 4px 20px rgba(0, 0, 0, 0.08), 0 0 0 1px rgba(0, 0, 0, 0.04);
  background: rgba(0, 0, 0, 0.02);
  user-select: none;
  -webkit-user-drag: none;
}

.finnegan-curio-card-body .gallery-nav-btn {
  position: absolute;
  top: clamp(40px, 50%, 180px);
  transform: translateY(-50%);
  width: 32px;
  height: 32px;
  border-radius: 50%;
  background: rgba(255, 255, 255, 0.9);
  backdrop-filter: blur(12px);
  -webkit-backdrop-filter: blur(12px);
  border: 1px solid rgba(0, 0, 0, 0.08);
  box-shadow: 0 3px 12px rgba(0, 0, 0, 0.15);
  color: rgba(30, 30, 30, 0.88);
  cursor: pointer;
  display: flex;
  align-items: center;
  justify-content: center;
  z-index: 10;
  transition: opacity 0.22s ease, transform 0.22s ease, background-color 0.2s ease, box-shadow 0.2s ease;
  padding: 0;
}

.finnegan-curio-card-body .gallery-nav-btn.prev-btn {
  left: 10px;
}

.finnegan-curio-card-body .gallery-nav-btn.next-btn {
  right: 10px;
}

.finnegan-curio-card-body .gallery-nav-btn:hover:not(.hidden) {
  background: #ffffff;
  color: #000000;
  transform: translateY(-50%) scale(1.1);
  box-shadow: 0 5px 16px rgba(0, 0, 0, 0.22);
}

.finnegan-curio-card-body .gallery-nav-btn:active:not(.hidden) {
  transform: translateY(-50%) scale(0.95);
}

.finnegan-curio-card-body .gallery-nav-btn.hidden {
  opacity: 0;
  pointer-events: none;
  transform: translateY(-50%) scale(0.8);
}

.finnegan-curio-card-body .card-image-gallery .indicator-dots {
  position: absolute;
  bottom: 10px;
  left: 50%;
  transform: translateX(-50%);
  display: flex;
  align-items: center;
  gap: 6px;
  padding: 4px 8px;
  background: rgba(0, 0, 0, 0.35);
  backdrop-filter: blur(10px);
  -webkit-backdrop-filter: blur(10px);
  border-radius: 12px;
  z-index: 10;
  pointer-events: auto;
}

.finnegan-curio-card-body .card-image-gallery .indicator-dot {
  width: 6px;
  height: 6px;
  border-radius: 50%;
  background: rgba(255, 255, 255, 0.55);
  border: none;
  padding: 0;
  cursor: pointer;
  transition: all 0.25s cubic-bezier(0.4, 0, 0.2, 1);
}

.finnegan-curio-card-body .card-image-gallery .indicator-dot:hover {
  background: rgba(255, 255, 255, 0.85);
}

.finnegan-curio-card-body .card-image-gallery .indicator-dot.active {
  width: 14px;
  border-radius: 3px;
  background: #ffffff;
  box-shadow: 0 1px 4px rgba(0, 0, 0, 0.45);
}

/* Markdown content scoped styles */
.finnegan-curio-card-body .curio-markdown-content {
  font-family: inherit;
  line-height: 1.5;
  color: var(--card-text-color, #1e293b);
  word-break: break-word;
}

.finnegan-curio-card-body .curio-markdown-content strong {
  font-weight: 600;
  color: var(--card-text-color, #0f172a);
}

.finnegan-curio-card-body .curio-markdown-content em {
  font-style: italic;
}

.finnegan-curio-card-body .curio-markdown-content code {
  font-family: 'JetBrains Mono', 'Fira Code', Consolas, monospace;
  background: rgba(0, 0, 0, 0.06);
  color: #0f172a;
  padding: 2px 4px;
  border-radius: 4px;
  font-size: 0.9em;
}

.finnegan-curio-card-body .curio-markdown-content.focused {
  font-size: 14px;
}

.finnegan-curio-card-body .curio-markdown-content.focused h1,
.finnegan-curio-card-body .curio-markdown-content.focused h2,
.finnegan-curio-card-body .curio-markdown-content.focused h3,
.finnegan-curio-card-body .curio-markdown-content.focused h4 {
  font-weight: 600;
  margin: 1em 0 0.5em;
  line-height: 1.3;
  color: var(--card-text-color, #0f172a);
}

.finnegan-curio-card-body .curio-markdown-content.focused p {
  margin: 0 0 1em;
}

.finnegan-curio-card-body .curio-markdown-content.focused p:last-child {
  margin-bottom: 0;
}

.finnegan-curio-card-body .curio-markdown-content.focused a {
  color: #2563eb;
  text-decoration: none;
}

.finnegan-curio-card-body .curio-markdown-content.focused a:hover {
  text-decoration: underline;
}

.finnegan-curio-card-body .curio-markdown-content.focused ul {
  margin: 0 0 1em;
  padding-left: 20px;
}

.finnegan-curio-card-body .curio-markdown-content.focused blockquote {
  margin: 0 0 1em;
  padding: 0.5em 1em;
  border-left: 3px solid rgba(0, 0, 0, 0.18);
  background: rgba(0, 0, 0, 0.03);
  border-radius: 0 4px 4px 0;
  color: rgba(30, 30, 30, 0.85);
}

.finnegan-curio-card-body .curio-markdown-content:not(.focused) {
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

.finnegan-curio-card-body .curio-markdown-content:not(.focused) * {
  margin: 0 !important;
  padding: 0 !important;
}

.finnegan-curio-card-body .curio-markdown-content:not(.focused) p {
  display: block;
  margin: 0 0 3px !important;
}

.finnegan-curio-card-body .curio-markdown-content:not(.focused) p:last-child {
  margin-bottom: 0 !important;
}
`;

function ensureStylesInjected() {
  if (typeof document === 'undefined' || !document.head) return;
  if (document.getElementById(STYLE_ELEMENT_ID)) return;
  const styleEl = document.createElement('style');
  styleEl.id = STYLE_ELEMENT_ID;
  styleEl.textContent = CURIO_STYLES;
  document.head.appendChild(styleEl);
}

function escapeHtml(raw) {
  return String(raw)
    .replace(/&/g, '&amp;')
    .replace(/</g, '&lt;')
    .replace(/>/g, '&gt;')
    .replace(/"/g, '&quot;')
    .replace(/'/g, '&#39;');
}

function renderInlineMarkdown(escapedLine) {
  return escapedLine
    .replace(/`([^`]+)`/g, '<code>$1</code>')
    .replace(/\*\*([^*]+)\*\*/g, '<strong>$1</strong>')
    .replace(/\*([^*]+)\*/g, '<em>$1</em>')
    .replace(/\[([^\]]+)\]\((https?:\/\/[^\s)]+)\)/g, '<a href="$2" target="_blank" rel="noopener noreferrer">$1</a>');
}

function renderSimpleMarkdown(raw) {
  if (!raw) return '';
  const escaped = escapeHtml(raw);
  const lines = escaped.split(/\r?\n/);
  const htmlParts = [];
  let inList = false;

  const closeList = () => {
    if (inList) {
      htmlParts.push('</ul>');
      inList = false;
    }
  };

  for (const line of lines) {
    const trimmed = line.trim();
    if (!trimmed) {
      closeList();
      continue;
    }
    const headingMatch = trimmed.match(/^(#{1,4})\s+(.*)$/);
    if (headingMatch) {
      closeList();
      const level = headingMatch[1].length;
      htmlParts.push(`<h${level}>${renderInlineMarkdown(headingMatch[2])}</h${level}>`);
      continue;
    }
    if (trimmed.startsWith('&gt; ')) {
      closeList();
      htmlParts.push(`<blockquote><p>${renderInlineMarkdown(trimmed.slice(5))}</p></blockquote>`);
      continue;
    }
    if (/^[-*]\s+/.test(trimmed)) {
      if (!inList) {
        htmlParts.push('<ul>');
        inList = true;
      }
      htmlParts.push(`<li>${renderInlineMarkdown(trimmed.replace(/^[-*]\s+/, ''))}</li>`);
      continue;
    }
    closeList();
    htmlParts.push(`<p>${renderInlineMarkdown(trimmed)}</p>`);
  }
  closeList();
  return htmlParts.join('');
}

function resolveCurioMeta(card) {
  const meta = (card && card.meta && typeof card.meta === 'object') ? card.meta : {};
  return {
    brand: meta.brand ?? card?.brand ?? '',
    platform: meta.platform ?? card?.platform ?? '',
    price: meta.price ?? card?.price ?? '',
    intent: meta.intent ?? card?.intent ?? '',
    availability: meta.availability ?? card?.availability ?? '',
    snapshotPath: meta.snapshotPath ?? card?.snapshotPath ?? '',
    screenshotPath: meta.screenshotPath ?? card?.screenshotPath ?? '',
    sourceUrl: meta.sourceUrl ?? card?.sourceUrl ?? '',
  };
}

function resolveAssetUrl(host, img) {
  if (!img || typeof img !== 'string') return '';
  if (host && typeof host.resolveAssetUrl === 'function') {
    return host.resolveAssetUrl(img);
  }
  if (
    img.startsWith('http://') ||
    img.startsWith('https://') ||
    img.startsWith('data:') ||
    img.startsWith('/api/assets/')
  ) {
    return img;
  }
  const clean = img.startsWith('/') ? img.slice(1) : img;
  return `/api/assets/${clean}`;
}

function createBadge(text, className, styleBg) {
  const span = document.createElement('span');
  span.className = className;
  span.textContent = text;
  if (styleBg) {
    span.style.backgroundColor = styleBg;
  }
  return span;
}

function createIntentBadge(intent) {
  const isAcquired = intent === 'acquired';
  const bg = !isAcquired ? (INTENT_COLORS[intent] || '#64748b') : 'transparent';
  return createBadge(
    intent,
    isAcquired ? 'badge intent-badge acquired' : 'badge intent-badge',
    bg
  );
}

function createTagsContainer(tags, isFocused, emitTagClick) {
  const tagsEl = document.createElement('div');
  tagsEl.className = isFocused ? 'tags focused' : 'tags';
  for (const tag of tags) {
    if (!tag) continue;
    const chip = document.createElement('span');
    chip.className = 'tag';
    chip.textContent = `#${tag}`;
    chip.addEventListener('click', (e) => {
      e.stopPropagation();
      if (emitTagClick) emitTagClick(tag);
    });
    tagsEl.appendChild(chip);
  }
  return tagsEl;
}

function createChevronSvg(points) {
  const ns = 'http://www.w3.org/2000/svg';
  const svg = document.createElementNS(ns, 'svg');
  svg.setAttribute('viewBox', '0 0 24 24');
  svg.setAttribute('width', '16');
  svg.setAttribute('height', '16');
  svg.setAttribute('fill', 'none');
  svg.setAttribute('stroke', 'currentColor');
  svg.setAttribute('stroke-width', '2.5');
  svg.setAttribute('stroke-linecap', 'round');
  svg.setAttribute('stroke-linejoin', 'round');
  const poly = document.createElementNS(ns, 'polyline');
  poly.setAttribute('points', points);
  svg.appendChild(poly);
  return svg;
}

function buildRenderSignature(card, isFocused) {
  const curioMeta = resolveCurioMeta(card);
  return JSON.stringify({
    isFocused: Boolean(isFocused),
    id: card?.id || '',
    title: card?.title || '',
    content: card?.content || '',
    images: Array.isArray(card?.images) ? card.images : [],
    tags: Array.isArray(card?.tags) ? card.tags : [],
    curioMeta,
  });
}

export function mount(container, initialProps, hostArg) {
  ensureStylesInjected();

  let currentProps = initialProps || {};
  let currentHost = hostArg || currentProps.host || {};
  let currentIndex = 0;
  let carouselTimer = null;
  let lastSignature = '';
  let rootEl = null;
  let timeEl = null;

  function getResolvedImages(props) {
    const raw = Array.isArray(props?.card?.images) ? props.card.images : [];
    return raw.map((img) => resolveAssetUrl(currentHost, img)).filter(Boolean);
  }

  function getDisplayTime(props) {
    return props?.displayTime || props?.thoughtTime || props?.babelTime || '';
  }

  function stopCarousel() {
    if (carouselTimer) {
      clearInterval(carouselTimer);
      carouselTimer = null;
    }
  }

  function applyScatteredCarouselActive(index) {
    if (!rootEl) return;
    const backdrops = rootEl.querySelectorAll('.backdrop-img');
    const stages = rootEl.querySelectorAll('.stage-img');
    const dots = rootEl.querySelectorAll('.image-stage .indicator-dot');
    backdrops.forEach((el, i) => el.classList.toggle('active', i === index));
    stages.forEach((el, i) => el.classList.toggle('active', i === index));
    dots.forEach((el, i) => el.classList.toggle('active', i === index));
  }

  function syncCarousel() {
    const images = getResolvedImages(currentProps);
    const isFocused = Boolean(currentProps?.isFocused);
    const isHovered = Boolean(currentProps?.isHovered);

    if (!isFocused && !isHovered && images.length > 1) {
      if (!carouselTimer) {
        carouselTimer = setInterval(() => {
          const imgs = getResolvedImages(currentProps);
          if (imgs.length <= 1) return;
          currentIndex = (currentIndex + 1) % imgs.length;
          applyScatteredCarouselActive(currentIndex);
        }, 3000);
      }
    } else {
      stopCarousel();
    }
  }

  function render() {
    if (!container) return;
    const props = currentProps;
    const card = props.card || {};
    const isFocused = Boolean(props.isFocused);
    const images = getResolvedImages(props);
    const curioMeta = resolveCurioMeta(card);
    const resolvedTime = getDisplayTime(props);
    const tags = Array.isArray(card.tags) ? card.tags : [];

    if (currentIndex >= images.length) {
      currentIndex = Math.max(0, images.length - 1);
    }

    container.innerHTML = '';
    rootEl = document.createElement('div');
    rootEl.className = isFocused ? 'finnegan-curio-card-body focused' : 'finnegan-curio-card-body';
    timeEl = null;

    if (!isFocused) {
      const topRow = document.createElement('div');
      topRow.className = 'scattering-top-row';
      if (curioMeta.platform) topRow.appendChild(createBadge(curioMeta.platform, 'badge platform-badge'));
      if (curioMeta.price) {
        const priceEl = document.createElement('span');
        priceEl.className = 'price';
        priceEl.textContent = curioMeta.price;
        topRow.appendChild(priceEl);
      }
      if (curioMeta.intent) topRow.appendChild(createIntentBadge(curioMeta.intent));
      if (curioMeta.availability) topRow.appendChild(createBadge(curioMeta.availability, 'badge availability-badge'));
      rootEl.appendChild(topRow);

      if (images.length > 0) {
        const stage = document.createElement('div');
        stage.className = 'image-stage';

        const backdrop = document.createElement('div');
        backdrop.className = 'album-glow-backdrop';
        images.forEach((src, i) => {
          const img = document.createElement('img');
          img.src = src;
          img.alt = '';
          img.className = i === currentIndex ? 'backdrop-img active' : 'backdrop-img';
          backdrop.appendChild(img);
        });
        stage.appendChild(backdrop);

        const viewport = document.createElement('div');
        viewport.className = 'stage-viewport';
        images.forEach((src, i) => {
          const img = document.createElement('img');
          img.src = src;
          img.alt = '';
          img.className = i === currentIndex ? 'stage-img active' : 'stage-img';
          viewport.appendChild(img);
        });
        stage.appendChild(viewport);

        if (images.length > 1) {
          const dots = document.createElement('div');
          dots.className = 'indicator-dots';
          images.forEach((_, i) => {
            const dot = document.createElement('button');
            dot.type = 'button';
            dot.className = i === currentIndex ? 'indicator-dot active' : 'indicator-dot';
            dot.addEventListener('click', (e) => {
              e.stopPropagation();
              currentIndex = i;
              applyScatteredCarouselActive(currentIndex);
            });
            dots.appendChild(dot);
          });
          stage.appendChild(dots);
        }

        rootEl.appendChild(stage);
      }

      const bottom = document.createElement('div');
      bottom.className = 'scattering-bottom';

      if (card.title) {
        const titleEl = document.createElement('div');
        titleEl.className = 'curio-title';
        titleEl.textContent = card.title;
        bottom.appendChild(titleEl);
      }

      if (card.content) {
        const fadeEl = document.createElement('div');
        fadeEl.className = 'content-fade';
        const mdEl = document.createElement('div');
        mdEl.className = 'curio-markdown-content';
        mdEl.innerHTML = renderSimpleMarkdown(card.content);
        fadeEl.appendChild(mdEl);
        bottom.appendChild(fadeEl);
      }

      if (tags.length > 0) {
        bottom.appendChild(createTagsContainer(tags, false, currentHost?.emitTagClick));
      }

      if (resolvedTime) {
        timeEl = document.createElement('div');
        timeEl.className = 'time';
        timeEl.textContent = resolvedTime;
        bottom.appendChild(timeEl);
      }

      rootEl.appendChild(bottom);
    } else {
      const header = document.createElement('div');
      header.className = 'focused-header';

      const metaRow = document.createElement('div');
      metaRow.className = 'meta-row';
      if (curioMeta.platform) metaRow.appendChild(createBadge(curioMeta.platform, 'badge platform-badge'));
      if (curioMeta.availability) metaRow.appendChild(createBadge(curioMeta.availability, 'badge availability-badge'));
      if (curioMeta.intent) metaRow.appendChild(createIntentBadge(curioMeta.intent));
      header.appendChild(metaRow);

      if (card.title) {
        const h2 = document.createElement('h2');
        h2.className = 'specimen-title';
        h2.textContent = card.title;
        header.appendChild(h2);
      }

      if (curioMeta.brand) {
        const brandEl = document.createElement('div');
        brandEl.className = 'brand';
        brandEl.textContent = curioMeta.brand;
        header.appendChild(brandEl);
      }

      if (curioMeta.price) {
        const priceEl = document.createElement('div');
        priceEl.className = 'price large';
        priceEl.textContent = curioMeta.price;
        header.appendChild(priceEl);
      }

      if (resolvedTime) {
        timeEl = document.createElement('div');
        timeEl.className = 'time';
        timeEl.textContent = resolvedTime;
        header.appendChild(timeEl);
      }

      rootEl.appendChild(header);

      if (images.length > 0) {
        const gallery = document.createElement('div');
        gallery.className = images.length > 1 ? 'card-image-gallery multi-images' : 'card-image-gallery';

        if (images.length === 1) {
          const singleWrap = document.createElement('div');
          singleWrap.className = 'single-image-wrapper';
          const img = document.createElement('img');
          img.src = images[0];
          img.alt = 'Curio specimen';
          img.className = 'gallery-image';
          img.addEventListener('error', () => { img.style.display = 'none'; });
          singleWrap.appendChild(img);
          gallery.appendChild(singleWrap);
        } else {
          const wrap = document.createElement('div');
          wrap.className = 'gallery-wrapper';

          const track = document.createElement('div');
          track.className = 'gallery-track';

          images.forEach((src, i) => {
            const slide = document.createElement('div');
            slide.className = 'gallery-slide';
            slide.setAttribute('role', 'group');
            slide.setAttribute('aria-label', `Image ${i + 1} of ${images.length}`);

            const img = document.createElement('img');
            img.src = src;
            img.alt = `Curio specimen ${i + 1}`;
            img.className = 'gallery-image';
            img.addEventListener('error', () => { img.style.display = 'none'; });
            slide.appendChild(img);
            track.appendChild(slide);
          });

          const prevBtn = document.createElement('button');
          prevBtn.type = 'button';
          prevBtn.className = 'gallery-nav-btn prev-btn';
          prevBtn.setAttribute('aria-label', 'Previous image');
          prevBtn.appendChild(createChevronSvg('15 18 9 12 15 6'));

          const nextBtn = document.createElement('button');
          nextBtn.type = 'button';
          nextBtn.className = 'gallery-nav-btn next-btn';
          nextBtn.setAttribute('aria-label', 'Next image');
          nextBtn.appendChild(createChevronSvg('9 18 15 12 9 6'));

          const dotsEl = document.createElement('div');
          dotsEl.className = 'indicator-dots';
          dotsEl.setAttribute('role', 'tablist');
          dotsEl.setAttribute('aria-label', 'Image gallery indicators');

          const dotButtons = images.map((_, i) => {
            const dot = document.createElement('button');
            dot.type = 'button';
            dot.className = 'indicator-dot';
            dot.setAttribute('aria-label', `View image ${i + 1} of ${images.length}`);
            dot.addEventListener('click', (e) => {
              e.stopPropagation();
              scrollToSlide(i);
            });
            dotsEl.appendChild(dot);
            return dot;
          });

          const syncGalleryControls = () => {
            prevBtn.classList.toggle('hidden', currentIndex === 0);
            nextBtn.classList.toggle('hidden', currentIndex >= images.length - 1);
            dotButtons.forEach((d, i) => d.classList.toggle('active', i === currentIndex));
          };

          const scrollToSlide = (index) => {
            const targetIdx = Math.max(0, Math.min(index, images.length - 1));
            currentIndex = targetIdx;
            syncGalleryControls();
            const width = track.clientWidth;
            if (width > 0 && typeof track.scrollTo === 'function') {
              track.scrollTo({ left: targetIdx * width, behavior: 'smooth' });
            }
          };

          track.addEventListener('scroll', () => {
            const width = track.clientWidth;
            if (width <= 0) return;
            const nextIdx = Math.round(track.scrollLeft / width);
            if (nextIdx !== currentIndex && nextIdx >= 0 && nextIdx < images.length) {
              currentIndex = nextIdx;
              syncGalleryControls();
            }
          });

          prevBtn.addEventListener('click', (e) => {
            e.stopPropagation();
            if (currentIndex > 0) scrollToSlide(currentIndex - 1);
          });

          nextBtn.addEventListener('click', (e) => {
            e.stopPropagation();
            if (currentIndex < images.length - 1) scrollToSlide(currentIndex + 1);
          });

          syncGalleryControls();
          wrap.appendChild(track);
          wrap.appendChild(prevBtn);
          wrap.appendChild(nextBtn);
          wrap.appendChild(dotsEl);
          gallery.appendChild(wrap);
        }

        rootEl.appendChild(gallery);
      }

      if (curioMeta.snapshotPath || curioMeta.screenshotPath || curioMeta.sourceUrl) {
        const actionBar = document.createElement('div');
        actionBar.className = 'archive-action-bar';

        if (curioMeta.snapshotPath) {
          const btn = document.createElement('button');
          btn.type = 'button';
          btn.textContent = '⚡ 打开本地网页快照';
          btn.addEventListener('click', (e) => {
            e.stopPropagation();
            currentHost?.openURL?.(curioMeta.snapshotPath);
          });
          actionBar.appendChild(btn);
        }
        if (curioMeta.screenshotPath) {
          const btn = document.createElement('button');
          btn.type = 'button';
          btn.textContent = '🖼️ 查看长截图';
          btn.addEventListener('click', (e) => {
            e.stopPropagation();
            currentHost?.openURL?.(curioMeta.screenshotPath);
          });
          actionBar.appendChild(btn);
        }
        if (curioMeta.sourceUrl) {
          const btn = document.createElement('button');
          btn.type = 'button';
          btn.textContent = '↗ 原始链接';
          btn.addEventListener('click', (e) => {
            e.stopPropagation();
            currentHost?.openURL?.(curioMeta.sourceUrl);
          });
          actionBar.appendChild(btn);
        }

        rootEl.appendChild(actionBar);
      }

      if (card.content) {
        const notesWrap = document.createElement('div');
        notesWrap.className = 'markdown-notes';
        const mdEl = document.createElement('div');
        mdEl.className = 'curio-markdown-content focused';
        mdEl.innerHTML = renderSimpleMarkdown(card.content);
        notesWrap.appendChild(mdEl);
        rootEl.appendChild(notesWrap);
      }

      if (tags.length > 0) {
        rootEl.appendChild(createTagsContainer(tags, true, currentHost?.emitTagClick));
      }
    }

    container.appendChild(rootEl);
  }

  lastSignature = buildRenderSignature(currentProps.card, currentProps.isFocused);
  render();
  syncCarousel();

  return {
    update(nextProps) {
      currentProps = nextProps || {};
      if (currentProps.host) {
        currentHost = currentProps.host;
      }
      const nextSignature = buildRenderSignature(currentProps.card, currentProps.isFocused);
      if (nextSignature !== lastSignature) {
        lastSignature = nextSignature;
        render();
      } else if (timeEl) {
        timeEl.textContent = getDisplayTime(currentProps);
      }
      syncCarousel();
    },
    destroy() {
      stopCarousel();
      if (container) {
        container.innerHTML = '';
      }
      rootEl = null;
      timeEl = null;
    },
  };
}

export default { mount };
