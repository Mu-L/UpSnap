<script lang="ts">
	import { goto } from '$app/navigation';
	import { page } from '$app/state';
	import DeviceForm from '$lib/components/DeviceForm.svelte';
	import PageLoading from '$lib/components/PageLoading.svelte';
	import { m } from '$lib/paraglide/messages';
	import { permission, pocketbase } from '$lib/stores/pocketbase';
	import type { Device, Group, Port } from '$lib/types/device';
	import toast from 'svelte-french-toast';

	$: if (Object.hasOwn($permission, 'update')) {
		if (!$pocketbase.authStore.isSuperuser && !$permission.update.includes(page.params.id)) {
			toast(m.toasts_no_permission({ url: page.url.pathname }), {
				icon: '⛔'
			});
			goto('/');
		}
	}

	async function getDevice(): Promise<Device> {
		const resp = await $pocketbase
			.collection('devices')
			.getOne(page.params.id, { expand: 'ports,groups' });

		let device = resp as Device;

		if (!device.expand)
			device.expand = {} as {
				ports: Port[];
				groups: Group[];
			};

		if (!device.expand.ports) {
			device.expand.ports = [] as Port[];
		}

		return resp as Device;
	}
</script>

{#await getDevice()}
	<PageLoading />
{:then device}
	<h1 class="mb-8 text-3xl font-bold sm:break-all">{device.name}</h1>
	<DeviceForm {device} />
{:catch err}
	<div class="container mx-auto max-w-lg">
		{err}
	</div>
{/await}
