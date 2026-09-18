import { api, ApiError, type AccountType, type Player, type SkillsResponse } from './api';

export type ViewKey =
  | 'overview'
  | 'skills'
  
  | 'progress'
  | 'bosses'
  | 'leaderboard'
  | 'clan'
  | 'settings'
  | 'rates';

export interface Toast {
  id: number;
  message: string;
  kind: 'ok' | 'err' | 'info';
}

const LS_KEY = 'runeinsights:selected-player';

class Store {
  players = $state<Player[]>([]);
  selectedId = $state<number | null>(null);
  skills = $state<SkillsResponse | null>(null);
  view = $state<ViewKey>('overview');

  booting = $state(true);
  skillsLoading = $state(false);
  mutating = $state(false);
  globalError = $state<string | null>(null);

  toasts = $state<Toast[]>([]);
  private toastId = 0;
  private pollTimer: ReturnType<typeof setInterval> | null = null;

  get selected(): Player | null {
    return this.players.find((p) => p.id === this.selectedId) ?? null;
  }

  async boot() {
    this.booting = true;
    this.globalError = null;
    try {
      this.players = await api.listPlayers();
      const saved = Number(localStorage.getItem(LS_KEY));
      const pick =
        this.players.find((p) => p.id === saved)?.id ?? this.players[0]?.id ?? null;
      this.selectedId = pick;
      if (pick != null) await this.loadSkills(pick);
    } catch (e) {
      this.globalError = errMessage(e);
    } finally {
      this.booting = false;
      this.startPolling();
    }
  }

  private startPolling() {
    if (this.pollTimer) clearInterval(this.pollTimer);
    this.pollTimer = setInterval(() => {
      if (document.visibilityState === 'visible' && this.selectedId != null && this.skills) {
        void this.loadSkills(this.selectedId, true);
      }
    }, 60_000);
  }

  async select(id: number) {
    if (this.selectedId === id) return;
    this.selectedId = id;
    localStorage.setItem(LS_KEY, String(id));
    this.skills = null;
    await this.loadSkills(id);
  }

  async loadSkills(id: number, silent = false) {
    if (!silent) this.skillsLoading = true;
    try {
      const data = await api.skills(id);
      if (this.selectedId === id) this.skills = data;
    } catch (e) {
      if (!silent) this.notify(errMessage(e), 'err');
    } finally {
      this.skillsLoading = false;
    }
  }

  async reloadSkills() {
    if (this.selectedId != null) await this.loadSkills(this.selectedId);
  }

  async addPlayer(name: string, accountType: AccountType): Promise<boolean> {
    this.mutating = true;
    try {
      const p = await api.createPlayer(name, accountType);
      this.players = await api.listPlayers();
      this.notify(`${p.name} is now being tracked`, 'ok');
      await this.select(p.id);
      return true;
    } catch (e) {
      this.notify(errMessage(e), 'err');
      return false;
    } finally {
      this.mutating = false;
    }
  }

  async refreshSelected(): Promise<boolean> {
    if (this.selectedId == null) return false;
    this.mutating = true;
    try {
      await api.refreshPlayer(this.selectedId);
      this.players = await api.listPlayers();
      await this.reloadSkills();
      this.notify('Snapshot recorded', 'ok');
      return true;
    } catch (e) {
      this.notify(errMessage(e), 'err');
      return false;
    } finally {
      this.mutating = false;
    }
  }

  async updateSelected(patch: { accountType?: AccountType; intervalMin?: number }): Promise<boolean> {
    if (this.selectedId == null) return false;
    this.mutating = true;
    try {
      await api.updatePlayer(this.selectedId, patch);
      this.players = await api.listPlayers();
      if (patch.accountType) await this.reloadSkills();
      this.notify('Settings saved', 'ok');
      return true;
    } catch (e) {
      this.notify(errMessage(e), 'err');
      return false;
    } finally {
      this.mutating = false;
    }
  }

  async removeSelected(): Promise<boolean> {
    if (this.selectedId == null) return false;
    const name = this.selected?.name ?? 'this player';
    this.mutating = true;
    try {
      await api.deletePlayer(this.selectedId);
      this.players = await api.listPlayers();
      this.skills = null;
      const next = this.players[0]?.id ?? null;
      this.selectedId = next;
      if (next != null) {
        localStorage.setItem(LS_KEY, String(next));
        await this.loadSkills(next);
      } else {
        localStorage.removeItem(LS_KEY);
      }
      this.notify(`${name} removed`, 'ok');
      return true;
    } catch (e) {
      this.notify(errMessage(e), 'err');
      return false;
    } finally {
      this.mutating = false;
    }
  }

  notify(message: string, kind: Toast['kind'] = 'info') {
    const id = ++this.toastId;
    this.toasts = [...this.toasts, { id, message, kind }];
    setTimeout(() => this.dismiss(id), 4200);
  }

  dismiss(id: number) {
    this.toasts = this.toasts.filter((t) => t.id !== id);
  }
}

export function errMessage(e: unknown): string {
  if (e instanceof ApiError) return e.message;
  if (e instanceof Error) return e.message;
  return 'Something went wrong';
}

export const store = new Store();
