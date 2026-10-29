import type { CommandDef } from '../types/card';

/**
 * Normalizes command trigger by removing leading slashes and converting to lowercase.
 */
export function normalizeTrigger(trigger: string = ''): string {
  return trigger.replace(/^\/+/, '').trim().toLowerCase();
}

/**
 * Returns the module-specific icon for a command.
 * - 'bsky' -> 🦋
 * - 'bugu' -> 🕊️
 * - Default -> 🔭
 */
export function getCommandIcon(cmd: CommandDef | null | undefined): string {
  if (!cmd) return '🔭';
  const ns = (cmd.namespace || '').toLowerCase();
  if (ns === 'bsky') return '🦋';
  if (ns === 'bugu') return '🕊️';
  return '🔭';
}

/**
 * Filters and ranks available commands based on user query string.
 * Query should already be stripped of leading '/'.
 * 
 * Ranking criteria:
 * 1. Exact trigger match
 * 2. Same-namespace group: primary commands always precede subcommands
 * 3. Starts-with trigger match
 * 4. Namespace prefix match
 * 5. General primary command preference
 * 6. Lexicographical order
 */
export function filterAndRankCommands(
  availableCommands: CommandDef[] = [],
  query: string = ''
): CommandDef[] {
  const cleanQuery = query.trim().toLowerCase();

  const filtered = availableCommands.filter(cmd => {
    if (!cleanQuery) return true;
    const trig = normalizeTrigger(cmd.trigger);
    const name = (cmd.name || '').toLowerCase();
    const desc = (cmd.description || '').toLowerCase();
    const ns = (cmd.namespace || '').toLowerCase();
    const sub = (cmd.subCommand || '').toLowerCase();
    return (
      trig.includes(cleanQuery) ||
      name.includes(cleanQuery) ||
      desc.includes(cleanQuery) ||
      ns.includes(cleanQuery) ||
      sub.includes(cleanQuery)
    );
  });

  return filtered.sort((a, b) => {
    const trigA = normalizeTrigger(a.trigger);
    const trigB = normalizeTrigger(b.trigger);
    const nsA = (a.namespace || '').toLowerCase();
    const nsB = (b.namespace || '').toLowerCase();

    // 1. Exact trigger match
    if (trigA === cleanQuery && trigB !== cleanQuery) return -1;
    if (trigB === cleanQuery && trigA !== cleanQuery) return 1;

    // 2. Namespace grouping: If both share the same namespace, primary comes first
    if (nsA && nsB && nsA === nsB) {
      if (a.isPrimary && !b.isPrimary) return -1;
      if (!a.isPrimary && b.isPrimary) return 1;
    }

    // 3. Trigger starts with query
    const aStarts = trigA.startsWith(cleanQuery);
    const bStarts = trigB.startsWith(cleanQuery);
    if (aStarts && !bStarts) return -1;
    if (!aStarts && bStarts) return 1;

    // 4. Namespace matches query exactly or starts with query
    const aNsStarts = nsA.startsWith(cleanQuery);
    const bNsStarts = nsB.startsWith(cleanQuery);
    if (aNsStarts && !bNsStarts) return -1;
    if (!aNsStarts && bNsStarts) return 1;

    // 5. Global primary flag preference
    if (a.isPrimary && !b.isPrimary) return -1;
    if (!a.isPrimary && b.isPrimary) return 1;

    // 6. Alphabetical
    return trigA.localeCompare(trigB);
  });
}

/**
 * Resolves typed input into a command and remainder parameter text.
 * Supports both standard `/trigger param` and modular space-separated `/namespace subcommand param`.
 * 
 * Example:
 * - "/bsky push my-card" -> resolves to subCommand "bsky-push" with restText "my-card"
 * - "/bsky-push my-card" -> resolves to command "bsky-push" with restText "my-card"
 * - "/bsky my-card" -> resolves to primary command "bsky" with restText "my-card"
 */
export function resolveCommandInput(
  rawInput: string,
  availableCommands: CommandDef[] = []
): { command: CommandDef; restText: string } | null {
  const cleanInput = rawInput.replace(/^\/+/, '').trimStart();
  const spaceIdx = cleanInput.indexOf(' ');
  if (spaceIdx <= 0) return null;

  const firstWord = cleanInput.slice(0, spaceIdx).trim().toLowerCase();
  const remainder = cleanInput.slice(spaceIdx + 1).trim();
  const remWords = remainder.split(/\s+/);
  const secondWord = remWords[0]?.toLowerCase() || '';

  // 1. Check for space-separated namespace + subcommand (e.g. "bsky push")
  if (secondWord) {
    const subCmd = availableCommands.find(c => {
      const ns = (c.namespace || '').toLowerCase();
      const sub = (c.subCommand || '').toLowerCase();
      return ns === firstWord && sub === secondWord;
    });

    if (subCmd) {
      const restText = remWords.slice(1).join(' ').trim();
      return { command: subCmd, restText };
    }
  }

  // 2. Check for exact trigger match on firstWord (e.g. "bsky" or "bsky-push")
  const directCmd = availableCommands.find(c => normalizeTrigger(c.trigger) === firstWord);
  if (directCmd) {
    return { command: directCmd, restText: remainder };
  }

  return null;
}

/**
 * Matches an exact command string against trigger or "namespace subcommand".
 * Used for Space key triggers when user has typed the command without trailing parameters yet.
 */
export function matchExactCommand(
  rawInput: string,
  availableCommands: CommandDef[] = []
): CommandDef | null {
  const clean = rawInput.replace(/^\/+/, '').trim().toLowerCase();
  if (!clean) return null;

  // Direct trigger match
  const direct = availableCommands.find(c => normalizeTrigger(c.trigger) === clean);
  if (direct) return direct;

  // Space-separated "namespace subcommand" match (e.g. "bsky push")
  const parts = clean.split(/\s+/);
  if (parts.length === 2) {
    const [ns, sub] = parts;
    const subCmd = availableCommands.find(c => {
      const cNs = (c.namespace || '').toLowerCase();
      const cSub = (c.subCommand || '').toLowerCase();
      return cNs === ns && cSub === sub;
    });
    if (subCmd) return subCmd;
  }

  return null;
}
