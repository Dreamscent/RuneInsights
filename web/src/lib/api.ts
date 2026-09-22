// Typed client for the RuneInsights Go REST API.

export interface Player {
  id: number;
  name: string;
  accountType: string;
  intervalMin: number;
  createdAt: string;
  lastFetchedAt: string | null;
}

/** Next auto-computed milestone for a skill: 99, 110, 120 or the 200M cap. */
export interface MilestoneView {
  label: string;
  xp: number;
  remaining: number;
  pct: number;
  etaHours: number | null;
}

export interface SkillView {
  key: string;
  name: string;
  level: number;
  virtualLevel: number;
  xp: number;
  rank: number | null;
  xpIntoLevel: number;
  xpToNext: number;
  maxed: boolean;
  elite: boolean;
  ratePerDay: number;
  /** training-rate estimates from data/skill_rates.csv (0 = unknown -> no estimate) */
  xpPerHour: number;
  method: string;
  nextLevelEtaHours: number | null;
  next: MilestoneView;
}

export interface MilestoneCounts {
  skillsAt99: number;
  skillsAt110: number;
  skillsAt120: number;
  skillsAt200m: number;
  skillsAtLevelCap: number;
  total: number;
}

export interface SkillsResponse {
  player: Player;
  overall: SkillView;
  combatLevel: number;
  skills: SkillView[];
  snapshotAt: string | null;
  snapshotCount: number;
  collectingSince: string | null;
  ratesAvailable: boolean;
  milestoneCounts: MilestoneCounts;
  /** focused (pinned) skill keys in pin order */
  focus: string[];
  /** overall level reported by the hiscores */
  overall: SkillView;
  /** sum of every skill's in-game maximum level */
  maxTotalLevel: number;
}

export interface SkillGain {
  key: string;
  name: string;
  gain: number;
  ratePerDay: number;
  hasBaseline: boolean;
  /** XP gained since tracking began; fallback when a period has no baseline */
  trackingGain: number;
  /** current hiscore rank (null when unranked) */
  rank: number | null;
  /** latest vs baseline rank; negative = climbed */
  rankChange: number | null;
}

export interface Rates {
  trackingGain: number;
  trackingSince: string | null;
  trackingDays: number;
  period: string;
  from: string | null;
  to: string | null;
  elapsedHours: number;
  hasBaseline: boolean;
  overall: SkillGain;
  skills: SkillGain[];
}

export interface DailyGainPoint {
  /** YYYY-MM-DD (UTC) */
  date: string;
  xp: number;
  /** delta vs the previous retained day (-1 for the first point) */
  gain: number;
}

export interface Consistency {
  days: DailyGainPoint[];
  totalGained: number;
  activeDays: number;
  bestDate: string;
  bestGain: number;
  streak: number;
}

export interface HistoryPoint {
  at: string;
  xp: number;
}

export interface HistoryResponse {
  skill: string;
  from: string;
  to: string;
  points: HistoryPoint[];
}

export interface ActivityView {
  key: string;
  name: string;
  score: number;
  rank: number | null;
}

export interface   ActivitiesResponse {
  player: Player;
  activities: ActivityView[];
  /** activity key -> score increase over the requested period (present when a period is given) */
  gains?: Record<string, number>;
}

export interface LeaderboardRow {
  rank: number;
  name: string;
  score: number;
  level: number;
}

export interface LeaderboardResponse {
  table: string;
  category: string;
  rows: LeaderboardRow[];
}

export interface   ClanMember {
  name: string;
  /** clan rank ordinal: 0=Owner ... 7=Recruit */
  rank: number;
  overallXp: number;
  kills: number;
}

export interface ClanResponse {
  clan: string;
  members: ClanMember[];
}

export type Period = 'day' | 'week' | 'month' | 'year';
export type AccountType = 'normal' | 'ironman' | 'hardcore';

export interface RateEntry {
  perHour: number;
  method: string;
}

export interface SkillRateRow {
  skill: string;
  key: string;
  methods: RateEntry[];
}

export interface SkillRatesResponse {
  ratesAvailable: boolean;
  skills: SkillRateRow[];
}

export class ApiError extends Error {
  status: number;
  constructor(status: number, message: string) {
    super(message);
    this.status = status;
    this.name = 'ApiError';
  }
}

async function request<T>(path: string, init?: RequestInit): Promise<T> {
  let res: Response;
  try {
    res = await fetch(path, {
      headers: { Accept: 'application/json', ...(init?.body ? { 'Content-Type': 'application/json' } : {}) },
      ...init,
    });
  } catch (e) {
    throw new ApiError(0, 'Cannot reach the server. Is it running?');
  }

  if (res.status === 204) return undefined as T;

  const text = await res.text();
  let data: unknown = null;
  if (text) {
    try {
      data = JSON.parse(text);
    } catch {
      data = null;
    }
  }

  if (!res.ok) {
    const msg =
      data && typeof data === 'object' && 'error' in data
        ? String((data as { error: unknown }).error)
        : `Request failed (${res.status})`;
    throw new ApiError(res.status, msg);
  }

  return data as T;
}

export const api = {
  listPlayers: () => request<Player[] | null>('/api/players').then((p) => p ?? []),

  createPlayer: (name: string, accountType: AccountType = 'normal') =>
    request<Player>('/api/players', {
      method: 'POST',
      body: JSON.stringify({ name, accountType }),
    }),

  getPlayer: (id: number) => request<Player>(`/api/players/${id}`),

  updatePlayer: (id: number, patch: { accountType?: AccountType; intervalMin?: number }) =>
    request<Player>(`/api/players/${id}`, {
      method: 'PATCH',
      body: JSON.stringify(patch),
    }),

  deletePlayer: (id: number) =>
    request<void>(`/api/players/${id}`, { method: 'DELETE' }),

  refreshPlayer: (id: number) =>
    request<Player>(`/api/players/${id}/refresh`, { method: 'POST' }),

  skills: (id: number) => request<SkillsResponse>(`/api/players/${id}/skills`),

  rates: (id: number, period: Period) =>
    request<Rates>(`/api/players/${id}/rates?period=${period}`),

  history: (id: number, skill = 'overall', from?: string, to?: string) => {
    const q = new URLSearchParams({ skill });
    if (from) q.set('from', from);
    if (to) q.set('to', to);
    return request<HistoryResponse>(`/api/players/${id}/history?${q.toString()}`);
  },

  setFocus: (id: number, skills: string[]) =>
    request<{ focus: string[] }>(`/api/players/${id}/focus`, {
      method: 'PUT',
      body: JSON.stringify({ skills }),
    }),

  dailyGains: (id: number) =>
    request<{ player: Player; consistency: Consistency }>(`/api/players/${id}/daily-gains`),

  activities: (id: number, period?: Period) =>
    request<ActivitiesResponse>(
      `/api/players/${id}/activities${period ? `?period=${period}` : ''}`,
    ),


  leaderboard: (table = 'overall', category = 'skills') =>
    request<LeaderboardResponse>(
      `/api/leaderboard?table=${encodeURIComponent(table)}&category=${encodeURIComponent(category)}`,
    ),

  clan: (name: string) => request<ClanResponse>(`/api/clans/${encodeURIComponent(name)}`),

  skillRates: () =>
    request<SkillRatesResponse>('/api/skill-rates'),

  setSkillRate: (skill: string, methods: RateEntry[]) =>
    request<{ skill: string; methods: RateEntry[] }>(`/api/skill-rates/${encodeURIComponent(skill)}`, {
      method: 'PUT',
      body: JSON.stringify({ methods }),
    }),
};
