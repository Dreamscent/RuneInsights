<script lang="ts">
  import { store } from '../lib/store.svelte';
  import { api, type SkillRateRow, type RateEntry } from '../lib/api';
  import SkillIcon from '../components/SkillIcon.svelte';
  import Spinner from '../components/Spinner.svelte';
  import EmptyState from '../components/EmptyState.svelte';
  import { compact, num } from '../lib/format';

  let rows = $state<SkillRateRow[]>([]);
  let ratesAvailable = $state(false);
  let loading = $state(true);
  let query = $state('');

  // rows being edited (key -> editable copy)
  let draft = $state<Record<string, RateEntry[]>>({});
  let saving = $state<Record<string, boolean>>({});
  let dirty = $state<Record<string, boolean>>({});
  let expanded = $state<Record<string, boolean>>({});

  async function load() {
    try {
      const resp = await api.skillRates();
      rows = resp.skills ?? [];
      ratesAvailable = resp.ratesAvailable;
      const d: Record<string, RateEntry[]> = {};
      for (const r of rows) d[r.key] = r.methods.map((m) => ({ ...m }));
      draft = d;
    } catch (e) {
      store.notify(e instanceof Error ? e.message : 'Failed to load rates', 'err');
    } finally {
      loading = false;
    }
  }

  // pre-expand skills that already have numbers, collapse empty ones
  $effect(() => {
    const rowsSnapshot = rows;
    for (const r of rowsSnapshot) {
      const has = r.methods.some((m) => m.perHour > 0);
      expanded[r.key] = has;
    }
  });

  const filtered = $derived.by(() => {
    const q = query.trim().toLowerCase();
    if (!q) return rows;
    return rows.filter((r) => r.skill.toLowerCase().includes(q));
  });

  function markDirty(key: string) {
    dirty[key] = true;
  }

  function addEntry(key: string) {
    const list = draft[key] ?? [];
    draft[key] = [...list, { perHour: 0, method: '' }];
    markDirty(key);
  }

  function removeEntry(key: string, i: number) {
    const list = [...(draft[key] ?? [])];
    list.splice(i, 1);
    draft[key] = list.length > 0 ? list : [{ perHour: 0, method: '' }];
    markDirty(key);
  }

  function moveEntry(key: string, i: number) {
    const list = [...(draft[key] ?? [])];
    if (i <= 0) return;
    [list[0], list[i]] = [list[i], list[0]];
    draft[key] = list;
    markDirty(key);
  }

  async function save(key: string) {
    if (saving[key]) return;
    saving[key] = true;
    try {
      const list = (draft[key] ?? []).map((m) => ({
        perHour: Math.max(0, Math.round(Number(m.perHour) || 0)),
        method: String(m.method ?? '').trim(),
      }));
      await api.setSkillRate(key, list);
      const resp = await api.skillRates();
      rows = resp.skills;
      draft[key] = (resp.skills.find((r) => r.key === key)?.methods ?? []).map((m) => ({ ...m }));
      dirty[key] = false;
      store.notify('Training rate saved', 'ok');
    } catch (e) {
      store.notify(e instanceof Error ? e.message : 'Failed to save rate', 'err');
    } finally {
      saving[key] = false;
    }
  }

  $effect(() => {
    void load();
  });

  const usedCount = $derived(rows.filter((r) => r.methods.some((m) => m.perHour > 0)).length);
</script>

<div class="flex flex-col gap-5">
  <div class="card p-4">
    <div class="flex flex-wrap items-start justify-between gap-3">
      <div>
        <h2 class="text-sm font-semibold text-[var(--color-ink)]">Training rates</h2>
        <p class="mt-1 max-w-xl text-xs text-[var(--color-muted)]">
          These values drive the XP-per-hour estimates for the next level and the next milestone.
          A skill can have several methods — the first entry is the default used everywhere.
          The file lives at <code class="text-[var(--color-gold-soft)]">data/skill_rates.csv</code>
          and can also be edited by hand.
        </p>
      </div>
      <span class="chip">{usedCount} / {rows.length} skills have rates</span>
    </div>
  </div>

  <input
    class="input max-w-64"
    placeholder="Filter skills…"
    bind:value={query}
    aria-label="Filter skills"
  />

  {#if loading}
    <div class="flex justify-center py-12"><Spinner size={22} /></div>
  {:else if rows.length === 0}
    <div class="card">
      <EmptyState title="No rate file yet" message="The file data/skill_rates.csv is missing or empty." />
    </div>
  {:else}
    <div class="flex flex-col gap-3">
      {#each filtered as r (r.key)}
        {@const entries = draft[r.key] ?? []}
        {@const hasRate = entries.some((m) => m.perHour > 0)}
        <div class="card overflow-hidden" class:border-[var(--color-line-2)]={dirty[r.key]}>
          <button
            class="flex w-full items-center gap-3 px-4 py-2.5 text-left transition-colors hover:bg-[var(--color-panel-2)]"
            onclick={() => (expanded[r.key] = !expanded[r.key])}
          >
            <SkillIcon skillKey={r.key} name={r.skill} size={28} rounded={7} dim={!hasRate} />
            <span class="text-sm font-medium text-[var(--color-ink)]">{r.skill}</span>
            {#if entries.length > 1}
              <span class="chip">{entries.length} methods</span>
            {/if}
            {#if hasRate}
              <span class="chip"><span class="text-[var(--color-jade)]">{compact(entries[0].perHour)}</span> xp/h</span>
              {#if entries[0].method}
                <span class="hidden truncate text-[11px] text-[var(--color-faint)] sm:inline">{entries[0].method}</span>
              {/if}
            {:else}
              <span class="chip text-[var(--color-faint)]">no estimate</span>
            {/if}
            <span class="flex-1"></span>
            {#if dirty[r.key]}
              <span class="chip text-[var(--color-gold)]">unsaved</span>
            {/if}
            <svg
              width="14"
              height="14"
              viewBox="0 0 24 24"
              fill="none"
              stroke="currentColor"
              stroke-width="2"
              stroke-linecap="round"
              stroke-linejoin="round"
              class="text-[var(--color-faint)] transition-transform {expanded[r.key] ? 'rotate-180' : ''}"
            >
              <path d="M6 9l6 6 6-6" />
            </svg>
          </button>

          {#if expanded[r.key]}
            <div class="border-t border-[var(--color-line)] bg-[var(--color-bg-soft)] p-3">
              <div class="mb-2 flex items-center gap-2 text-[10px] font-semibold tracking-widest text-[var(--color-faint)] uppercase">
                <span class="w-1/2">Training method</span>
                <span class="tabular w-36 text-right">XP / hour</span>
                <span class="w-16 text-right"></span>
              </div>
              <div class="flex flex-col gap-2">
                {#each entries as m, i (i)}
                  <div class="flex items-center gap-2">
                    <span class="chip shrink-0 {i === 0 ? 'text-[var(--color-gold)]' : ''}" title={i === 0 ? 'Default — used on the skills page' : 'Alt method'}>
                      {i === 0 ? 'default' : `method ${i + 1}`}
                    </span>
                    <input
                      class="input flex-1"
                      placeholder="e.g. Afk Croesus Front"
                      value={m.method}
                      oninput={(e) => { draft[r.key][i].method = (e.currentTarget as HTMLInputElement).value; markDirty(r.key); }}
                    />
                    <input
                      class="input tabular w-36"
                      inputmode="numeric"
                      value={m.perHour || ''}
                      oninput={(e) => { const v = Number((e.currentTarget as HTMLInputElement).value.replace(/[,\s]/g, '')); draft[r.key][i].perHour = Number.isFinite(v) && v >= 0 ? v : 0; markDirty(r.key); }}
                    />
                    <div class="flex gap-1">
                      {#if i > 0}
                        <button
                          class="btn-ghost rounded-md p-1.5"
                          title="Set as default (moves to top)"
                          onclick={() => moveEntry(r.key, i)}
                        >
                          <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="var(--color-gold)" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
                            <path d="M12 19V5M5 12l7-7 7 7" />
                          </svg>
                        </button>
                      {/if}
                      <button
                        class="btn-ghost rounded-md p-1.5"
                        title="Remove method"
                        onclick={() => removeEntry(r.key, i)}
                      >
                        <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="var(--color-rose)" stroke-width="2" stroke-linecap="round">
                          <path d="M4 7h16M10 11v6M14 11v6M6 7l1 13h10l1-13M9 7V4h6v3" />
                        </svg>
                      </button>
                    </div>
                  </div>
                {/each}
              </div>

              <div class="mt-3 flex items-center gap-2">
                <button class="btn" onclick={() => addEntry(r.key)}>
                  <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round">
                    <path d="M12 5v14M5 12h14" />
                  </svg>
                  Add method
                </button>
                <span class="flex-1"></span>
                <button class="btn btn-gold" disabled={saving[r.key]} onclick={() => void save(r.key)}>
                  {saving[r.key] ? 'Saving…' : 'Save'}
                </button>
              </div>
            </div>
          {/if}
        </div>
      {/each}
    </div>
  {/if}
</div>
