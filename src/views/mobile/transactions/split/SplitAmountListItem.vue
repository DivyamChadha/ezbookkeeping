<template>
    <f7-list-item
        class="transaction-edit-amount"
        link="#" no-chevron
        :class="amountClass"
        :header="header"
        :title="formatAmountToLocalizedNumeralsWithCurrency(parseBigDecimal(modelValue), currency)"
        @click="showSheet = !readonly && !disabled"
    >
        <number-pad-sheet :min-value="0"
                          :max-value="TRANSACTION_MAX_AMOUNT"
                          :currency="currency"
                          v-model:show="showSheet"
                          :model-value="modelValue"
                          @update:model-value="emit('update:modelValue', $event)"
                          v-if="!readonly"
        ></number-pad-sheet>
    </f7-list-item>
</template>

<script setup lang="ts">
import { ref, computed } from 'vue';

import { useI18n } from '@/locales/helpers.ts';

import { TRANSACTION_MAX_AMOUNT } from '@/consts/transaction.ts';

import { parseBigDecimal } from '@/lib/numeral.ts';

const props = defineProps<{
    header: string;
    currency: string;
    modelValue: number;
    color?: 'expense' | 'income' | 'primary';
    readonly?: boolean;
    disabled?: boolean;
}>();

const emit = defineEmits<{
    (e: 'update:modelValue', value: number): void;
}>();

const { formatAmountToLocalizedNumeralsWithCurrency } = useI18n();

const showSheet = ref<boolean>(false);

const amountClass = computed<Record<string, boolean>>(() => {
    const classes: Record<string, boolean> = {
        'readonly': !!props.readonly,
        'disabled': !!props.disabled,
        'text-expense': props.color === 'expense',
        'text-income': props.color === 'income',
        'text-color-primary': props.color === 'primary',
        'ebk-normal-amount': true
    };

    return classes;
});
</script>
