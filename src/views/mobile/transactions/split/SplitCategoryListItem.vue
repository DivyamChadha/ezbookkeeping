<template>
    <f7-list-item
        class="list-item-with-header-and-title list-item-title-hide-overflow"
        link="#" no-chevron
        :class="{ 'disabled': disabled || !hasVisibleCategories }"
        :header="header"
        @click="showSheet = true"
    >
        <template #title>
            <div class="list-item-custom-title" v-if="modelValue && hasVisibleCategories">
                <span>{{ getTransactionPrimaryCategoryName(modelValue, categories) }}</span>
                <f7-icon class="category-separate-icon icon-with-direction" f7="chevron_right"></f7-icon>
                <span>{{ getTransactionSecondaryCategoryName(modelValue, categories) }}</span>
            </div>
            <div class="list-item-custom-title" v-else>
                <span>{{ tt('None') }}</span>
            </div>
        </template>
        <tree-view-selection-sheet primary-key-field="id" primary-title-field="name"
                                   primary-icon-field="icon" primary-icon-type-field="iconType" primary-icon-type="category" primary-color-field="color"
                                   primary-hidden-field="hidden" primary-sub-items-field="subCategories"
                                   secondary-key-field="id" secondary-value-field="id" secondary-title-field="name"
                                   secondary-icon-field="icon" secondary-icon-type-field="iconType" secondary-icon-type="category" secondary-color-field="color"
                                   secondary-hidden-field="hidden"
                                   :enable-filter="true" :filter-placeholder="tt('Find category')" :filter-no-items-text="tt('No available category')"
                                   :items="categories"
                                   v-model:show="showSheet"
                                   :model-value="modelValue"
                                   @update:model-value="emit('update:modelValue', $event as string)">
        </tree-view-selection-sheet>
    </f7-list-item>
</template>

<script setup lang="ts">
import { ref, computed } from 'vue';

import { useI18n } from '@/locales/helpers.ts';

import { useTransactionCategoriesStore } from '@/stores/transactionCategory.ts';

import type { CategoryType } from '@/core/category.ts';
import type { TransactionCategory } from '@/models/transaction_category.ts';

import {
    getFirstVisibleCategoryId,
    getTransactionPrimaryCategoryName,
    getTransactionSecondaryCategoryName
} from '@/lib/category.ts';

const props = defineProps<{
    header: string;
    categoryType: CategoryType;
    modelValue: string;
    disabled?: boolean;
}>();

const emit = defineEmits<{
    (e: 'update:modelValue', value: string): void;
}>();

const { tt } = useI18n();

const transactionCategoriesStore = useTransactionCategoriesStore();

const showSheet = ref<boolean>(false);

const categories = computed<TransactionCategory[]>(() => transactionCategoriesStore.allTransactionCategories[props.categoryType] || []);
const hasVisibleCategories = computed<boolean>(() => !!getFirstVisibleCategoryId(categories.value));
</script>
