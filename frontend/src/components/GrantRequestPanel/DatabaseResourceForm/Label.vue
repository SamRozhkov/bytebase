<template>
  <div class="w-full flex items-center truncate gap-x-1">
    <span v-if="environment" class="text-gray-400">
      (<EnvironmentV1Name :environment="environment" :link="false" />)
    </span>
    <Prefix />
    <!-- eslint-disable-next-line vue/no-v-html -->
    <span v-html="optionName" />
    <span v-if="database" class="ml-1 text-gray-500 flex flex-row items-center">
      (<InstanceV1Name
        :instance="database.instanceResource"
        :link="false"
        class="whitespace-nowrap"
      />)
    </span>
  </div>
</template>

<script setup lang="ts">
import { computedAsync } from "@vueuse/core";
import { escape } from "lodash-es";
import { computed, h } from "vue";
import DatabaseIcon from "~icons/heroicons-outline/circle-stack";
import TableIcon from "~icons/heroicons-outline/table-cells";
import SchemaIcon from "~icons/heroicons-outline/view-columns";
import { EnvironmentV1Name, InstanceV1Name } from "@/components/v2";
import { useDatabaseV1Store } from "@/store";
import { getHighlightHTMLByRegExp } from "@/utils";
import type { DatabaseTreeOption } from "./common";

const props = defineProps<{
  option: DatabaseTreeOption;
  keyword?: string;
}>();

const option = computed(() => props.option);

const database = computedAsync(() => {
  const { option } = props;
  if (option.level !== "databases") return undefined;
  return useDatabaseV1Store().getOrFetchDatabaseByName(option.value);
});

const environment = computed(() => {
  const { option } = props;
  if (option.level !== "databases") return undefined;
  return database.value?.effectiveEnvironmentEntity;
});

const Prefix = () => {
  if (option.value.level === "databases") {
    return h(DatabaseIcon, {
      class: "w-4 h-auto text-gray-400 shrink-0",
    });
  } else if (option.value.level === "schemas") {
    return h(SchemaIcon, {
      class: "w-4 h-auto text-gray-400 shrink-0",
    });
  } else if (option.value.level === "tables") {
    return h(TableIcon, {
      class: "w-4 h-auto text-gray-400 shrink-0",
    });
  }
  return null;
};

const optionName = computed(() => {
  const name = option.value?.label ?? "";
  const keyword = (props.keyword ?? "").trim();

  return getHighlightHTMLByRegExp(
    escape(name),
    escape(keyword),
    false /* !caseSensitive */
  );
});
</script>
