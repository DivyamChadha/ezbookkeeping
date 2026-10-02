<template>
    <f7-popup push swipe-to-close :opened="show" @popup:open="onPopupOpen" @popup:closed="onPopupClosed">
        <f7-page>
            <f7-navbar>
                <f7-nav-left>
                    <f7-link icon-f7="xmark" :class="{ 'disabled': submitting }" :aria-label="tt('Cancel')" @click="close"></f7-link>
                </f7-nav-left>
                <f7-nav-title :title="tt(title)"></f7-nav-title>
                <f7-nav-right>
                    <f7-link icon-f7="checkmark_alt" :class="{ 'disabled': submitting || !transaction }" :aria-label="tt('Confirm')" @click="confirm"></f7-link>
                </f7-nav-right>
            </f7-navbar>

            <f7-block class="margin-vertical-half">
                <p class="no-margin">{{ description }}</p>
            </f7-block>

            <template v-if="action === TransactionSplitAction.SplitByCategories">
                <f7-list strong inset dividers class="margin-vertical-half">
                    <template #list>
                        <split-category-list-item :header="tt('Category of this transaction')" :category-type="CategoryType.Expense"
                                                  :disabled="submitting" v-model="remainingCategoryId" />
                        <split-amount-list-item color="expense" :readonly="true" :currency="currency"
                                                :header="tt('Amount kept in this transaction')" :model-value="remainingAmount" />
                    </template>
                </f7-list>

                <f7-list strong inset dividers class="margin-vertical-half" :key="index" v-for="(item, index) in categoryItems">
                    <template #list>
                        <split-category-list-item :header="tt('Category of new expense')" :category-type="CategoryType.Expense"
                                                  :disabled="submitting" v-model="item.categoryId" />
                        <split-amount-list-item color="expense" :currency="currency" :disabled="submitting"
                                                :header="tt('Amount')" v-model="item.amount" />
                        <f7-list-button color="red" :class="{ 'disabled': submitting }"
                                        @click="removeCategoryItem(index)" v-if="categoryItems.length > 1">{{ tt('Remove') }}</f7-list-button>
                    </template>
                </f7-list>

                <f7-list strong inset dividers class="margin-vertical-half">
                    <template #list>
                        <f7-list-button :class="{ 'disabled': submitting || !canAddCategoryItem }" @click="addCategoryItem">{{ tt('Add another category') }}</f7-list-button>
                    </template>
                </f7-list>
            </template>

            <f7-list strong inset dividers class="margin-vertical-half" v-if="action === TransactionSplitAction.SplitWithPeople">
                <template #list>
                    <f7-list-item :title="tt('Number of people, including you')">
                        <template #after>
                            <div class="display-flex align-items-center">
                                <f7-link icon-f7="minus_circle" :aria-label="tt('Decrease')"
                                         :class="{ 'disabled': submitting || peopleCount <= MIN_PEOPLE_COUNT }"
                                         @click="changePeopleCount(-1)"></f7-link>
                                <span class="split-people-count">{{ peopleCount }}</span>
                                <f7-link icon-f7="plus_circle" :aria-label="tt('Increase')"
                                         :class="{ 'disabled': submitting || peopleCount >= MAX_PEOPLE_COUNT }"
                                         @click="changePeopleCount(1)"></f7-link>
                            </div>
                        </template>
                    </f7-list-item>
                    <split-amount-list-item color="primary" :currency="currency" :disabled="submitting"
                                            :header="tt('Others owe you')" v-model="othersAmount" />
                    <split-amount-list-item color="expense" :readonly="true" :currency="currency"
                                            :header="tt('Your share')" :model-value="remainingAmount" />
                    <split-category-list-item :header="tt('Category of your share')" :category-type="CategoryType.Expense"
                                              :disabled="submitting" v-model="remainingCategoryId" />
                    <f7-list-input type="text" autocomplete="off" clear-button
                                   maxlength="64"
                                   :disabled="submitting"
                                   :label="tt('Who owes you (optional)')"
                                   :placeholder="tt('Names, e.g. Alex, Sam')"
                                   v-model:value="note"></f7-list-input>
                </template>
            </f7-list>

            <f7-list strong inset dividers class="margin-vertical-half" v-if="action === TransactionSplitAction.MarkAsRepayment || action === TransactionSplitAction.MarkAsRefund">
                <template #list>
                    <split-amount-list-item :color="action === TransactionSplitAction.MarkAsRefund ? 'expense' : 'primary'"
                                            :readonly="true" :currency="currency"
                                            :header="tt('Amount')" :model-value="totalAmount" />
                    <split-category-list-item :header="tt('Category of the refunded expense')" :category-type="CategoryType.Expense"
                                              :disabled="submitting" v-model="refundCategoryId"
                                              v-if="action === TransactionSplitAction.MarkAsRefund" />
                </template>
            </f7-list>

            <f7-list strong inset dividers class="margin-vertical-half" v-if="action === TransactionSplitAction.SplitWithPeople || action === TransactionSplitAction.MarkAsRepayment">
                <template #list>
                    <f7-list-item class="list-item-with-header-and-title list-item-title-hide-overflow"
                                  link="#" no-chevron
                                  :class="{ 'disabled': submitting }"
                                  :header="tt(action === TransactionSplitAction.MarkAsRepayment ? 'Paid back from' : 'Record what is owed in')"
                                  :title="selectedReceivableAccountName"
                                  @click="showReceivableAccountSheet = true"
                                  v-if="receivableAccountOptions.length">
                        <list-item-selection-sheet value-type="item"
                                                   key-field="id" value-field="id" title-field="name"
                                                   :items="receivableAccountOptions"
                                                   v-model:show="showReceivableAccountSheet"
                                                   v-model="receivableAccountId">
                        </list-item-selection-sheet>
                    </f7-list-item>
                    <f7-list-item v-else>
                        <template #title>
                            <span>{{ tt('Due back') }}</span>
                        </template>
                        <template #footer>
                            <span>{{ tt('A receivables account named "Due back" will be created to keep track of what others owe you') }}</span>
                        </template>
                    </f7-list-item>
                    <split-category-list-item :header="tt('Transfer category')" :category-type="CategoryType.Transfer"
                                              :disabled="submitting" v-model="transferCategoryId" />
                </template>
            </f7-list>

            <f7-block class="margin-vertical-half text-color-orange" v-if="isRepaymentMoreThanOwed">
                <p class="no-margin">{{ tt('This amount is more than what is currently owed to you') }}</p>
            </f7-block>

            <f7-block class="margin-vertical-half text-color-gray" v-if="transaction && inputProblemMessage">
                <p class="no-margin">{{ tt(inputProblemMessage) }}</p>
            </f7-block>
        </f7-page>
    </f7-popup>
</template>

<script setup lang="ts">
import SplitCategoryListItem from './SplitCategoryListItem.vue';
import SplitAmountListItem from './SplitAmountListItem.vue';

import { ref, computed, watch } from 'vue';

import { useI18n } from '@/locales/helpers.ts';
import { useI18nUIComponents, showLoading, hideLoading } from '@/lib/ui/mobile.ts';
import { useTransactionSplitDialogBase } from '@/views/base/transactions/TransactionSplitDialogBase.ts';

import { CategoryType } from '@/core/category.ts';
import type { Transaction } from '@/models/transaction.ts';

import { TransactionSplitAction } from '@/lib/transaction_split.ts';

const props = defineProps<{
    show: boolean;
    currentTransaction: Transaction | null;
    splitAction: TransactionSplitAction;
}>();

const emit = defineEmits<{
    (e: 'update:show', value: boolean): void;
    (e: 'done', message: string): void;
}>();

const { tt } = useI18n();
const { showAlert, showToast } = useI18nUIComponents();

const {
    action,
    transaction,
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

const MIN_PEOPLE_COUNT = 2;
const MAX_PEOPLE_COUNT = 100;

const showReceivableAccountSheet = ref<boolean>(false);

const selectedReceivableAccountName = computed<string>(() => {
    const option = receivableAccountOptions.value.find(item => item.id === receivableAccountId.value);
    return option ? option.name : tt('None');
});

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

watch(peopleCount, (newValue, oldValue) => {
    if (newValue !== oldValue && Number.isFinite(newValue) && newValue >= 2 && newValue <= 100) {
        splitEqually();
    }
});

function changePeopleCount(delta: number): void {
    const newCount = peopleCount.value + delta;

    if (newCount < MIN_PEOPLE_COUNT || newCount > MAX_PEOPLE_COUNT) {
        return;
    }

    peopleCount.value = newCount;
}

function onPopupOpen(): void {
    if (props.currentTransaction) {
        init(props.currentTransaction, props.splitAction);
    }
}

function onPopupClosed(): void {
    showReceivableAccountSheet.value = false;
    emit('update:show', false);
}

function close(): void {
    if (submitting.value) {
        return;
    }

    emit('update:show', false);
}

function confirm(): void {
    if (submitting.value || !transaction.value) {
        return;
    }

    if (inputProblemMessage.value) {
        showAlert(inputProblemMessage.value);
        return;
    }

    showLoading();

    submit().then(message => {
        hideLoading();
        emit('update:show', false);
        emit('done', message);
    }).catch(error => {
        hideLoading();

        if (!error.processed) {
            showToast(error.message || error);
        }
    });
}
</script>

<style>
.split-people-count {
    min-width: 2.5em;
    text-align: center;
    font-size: 17px;
}
</style>
