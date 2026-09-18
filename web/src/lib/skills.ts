// Deterministic, offline-safe visual identity for each skill.

const ABBREV: Record<string, string> = {
  overall: 'Σ',
  attack: 'At',
  defence: 'Df',
  strength: 'St',
  hitpoints: 'Hp',
  constitution: 'Hp',
  ranged: 'Ra',
  prayer: 'Pr',
  magic: 'Ma',
  cooking: 'Ck',
  woodcutting: 'Wc',
  fletching: 'Fl',
  fishing: 'Fi',
  firemaking: 'Fm',
  crafting: 'Cr',
  smithing: 'Sm',
  mining: 'Mi',
  herblore: 'He',
  agility: 'Ag',
  thieving: 'Th',
  slayer: 'Sl',
  farming: 'Fa',
  runecraft: 'Rc',
  hunter: 'Hu',
  construction: 'Co',
  summoning: 'Su',
  dungeoneering: 'Dg',
  divination: 'Dv',
  invention: 'In',
  archaeology: 'Ar',
  necromancy: 'Ne',
};

export function abbrev(key: string, name?: string): string {
  const k = key.toLowerCase();
  if (ABBREV[k]) return ABBREV[k];
  const src = (name || key).trim();
  return src ? src.slice(0, 2) : '?';
}

function hash(str: string): number {
  let h = 2166136261;
  for (let i = 0; i < str.length; i++) {
    h ^= str.charCodeAt(i);
    h = Math.imul(h, 16777619);
  }
  return h >>> 0;
}

export interface SkillColor {
  h: number;
  accent: string;
  accentHex: string;
  gradient: string;
  soft: string;
}

export function hslToHex(h: number, s = 70, l = 60): string {
  const sN = s / 100;
  const lN = l / 100;
  const c = (1 - Math.abs(2 * lN - 1)) * sN;
  const x = c * (1 - Math.abs(((h / 60) % 2) - 1));
  const m = lN - c / 2;
  let r = 0;
  let g = 0;
  let b = 0;
  if (h < 60) [r, g, b] = [c, x, 0];
  else if (h < 120) [r, g, b] = [x, c, 0];
  else if (h < 180) [r, g, b] = [0, c, x];
  else if (h < 240) [r, g, b] = [0, x, c];
  else if (h < 300) [r, g, b] = [x, 0, c];
  else [r, g, b] = [c, 0, x];
  const to = (v: number) =>
    Math.round((v + m) * 255)
      .toString(16)
      .padStart(2, '0');
  return `#${to(r)}${to(g)}${to(b)}`;
}

export function skillColor(key: string): SkillColor {
  const k = key.toLowerCase();
  if (k === 'overall') {
    return {
      h: 43,
      accent: '#f5a524',
      accentHex: '#f5a524',
      gradient: 'linear-gradient(140deg, #fbbf24, #b45309)',
      soft: 'rgba(245,165,36,0.16)',
    };
  }
  const h = hash(k) % 360;
  return {
    h,
    accent: `hsl(${h} 72% 62%)`,
    accentHex: hslToHex(h, 72, 62),
    gradient: `linear-gradient(140deg, hsl(${h} 70% 60%), hsl(${(h + 34) % 360} 68% 44%))`,
    soft: `hsl(${h} 70% 60% / 0.16)`,
  };
}
