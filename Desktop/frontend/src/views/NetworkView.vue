<script setup lang="ts">
import PageHeader from '../components/ui/PageHeader.vue';
import SegmentedTabs from '../components/ui/SegmentedTabs.vue';
import AdaptersPanel from '../components/network/AdaptersPanel.vue';
import UpstreamPanel from '../components/network/UpstreamPanel.vue';
import LanDnsCard from '../components/LanDnsCard.vue';
import ClientsPanel from '../components/network/ClientsPanel.vue';
import DNS64Panel from '../components/network/DNS64Panel.vue';
import { SUB_TABS } from '../constants/navigation';
import { useSubTab } from '../composables/useSubTab';

const props = defineProps<{ sub?: string }>();

const segments = SUB_TABS.network ?? [];
const active = useSubTab(segments, () => props.sub, 'adapters');
</script>

<template>
  <div class="pb-12 select-none">
    <PageHeader
      title="网络"
      subtitle="接管物理网卡的 DNS 解析，管理上游服务器与分流路由，管控局域网设备、访问权限与 IPv6 过渡合成。"
    />
    <SegmentedTabs v-model="active" :segments="segments" />

    <AdaptersPanel v-if="active === 'adapters'" />
    <UpstreamPanel v-else-if="active === 'upstream'" />
    <LanDnsCard v-else-if="active === 'lan'" />
    <ClientsPanel v-else-if="active === 'clients'" />
    <DNS64Panel v-else-if="active === 'dns64'" />
  </div>
</template>
