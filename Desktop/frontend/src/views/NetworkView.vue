<script setup lang="ts">
import PageHeader from '../components/ui/PageHeader.vue';
import SegmentedTabs from '../components/ui/SegmentedTabs.vue';
import AdaptersPanel from '../components/network/AdaptersPanel.vue';
import UpstreamPanel from '../components/network/UpstreamPanel.vue';
import LanDnsCard from '../components/LanDnsCard.vue';
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
      subtitle="接管物理网卡的 DNS 解析，管理上游服务器与调度策略，并向局域网开放独立 DNS 服务。"
    />
    <SegmentedTabs v-model="active" :segments="segments" />

    <AdaptersPanel v-if="active === 'adapters'" />
    <UpstreamPanel v-else-if="active === 'upstream'" />
    <LanDnsCard v-else />
  </div>
</template>
