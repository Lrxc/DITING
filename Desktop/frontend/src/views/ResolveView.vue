<script setup lang="ts">
import PageHeader from '../components/ui/PageHeader.vue';
import SegmentedTabs from '../components/ui/SegmentedTabs.vue';
import UpstreamPanel from '../components/network/UpstreamPanel.vue';
import CachePanel from '../components/accel/CachePanel.vue';
import RewritePanel from '../components/rules/RewritePanel.vue';
import DNS64Panel from '../components/network/DNS64Panel.vue';
import { SUB_TABS } from '../constants/navigation';
import { useSubTab } from '../composables/useSubTab';

const props = defineProps<{ sub?: string }>();

const segments = SUB_TABS.resolve ?? [];
const active = useSubTab(segments, () => props.sub, 'upstream');
</script>

<template>
  <div class="pb-12 select-none">
    <PageHeader
      title="解析"
      subtitle="选择上游 DNS 与调度策略，叠加本地缓存加速、自定义域名重写，并为 IPv6-only 网络提供 DNS64 过渡合成。"
    />
    <SegmentedTabs v-model="active" :segments="segments" />

    <UpstreamPanel v-if="active === 'upstream'" />
    <CachePanel v-else-if="active === 'cache'" />
    <RewritePanel v-else-if="active === 'rewrite'" />
    <DNS64Panel v-else-if="active === 'dns64'" />
  </div>
</template>
