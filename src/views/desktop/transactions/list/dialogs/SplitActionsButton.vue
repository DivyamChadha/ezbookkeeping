<template>
    <v-btn variant="tonal" :disabled="disabled" v-if="actions.length">
        {{ tt(transaction.type === TransactionType.Expense ? 'Split' : 'Mark as') }}
        <v-icon class="ms-1" :icon="mdiMenuDown" size="20" />
        <v-menu activator="parent">
            <v-list>
                <v-list-item :key="item.action" :title="tt(item.name)"
                             v-for="item in actions"
                             @click="openSplitDialog(item.action)"></v-list-item>
            </v-list>
        </v-menu>
    </v-btn>

    <split-dialog ref="splitDialog" />
</template>

<script setup lang="ts">
import SplitDialog from './SplitDialog.vue';

import { computed, useTemplateRef } from 'vue';

import { useI18n } from '@/locales/helpers.ts';
import {
    type TransactionSplitActionInfo,
    getAvailableTransactionSplitActions
} from '@/views/base/transactions/TransactionSplitDialogBase.ts';

import { TransactionType } from '@/core/transaction.ts';
import type { Transaction } from '@/models/transaction.ts';

import type { TransactionSplitAction } from '@/lib/transaction_split.ts';

import {
    mdiMenuDown
} from '@mdi/js';

type SplitDialogType = InstanceType<typeof SplitDialog>;

const props = defineProps<{
    transaction: Transaction;
    disabled?: boolean;
}>();

const emit = defineEmits<{
    (e: 'done', message: string): void;
}>();

const { tt } = useI18n();

const splitDialog = useTemplateRef<SplitDialogType>('splitDialog');

const actions = computed<TransactionSplitActionInfo[]>(() => getAvailableTransactionSplitActions(props.transaction));

function openSplitDialog(action: TransactionSplitAction): void {
    splitDialog.value?.open(props.transaction, action).then(message => {
        emit('done', message);
    }).catch(() => {
        // the dialog is closed without any change
    });
}
</script>
