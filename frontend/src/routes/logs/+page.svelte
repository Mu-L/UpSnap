<script lang="ts">
	import { goto } from '$app/navigation';
	import { resolve } from '$app/paths';
	import { page } from '$app/state';
	import { m } from '$lib/paraglide/messages';
	import { pocketbase } from '$lib/stores/pocketbase';
	import {
		faArrowDown,
		faArrowUp,
		faArrowsRotate,
		faChevronDown,
		faChevronLeft,
		faChevronRight,
		faChevronUp,
		faFilter
	} from '@fortawesome/free-solid-svg-icons';
	import type { LogModel } from 'pocketbase';
	import { onMount } from 'svelte';
	import Fa from 'svelte-fa';
	import toast from 'svelte-french-toast';

	let logs = $state.raw<LogModel[]>([]);
	let currentPage = $state(1);
	let perPage = $state(20);
	let totalItems = $state(0);
	let totalPages = $state(0);
	let selectedLevel = $state('');
	let textSearch = $state('');
	let onlyDeviceLogs = $state(true);
	let sort = $state('-created');
	let intervalSeconds = $state(5);
	let loading = $state(true);
	let refreshing = $state(false);
	let reloadPending = false;
	let error = $state('');
	let lastUpdated = $state<Date>();
	let pollTimer: ReturnType<typeof setInterval> | undefined;
	let searchTimer: ReturnType<typeof setTimeout> | undefined;
	let detailsDialog: HTMLDialogElement;
	let selectedLog = $state<LogModel>();
	let mobileFiltersOpen = $state(false);

	const levels = [
		{ value: '', label: () => m.logs_level_all() },
		{ value: '8', label: () => 'ERROR' },
		{ value: '4', label: () => 'WARN' },
		{ value: '0', label: () => 'INFO' },
		{ value: '-4', label: () => 'DEBUG' }
	];

	function levelMeta(level: string) {
		switch (Number(level)) {
			case 8:
				return { label: 'ERROR', dotClass: 'bg-error' };
			case 4:
				return { label: 'WARN', dotClass: 'bg-warning' };
			case 0:
				return { label: 'INFO', dotClass: 'bg-success' };
			case -4:
				return { label: 'DEBUG', dotClass: 'bg-info' };
			default:
				return { label: 'UNKNOWN', dotClass: 'bg-base-content/60' };
		}
	}

	function hasData(entry: LogModel) {
		return entry.data && Object.keys(entry.data).length > 0;
	}

	function showDetails(entry: LogModel) {
		selectedLog = entry;
		detailsDialog.showModal();
	}

	function dataEntries(entry: LogModel) {
		return Object.entries(entry.data ?? {});
	}

	function dataValue(value: unknown) {
		if (value === '') return 'N/A';
		if (typeof value === 'object') return JSON.stringify(value);
		return String(value);
	}

	function updateSearch() {
		if (searchTimer) clearTimeout(searchTimer);
		searchTimer = setTimeout(() => updateQuery(), 250);
	}

	function toggleSort(field: 'created' | 'message') {
		sort = sort === field ? `-${field}` : field;
		void loadLogs();
	}

	async function loadLogs(initial = false, notifyError = false) {
		if (refreshing) {
			reloadPending = true;
			return;
		}

		refreshing = true;
		if (initial) loading = true;
		error = '';

		try {
			const filters = [];
			if (selectedLevel) filters.push(`level = ${selectedLevel}`);
			if (onlyDeviceLogs) filters.push('data.device != null');
			if (textSearch.trim()) {
				filters.push(
					$pocketbase.filter('(level ?~ {:search} || message ?~ {:search} || data ?~ {:search})', {
						search: textSearch.trim()
					})
				);
			}
			const result = await $pocketbase.logs.getList(currentPage, perPage, {
				sort,
				filter: filters.join(' && ')
			});
			if (result.totalPages > 0 && currentPage > result.totalPages) {
				currentPage = result.totalPages;
				reloadPending = true;
				return;
			}
			if (result.totalPages === 0) currentPage = 1;
			logs = result.items;
			totalItems = result.totalItems;
			totalPages = result.totalPages;
			lastUpdated = new Date();
		} catch (err) {
			const message = err instanceof Error ? err.message : String(err);
			error = m.logs_refresh_failed({ message });
			if (notifyError) toast.error(error);
		} finally {
			loading = false;
			refreshing = false;
			if (reloadPending) {
				reloadPending = false;
				void loadLogs();
			}
		}
	}

	function restartPolling() {
		if (pollTimer) clearInterval(pollTimer);
		pollTimer = undefined;

		if (intervalSeconds > 0) {
			pollTimer = setInterval(() => void loadLogs(), intervalSeconds * 1000);
		}
	}

	function updateQuery() {
		currentPage = 1;
		void loadLogs();
	}

	function changePage(nextPage: number) {
		currentPage = nextPage;
		void loadLogs();
	}

	onMount(() => {
		if (!$pocketbase.authStore.isSuperuser) {
			toast(m.toasts_no_permission({ url: page.url.pathname }), { icon: '⛔' });
			goto(resolve('/'));
			return;
		}

		void loadLogs(true);
		restartPolling();

		return () => {
			if (pollTimer) clearInterval(pollTimer);
			if (searchTimer) clearTimeout(searchTimer);
		};
	});
</script>

<main class="container mx-auto flex flex-col gap-4 p-4">
	<header class="flex flex-col justify-between gap-3 md:flex-row md:items-end">
		<div>
			<h1 class="text-2xl font-bold">{m.logs_page_title()}</h1>
			{#if lastUpdated}
				<p class="text-base-content/60 text-sm">
					{m.logs_refreshed_at({ time: lastUpdated.toLocaleTimeString() })}
				</p>
			{/if}
		</div>

		<div class="join w-full md:w-auto">
			<button
				class="btn btn-sm join-item flex-1 md:flex-none"
				onclick={() => loadLogs(false, true)}
				disabled={refreshing}
			>
				<Fa icon={faArrowsRotate} class={refreshing ? 'animate-spin' : ''} />
				{m.logs_refresh()}
			</button>
			<select
				class="select select-sm join-item"
				aria-label={m.logs_refresh()}
				bind:value={intervalSeconds}
				onchange={restartPolling}
			>
				<option value={0}>{m.logs_off()}</option>
				{#each [2, 5, 10, 30, 60] as seconds (seconds)}
					<option value={seconds}>{m.logs_seconds({ seconds })}</option>
				{/each}
			</select>
		</div>
	</header>

	<section class="xl:hidden">
		<button
			class="btn w-full justify-start gap-3"
			aria-expanded={mobileFiltersOpen}
			onclick={() => (mobileFiltersOpen = !mobileFiltersOpen)}
		>
			<Fa icon={faFilter} />
			<span class="text-base-content/60 flex-1 truncate text-left font-normal">
				{m.logs_filters()}
			</span>
			<Fa icon={mobileFiltersOpen ? faChevronUp : faChevronDown} />
		</button>

		{#if mobileFiltersOpen}
			<div class="card card-border bg-base-100 mt-2">
				<div class="card-body grid grid-cols-2 gap-3 p-3">
					<fieldset class="fieldset gap-1">
						<legend class="fieldset-legend">{m.logs_column_level()}</legend>
						<select
							class="select select-sm w-full"
							bind:value={selectedLevel}
							onchange={updateQuery}
						>
							{#each levels as level (level.value)}
								<option value={level.value}>{level.label()}</option>
							{/each}
						</select>
					</fieldset>

					<fieldset class="fieldset gap-1">
						<legend class="fieldset-legend">{m.logs_device_logs()}</legend>
						<label class="label gap-2 py-1">
							<input
								type="checkbox"
								class="toggle toggle-sm"
								bind:checked={onlyDeviceLogs}
								onchange={updateQuery}
							/>
							<span class="text-sm">{m.logs_only()}</span>
						</label>
					</fieldset>

					<fieldset class="fieldset gap-1">
						<legend class="fieldset-legend">{m.logs_page_size()}</legend>
						<select class="select select-sm w-full" bind:value={perPage} onchange={updateQuery}>
							<option value={10}>10</option>
							<option value={20}>20</option>
							<option value={40}>40</option>
							<option value={80}>80</option>
						</select>
					</fieldset>
				</div>
			</div>
		{/if}

		<input
			class="input mt-3 w-full"
			placeholder={m.logs_search_placeholder()}
			aria-label={m.logs_column_message()}
			bind:value={textSearch}
			oninput={updateSearch}
		/>
	</section>

	<section class="card card-border bg-base-100 hidden xl:block">
		<div class="flex items-end gap-3 p-4">
			<fieldset class="fieldset shrink-0">
				<legend class="fieldset-legend">{m.logs_column_level()}</legend>
				<select class="select w-auto" bind:value={selectedLevel} onchange={updateQuery}>
					{#each levels as level (level.value)}
						<option value={level.value}>{level.label()}</option>
					{/each}
				</select>
			</fieldset>

			<fieldset class="fieldset flex-1">
				<legend class="fieldset-legend">{m.logs_column_message()}</legend>
				<input
					class="input w-full"
					placeholder={m.logs_search_placeholder()}
					bind:value={textSearch}
					oninput={updateSearch}
				/>
			</fieldset>

			<fieldset class="fieldset flex! shrink-0 flex-col self-stretch">
				<legend class="fieldset-legend">{m.logs_device_logs()}</legend>
				<label class="label flex-1 gap-3">
					<input
						type="checkbox"
						class="toggle"
						bind:checked={onlyDeviceLogs}
						onchange={updateQuery}
					/>
					<span>{m.logs_only()}</span>
				</label>
			</fieldset>

			<fieldset class="fieldset shrink-0">
				<legend class="fieldset-legend">{m.logs_page_size()}</legend>
				<select class="select w-auto" bind:value={perPage} onchange={updateQuery}>
					<option value={10}>10</option>
					<option value={20}>20</option>
					<option value={40}>40</option>
					<option value={80}>80</option>
				</select>
			</fieldset>
		</div>
	</section>

	{#if error}
		<div role="alert" class="alert alert-error">
			<span>{error}</span>
		</div>
	{/if}

	<section class="card card-border bg-base-100 overflow-hidden">
		{#if loading}
			<div class="flex min-h-64 items-center justify-center">
				<span class="loading loading-spinner loading-lg"></span>
			</div>
		{:else if logs.length === 0}
			<div class="flex min-h-64 items-center justify-center p-6 text-center">
				<p class="text-base-content/60">{m.logs_empty()}</p>
			</div>
		{:else}
			<div class="hidden md:block">
				<div class="border-base-300 flex border-b text-sm font-bold">
					<div class="w-32 shrink-0 p-3">{m.logs_column_level()}</div>
					<button
						class="hover:bg-base-300 flex flex-1 items-center gap-1 p-3 text-left"
						onclick={() => toggleSort('message')}
					>
						{m.logs_column_message()}
						{#if sort === 'message'}<Fa icon={faArrowUp} />{:else if sort === '-message'}<Fa
								icon={faArrowDown}
							/>{/if}
					</button>
					<button
						class="hover:bg-base-300 flex w-40 shrink-0 items-center gap-1 p-3 text-left"
						onclick={() => toggleSort('created')}
					>
						{m.logs_column_time()}
						{#if sort === 'created'}<Fa icon={faArrowUp} />{:else if sort === '-created'}<Fa
								icon={faArrowDown}
							/>{/if}
					</button>
				</div>

				<div class="divide-base-300 divide-y">
					{#each logs as entry (entry.id)}
						{@const meta = levelMeta(entry.level)}
						<button
							class="hover:bg-base-200 flex w-full cursor-pointer items-start text-left"
							onclick={() => showDetails(entry)}
						>
							<div class="w-32 shrink-0 p-3">
								<span class="badge bg-base-300 badge-sm gap-1.5 border-0 font-semibold"
									><span class={`size-2 rounded-full ${meta.dotClass}`}></span>{meta.label}</span
								>
							</div>
							<div class="flex-1 p-3">
								<p class="text-xs wrap-break-word whitespace-pre-wrap">{entry.message}</p>
								{#if hasData(entry)}
									<div class="mt-1 flex flex-wrap gap-1">
										{#each dataEntries(entry) as [key, value] (key)}
											<span
												class="badge bg-base-300 badge-ghost badge-xs h-auto min-h-4 gap-1 py-0.5 font-normal whitespace-normal"
											>
												<span class="shrink-0">{key}:</span>
												<span>{dataValue(value)}</span>
											</span>
										{/each}
									</div>
								{/if}
							</div>
							<time class="w-40 shrink-0 p-3 text-xs whitespace-nowrap"
								>{new Date(entry.created).toLocaleString()}</time
							>
						</button>
					{/each}
				</div>
			</div>

			<ul class="divide-base-300 divide-y md:hidden">
				{#each logs as entry (entry.id)}
					{@const meta = levelMeta(entry.level)}
					<li>
						<button
							class="hover:bg-base-200 flex w-full cursor-pointer flex-col gap-2 p-4 text-left"
							onclick={() => showDetails(entry)}
						>
							<div class="flex items-center justify-between gap-3">
								<span class="badge bg-base-300 badge-sm gap-1.5 border-0 font-semibold"
									><span class={`size-2 rounded-full ${meta.dotClass}`}></span>{meta.label}</span
								>
								<time class="text-base-content/60 text-xs"
									>{new Date(entry.created).toLocaleString()}</time
								>
							</div>
							<p class="text-xs wrap-break-word whitespace-pre-wrap">{entry.message}</p>
							{#if hasData(entry)}
								<div class="flex flex-wrap gap-1">
									{#each dataEntries(entry) as [key, value] (key)}
										<span
											class="badge bg-base-300 badge-ghost badge-xs h-auto min-h-4 gap-1 py-0.5 font-normal whitespace-normal"
										>
											<span class="shrink-0">{key}:</span>
											<span>{dataValue(value)}</span>
										</span>
									{/each}
								</div>
							{/if}
						</button>
					</li>
				{/each}
			</ul>
		{/if}

		{#if totalPages > 1}
			<footer class="border-base-300 flex items-center justify-between border-t p-3">
				<span class="text-base-content/60 text-sm">{currentPage} / {totalPages} ({totalItems})</span
				>
				<div class="join">
					<button
						class="btn btn-sm join-item"
						onclick={() => changePage(currentPage - 1)}
						disabled={currentPage <= 1 || refreshing}
						aria-label={m.logs_previous()}
					>
						<Fa icon={faChevronLeft} />
					</button>
					<button
						class="btn btn-sm join-item"
						onclick={() => changePage(currentPage + 1)}
						disabled={currentPage >= totalPages || refreshing}
						aria-label={m.logs_next()}
					>
						<Fa icon={faChevronRight} />
					</button>
				</div>
			</footer>
		{/if}
	</section>
</main>

<dialog class="modal" bind:this={detailsDialog}>
	<div class="modal-box max-w-3xl">
		{#if selectedLog}
			{@const meta = levelMeta(selectedLog.level)}
			<h2 class="text-lg font-bold">{m.logs_column_data()}</h2>
			<dl class="mt-4 grid items-center gap-x-6 gap-y-2 text-xs sm:grid-cols-[auto_1fr]">
				<dt class="text-base-content/60">ID</dt>
				<dd class="break-all">{selectedLog.id}</dd>
				<dt class="text-base-content/60">{m.logs_column_level()}</dt>
				<dd>
					<span class="badge bg-base-300 badge-sm gap-1.5 border-0 font-semibold"
						><span class={`size-2 rounded-full ${meta.dotClass}`}></span>{meta.label}</span
					>
				</dd>
				<dt class="text-base-content/60">{m.logs_column_message()}</dt>
				<dd class="wrap-break-word whitespace-pre-wrap">{selectedLog.message}</dd>
				<dt class="text-base-content/60">{m.logs_column_time()}</dt>
				<dd>{new Date(selectedLog.created).toLocaleString()}</dd>
			</dl>
			<pre class="bg-base-200 rounded-box mt-4 max-h-96 overflow-auto p-4 text-xs">{JSON.stringify(
					selectedLog,
					null,
					2
				)}</pre>
		{/if}
		<div class="modal-action">
			<form method="dialog">
				<button class="btn">{m.buttons_cancel()}</button>
			</form>
		</div>
	</div>
	<form method="dialog" class="modal-backdrop"><button>close</button></form>
</dialog>
