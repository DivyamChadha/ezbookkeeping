<template>
    <v-dialog width="720" :persistent="submitting" v-model="showState">
        <one-column-dialog-layout :title="tt(title)" :cancel-button-title="tt('Cancel')"
                                  :disabled="submitting" @cancel="cancel">
            <template #content>
                <div class="text-body-medium mb-6">
                    <span>{{ description }}</span>
                </div>

                <v-form>
                    <v-row v-if="action === TransactionSplitAction.SplitByCategories">
                        <v-col cols="12" md="7">
                            <two-column-select :disabled="submitting"
                                               :label="tt('Category of this transaction')" :placeholder="tt('Category')"
                                               v-bind="getCategorySelectProps(CategoryType.Expense, remainingCategoryId)"
                                               v-model="remainingCategoryId" />
                        </v-col>
                        <v-col cols="10" md="4">
                            <amount-input color="expense" :currency="currency" :show-currency="true" :readonly="true"
                                          :persistent-placeholder="true" :enable-rules="false"
                                          :label="tt('Amount kept in this transaction')" :placeholder="tt('Amount')"
                                          :model-value="remainingAmount" />
                        </v-col>
                        <v-col cols="2" md="1"></v-col>
                        <template :key="index" v-for="(item, index) in categoryItems">
                            <v-col cols="12" md="7">
                                <two-column-select :disabled="submitting"
                                                   :label="tt('Category of new expense')" :placeholder="tt('Category')"
                                                   v-bind="getCategorySelectProps(CategoryType.Expense, item.categoryId)"
                                                   v-model="item.categoryId" />
                            </v-col>
                            <v-col cols="10" md="4">
                                <amount-input color="expense" :currency="currency" :show-currency="true"
                                              :disabled="submitting" :persistent-placeholder="true" :enable-formula="true"
                                              :label="tt('Amount')" :placeholder="tt('Amount')"
                                              v-model="item.amount" />
                            </v-col>
                            <v-col cols="2" md="1" class="d-flex align-center justify-center">
                                <v-btn density="comfortable" color="default" variant="text" :icon="true"
                                       :aria-label="tt('Remove')" :disabled="submitting || categoryItems.length <= 1"
                                       @click="removeCategoryItem(index)">
                                    <v-icon :icon="mdiTrashCanOutline" size="20" />
                                    <v-tooltip activator="parent">{{ tt('Remove') }}</v-tooltip>
                                </v-btn>
                            </v-col>
                        </template>
                        <v-col cols="12">
                            <v-btn variant="tonal" density="comfortable" :prepend-icon="mdiPlus"
                                   :disabled="submitting || !canAddCategoryItem" @click="addCategoryItem">
                                {{ tt('Add another category') }}
                            </v-btn>
                        </v-col>
                    </v-row>

                    <v-row v-if="action === TransactionSplitAction.SplitWithPeople">
                        <v-col cols="12" md="6">
                            <v-text-field type="number" autocomplete="off" persistent-placeholder
                                          min="2" max="100" step="1"
                                          :disabled="submitting"
                                          :label="tt('Number of people, including you')"
                                          :model-value="peopleCount"
                                          @update:model-value="updatePeopleCount" />
                        </v-col>
                        <v-col cols="12" md="6">
                            <v-text-field type="text" autocomplete="off" persistent-placeholder
                                          maxlength="64"
                                          :disabled="submitting"
                                          :label="tt('Who owes you (optional)')"
                                          :placeholder="tt('Names, e.g. Alex, Sam')"
                                          v-model="note" />
                        </v-col>
                        <v-col cols="12" md="6">
                            <amount-input color="primary" :currency="currency" :show-currency="true"
                                          :disabled="submitting" :persistent-placeholder="true" :enable-formula="true"
                                          :label="tt('Others owe you')" :placeholder="tt('Amount')"
                                          v-model="othersAmount" />
                        </v-col>
                        <v-col cols="12" md="6">
                            <amount-input color="expense" :currency="currency" :show-currency="true" :readonly="true"
                                          :persistent-placeholder="true" :enable-rules="false"
                                          :label="tt('Your share')" :placeholder="tt('Amount')"
                                          :model-value="remainingAmount" />
                        </v-col>
                        <v-col cols="12">
                            <two-column-select :disabled="submitting"
                                               :label="tt('Category of your share')" :placeholder="tt('Category')"
                                               v-bind="getCategorySelectProps(CategoryType.Expense, remainingCategoryId)"
                                               v-model="remainingCategoryId" />
                        </v-col>
                        <v-col cols="12" md="6">
                            <v-select item-title="name" item-value="id" persistent-placeholder
                                      :disabled="submitting"
                                      :label="tt('Record what is owed in')"
                                      :items="receivableAccountOptions"
                                      v-model="receivableAccountId"
                                      v-if="receivableAccountOptions.length" />
                            <v-alert type="info" variant="tonal" density="compact" v-else>
                                {{ tt('A receivables account named "Due back" will be created to keep track of what others owe you') }}
                            </v-alert>
                        </v-col>
                        <v-col cols="12" md="6">
                            <two-column-select :disabled="submitting || !hasVisibleTransferCategories"
                                               :label="tt('Transfer category')" :placeholder="tt('Category')"
                                               v-bind="getCategorySelectProps(CategoryType.Transfer, transferCategoryId)"
                                               v-model="transferCategoryId" />
                        </v-col>
                    </v-row>

                    <v-row v-if="action === TransactionSplitAction.MarkAsRepayment">
                        <v-col cols="12">
                            <amount-input color="primary" :currency="currency" :show-currency="true" :readonly="true"
                                          :persistent-placeholder="true" :enable-rules="false"
                                          :label="tt('Amount')" :placeholder="tt('Amount')"
                                          :model-value="totalAmount" />
                        </v-col>
                        <v-col cols="12" md="6">
                            <v-select item-title="name" item-value="id" persistent-placeholder
                                      :disabled="submitting"
                                      :label="tt('Paid back from')"
                                      :items="receivableAccountOptions"
                                      v-model="receivableAccountId"
                                      v-if="receivableAccountOptions.length" />
                            <v-alert type="info" variant="tonal" density="compact" v-else>
                                {{ tt('A receivables account named "Due back" will be created to keep track of what others owe you') }}
                            </v-alert>
                        </v-col>
                        <v-col cols="12" md="6">
                            <two-column-select :disabled="submitting || !hasVisibleTransferCategories"
                                               :label="tt('Transfer category')" :placeholder="tt('Category')"
                                               v-bind="getCategorySelectProps(CategoryType.Transfer, transferCategoryId)"
                                               v-model="transferCategoryId" />
                        </v-col>
                        <v-col cols="12" v-if="isRepaymentMoreThanOwed">
                            <v-alert type="warning" variant="tonal" density="compact">
                                {{ tt('This amount is more than what is currently owed to you') }}
                            </v-alert>
                        </v-col>
                    </v-row>

                    <v-row v-if="action === TransactionSplitAction.MarkAsRefund">
                        <v-col cols="12">
                            <amount-input color="expense" :currency="currency" :show-currency="true" :readonly="true"
                                          :persistent-placeholder="true" :enable-rules="false"
                                          :label="tt('Amount')" :placeholder="tt('Amount')"
                                          :model-value="totalAmount" />
                        </v-col>
                        <v-col cols="12">
                            <two-column-select :disabled="submitting"
                                               :label="tt('Category of the refunded expense')" :placeholder="tt('Category')"
                                               v-bind="getCategorySelectProps(CategoryType.Expense, refundCategoryId)"
                                               v-model="refundCategoryId" />
                        </v-col>
                    </v-row>
                </v-form>
            </template>

            <template #footer>
                <v-tooltip :disabled="!inputProblemMessage" :text="inputProblemMessage ? tt(inputProblemMessage) : ''">
                    <template v-slot:activator="{ props }">
                        <div v-bind="props" class="d-inline-block">
                            <v-btn color="primary" :disabled="!!inputProblemMessage || submitting" @click="confirm">
                                {{ tt('Confirm') }}
                                <v-progress-circular indeterminate size="22" class="ms-2" v-if="submitting"></v-progress-circular>
                            </v-btn>
                        </div>
                    </template>
                </v-tooltip>
                <v-btn color="secondary" variant="tonal" :disabled="submitting" @click="cancel">{{ tt('Cancel') }}</v-btn>
            </template>
        </one-column-dialog-layout>
    </v-dialog>

    <snack-bar ref="snackbar" />
</template>

<script setup lang="ts">
import SnackBar from '@/components/desktop/SnackBar.vue';

import { ref, computed, useTemplateRef } from 'vue';

import { useI18n } from '@/locales/helpers.ts';
import { useTransactionSplitDialogBase } from '@/views/base/transactions/TransactionSplitDialogBase.ts';

import { CategoryType } from '@/core/category.ts';
import type { Transaction } from '@/models/transaction.ts';

import {
    getTransactionPrimaryCategoryName,
    getTransactionSecondaryCategoryName
} from '@/lib/category.ts';
import { TransactionSplitAction } from '@/lib/transaction_split.ts';

import {
    mdiPlus,
    mdiTrashCanOutline
} from '@mdi/js';

type SnackBarType = InstanceType<typeof SnackBar>;

const { tt } = useI18n();

const {
    action,
    submitting,
    remainingCategoryId,
    categoryItems,
    peopleCount,
    othersAmount,
    note,
    receivableAccountId,
    transferCategoryId,
    refundCategoryId,
    title,
    allCategories,
    hasVisibleTransferCategories,
    totalAmount,
    currency,
    receivableAccountOptions,
    remainingAmount,
    canAddCategoryItem,
    isRepaymentMoreThanOwed,
    inputProblemMessage,
    formatAmount,
    init,
    addCategoryItem,
    removeCategoryItem,
    splitEqually,
    submit
} = useTransactionSplitDialogBase();

const snackbar = useTemplateRef<SnackBarType>('snackbar');

let resolveFunc: ((message: string) => void) | null = null;
let rejectFunc: ((reason?: unknown) => void) | null = null;

const showState = ref<boolean>(false);

const description = computed<string>(() => {
    const amount = formatAmount(totalAmount.value);

    switch (action.value) {
        case TransactionSplitAction.SplitByCategories:
            return tt('Keep part of this {amount} expense in this transaction and move the rest to new expenses in other categories. The total amount and the account balance do not change.', { amount });
        case TransactionSplitAction.SplitWithPeople:
            return tt('You paid {amount} and others will pay you back. Only your share stays as your expense, the rest is recorded as money others owe you.', { amount });
        case TransactionSplitAction.MarkAsRepayment:
            return tt('This {amount} is money paid back to you, not income. It will reduce what others owe you instead.', { amount });
        case TransactionSplitAction.MarkAsRefund:
            return tt('This {amount} is a refund, not income. It will reduce the total expense of the category you choose.', { amount });
        default:
            return '';
    }
});

function getCategorySelectProps(categoryType: CategoryType, categoryId: string): Record<string, unknown> {
    const categories = allCategories.value[categoryType] || [];

    return {
        primaryKeyField: 'id',
        primaryValueField: 'id',
        primaryTitleField: 'name',
        primaryIconField: 'icon',
        primaryIconTypeField: 'iconType',
        primaryIconType: 'category',
        primaryColorField: 'color',
        primaryHiddenField: 'hidden',
        primarySubItemsField: 'subCategories',
        secondaryKeyField: 'id',
        secondaryValueField: 'id',
        secondaryTitleField: 'name',
        secondaryIconField: 'icon',
        secondaryIconTypeField: 'iconType',
        secondaryIconType: 'category',
        secondaryColorField: 'color',
        secondaryHiddenField: 'hidden',
        enableFilter: true,
        filterPlaceholder: tt('Find category'),
        filterNoItemsText: tt('No available category'),
        showSelectionPrimaryText: true,
        customSelectionPrimaryText: getTransactionPrimaryCategoryName(categoryId, categories),
        customSelectionSecondaryText: getTransactionSecondaryCategoryName(categoryId, categories),
        items: categories
    };
}

function updatePeopleCount(value: string | number): void {
    const count = typeof value === 'number' ? value : parseInt(value, 10);

    if (!Number.isFinite(count) || count < 2 || count > 100) {
        return;
    }

    peopleCount.value = Math.floor(count);
    splitEqually();
}

function open(transaction: Transaction, splitAction: TransactionSplitAction): Promise<string> {
    init(transaction, splitAction);
    showState.value = true;

    return new Promise((resolve, reject) => {
        resolveFunc = resolve;
        rejectFunc = reject;
    });
}

function confirm(): void {
    submit().then(message => {
        showState.value = false;
        resolveFunc?.(message);
    }).catch(error => {
        if (!error.processed) {
            snackbar.value?.showError(error);
        }
    });
}

function cancel(): void {
    if (submitting.value) {
        return;
    }

    showState.value = false;
    rejectFunc?.();
}

defineExpose({
    open
});
</script>
