<script setup lang="ts">
import PageHeader from '../components/ui/PageHeader.vue';
import SegmentedTabs from '../components/ui/SegmentedTabs.vue';
import RulesPanel from '../components/rules/RulesPanel.vue';
import RewritePanel from '../components/rules/RewritePanel.vue';
import PrivacyPanel from '../components/rules/PrivacyPanel.vue';
import { SUB_TABS } from '../constants/navigation';
import { useSubTab } from '../composables/useSubTab';

const props = defineProps<{ sub?: string }>();

const segments = SUB_TABS.rules ?? [];
const active = useSubTab(segments, () => props.sub, 'lists');
</script>

<template>
  <div class="pb-12 select-none">
    <PageHeader
      title="拦截规则"
      subtitle="基于 AdGuard 语法的规则引擎：订阅远程规则库、编写自定义规则与 DNS 重写伪装、配置协议隐私与 ECS 策略。"
    />

    <SegmentedTabs v-model="active" :segments="segments" />

    <RewritePanel v-if="active === 'rewrite'" />
    <PrivacyPanel v-else-if="active === 'privacy'" />
    <RulesPanel v-else v-model:sub="active" />
  </div>
</template>
