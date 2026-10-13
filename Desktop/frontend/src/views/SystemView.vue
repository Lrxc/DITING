<script setup lang="ts">
import PageHeader from '../components/ui/PageHeader.vue';
import SegmentedTabs from '../components/ui/SegmentedTabs.vue';
import AppearancePanel from '../components/system/AppearancePanel.vue';
import ServicePanel from '../components/system/ServicePanel.vue';
import LanWebCard from '../components/LanWebCard.vue';
import DiagnosticsPanel from '../components/system/DiagnosticsPanel.vue';
import { SUB_TABS } from '../constants/navigation';
import { useSubTab } from '../composables/useSubTab';

const props = defineProps<{ sub?: string }>();

const segments = SUB_TABS.system ?? [];
const active = useSubTab(segments, () => props.sub, 'appearance');
</script>

<template>
  <div class="pb-12 select-none">
    <PageHeader
      title="系统"
      subtitle="配置外观主题、后台服务连接与开机自启，开启局域网 Web 远程控制台，并在网络异常时一键诊断与应急恢复。"
    />
    <SegmentedTabs v-model="active" :segments="segments" />

    <AppearancePanel v-if="active === 'appearance'" />
    <ServicePanel v-else-if="active === 'service'" />
    <LanWebCard v-else-if="active === 'lanweb'" />
    <DiagnosticsPanel v-else-if="active === 'diagnostics'" />
  </div>
</template>
