import { ref, computed } from 'vue';

import axios from 'axios';

import { useI18n } from '@/locales/helpers.ts';

import { useUserStore } from '@/stores/user.ts';
import { useAccountsStore } from '@/stores/account.ts';
import { useTransactionCategoriesStore } from '@/stores/transactionCategory.ts';
import { useTransactionsStore } from '@/stores/transaction.ts';
import { useOverviewStore } from '@/stores/overview.ts';
import { useStatisticsStore } from '@/stores/statistics.ts';
import { useExplorersStore } from '@/stores/explorer.ts';

import type { ApiResponse } from '@/core/api.ts';
import { AccountCategory, AccountType } from '@/core/account.ts';
import { CategoryType } from '@/core/category.ts';
import { TransactionType } from '@/core/transaction.ts';

import { Account } from '@/models/account.ts';
import type { TransactionCategory } from '@/models/transaction_category.ts';
import type { Transaction, TransactionModifyRequest } from '@/models/transaction.ts';

import { parseBigDecimal } from '@/lib/numeral.ts';
import { getCurrentUnixTime } from '@/lib/datetime.ts';
import { generateRandomUUID } from '@/lib/misc.ts';
import {
    TRANSACTION_SPLIT_MAX_NEW_ITEMS_COUNT,
    TRANSACTION_SPLIT_DEFAULT_RECEIVABLE_ACCOUNT_NAME,
    TRANSACTION_SPLIT_DEFAULT_TRANSFER_CATEGORY_NAME,
    TransactionSplitAction,
    type TransactionSplitRequest,
    type TransactionSplitResponse,
    type TransactionSplitCategoryItem,
    getEqualSplitOthersAmount,
    getFirstVisibleSubCategoryIdOfType,
    getTransactionSplitRemainingAmount,
    prependTransactionComment,
    buildMarkAsRepaymentRequest,
    buildMarkAsRefundRequest
} from '@/lib/transaction_split.ts';
import services from '@/lib/services.ts';
import logger from '@/lib/logger.ts';

export interface TransactionSplitActionInfo {
    readonly action: TransactionSplitAction;
    readonly name: string;
}

export const EXPENSE_TRANSACTION_SPLIT_ACTIONS: TransactionSplitActionInfo[] = [
    { action: TransactionSplitAction.SplitByCategories, name: 'Split by Category' },
    { action: TransactionSplitAction.SplitWithPeople, name: 'Split with People' }
];

export const INCOME_TRANSACTION_SPLIT_ACTIONS: TransactionSplitActionInfo[] = [
    { action: TransactionSplitAction.MarkAsRepayment, name: 'Mark as Repayment' },
    { action: TransactionSplitAction.MarkAsRefund, name: 'Mark as Refund' }
];

export function getAvailableTransactionSplitActions(transaction?: { type: number, sourceAmount: number } | null): TransactionSplitActionInfo[] {
    if (!transaction || transaction.sourceAmount <= 0) {
        return [];
    }

    if (transaction.type === TransactionType.Expense) {
        return EXPENSE_TRANSACTION_SPLIT_ACTIONS;
    } else if (transaction.type === TransactionType.Income) {
        return INCOME_TRANSACTION_SPLIT_ACTIONS;
    } else {
        return [];
    }
}

export function useTransactionSplitDialogBase() {
    const { tt, formatAmountToLocalizedNumeralsWithCurrency } = useI18n();

    const userStore = useUserStore();
    const accountsStore = useAccountsStore();
    const transactionCategoriesStore = useTransactionCategoriesStore();
    const transactionsStore = useTransactionsStore();
    const overviewStore = useOverviewStore();
    const statisticsStore = useStatisticsStore();
    const explorersStore = useExplorersStore();

    const action = ref<TransactionSplitAction>(TransactionSplitAction.SplitByCategories);
    const transaction = ref<Transaction | null>(null);
    const submitting = ref<boolean>(false);

    const remainingCategoryId = ref<string>('');
    const categoryItems = ref<TransactionSplitCategoryItem[]>([]);
    const peopleCount = ref<number>(2);
    const othersAmount = ref<number>(0);
    const note = ref<string>('');
    const receivableAccountId = ref<string>('');
    const transferCategoryId = ref<string>('');
    const refundCategoryId = ref<string>('');

    const title = computed<string>(() => {
        switch (action.value) {
            case TransactionSplitAction.SplitByCategories:
                return 'Split by Category';
            case TransactionSplitAction.SplitWithPeople:
                return 'Split with People';
            case TransactionSplitAction.MarkAsRepayment:
                return 'Mark as Repayment';
            case TransactionSplitAction.MarkAsRefund:
                return 'Mark as Refund';
            default:
                return '';
        }
    });

    const allCategories = computed<Record<number, TransactionCategory[]>>(() => transactionCategoriesStore.allTransactionCategories);
    const hasVisibleExpenseCategories = computed<boolean>(() => !!getFirstVisibleSubCategoryIdOfType(allCategories.value[CategoryType.Expense], CategoryType.Expense));
    const hasVisibleTransferCategories = computed<boolean>(() => !!getFirstVisibleSubCategoryIdOfType(allCategories.value[CategoryType.Transfer], CategoryType.Transfer));

    const totalAmount = computed<number>(() => transaction.value?.sourceAmount ?? 0);

    const currency = computed<string>(() => {
        const account = transaction.value ? accountsStore.allAccountsMap[transaction.value.sourceAccountId] : undefined;
        return account?.currency || userStore.currentUserDefaultCurrency;
    });

    const receivableAccounts = computed<Account[]>(() => {
        const accounts: Account[] = [];

        for (const account of accountsStore.allAccounts) {
            if (account.category === AccountCategory.Receivables.type
                && account.type === AccountType.SingleAccount.type
                && account.visible
                && account.currency === currency.value
                && account.id !== transaction.value?.sourceAccountId) {
                accounts.push(account);
            }
        }

        return accounts;
    });

    const receivableAccountOptions = computed<{ id: string, name: string }[]>(() => {
        return receivableAccounts.value.map(account => ({
            id: account.id,
            name: `${account.name} (${formatAmountToLocalizedNumeralsWithCurrency(parseBigDecimal(account.numericBalance), account.currency)})`
        }));
    });

    const selectedReceivableAccount = computed<Account | undefined>(() => receivableAccounts.value.find(account => account.id === receivableAccountId.value));

    const remainingAmount = computed<number>(() => {
        if (action.value === TransactionSplitAction.SplitByCategories) {
            return getTransactionSplitRemainingAmount(totalAmount.value, categoryItems.value);
        } else if (action.value === TransactionSplitAction.SplitWithPeople) {
            return totalAmount.value - (othersAmount.value || 0);
        } else {
            return totalAmount.value;
        }
    });

    const canAddCategoryItem = computed<boolean>(() => categoryItems.value.length < TRANSACTION_SPLIT_MAX_NEW_ITEMS_COUNT);

    const isRepaymentMoreThanOwed = computed<boolean>(() => {
        return action.value === TransactionSplitAction.MarkAsRepayment
            && !!selectedReceivableAccount.value
            && totalAmount.value > selectedReceivableAccount.value.numericBalance;
    });

    const inputProblemMessage = computed<string | null>(() => {
        if (!transaction.value || totalAmount.value <= 0) {
            return 'An error occurred';
        }

        if (action.value === TransactionSplitAction.SplitByCategories) {
            if (!remainingCategoryId.value) {
                return 'Please select a category for each part';
            }

            for (const item of categoryItems.value) {
                if (!item.categoryId) {
                    return 'Please select a category for each part';
                }

                if (!item.amount || item.amount <= 0) {
                    return 'The amount of each part must be greater than zero';
                }
            }

            if (categoryItems.value.length < 1 || remainingAmount.value <= 0) {
                return 'The total amount of the other parts must be less than the transaction amount';
            }
        } else if (action.value === TransactionSplitAction.SplitWithPeople) {
            if (!remainingCategoryId.value) {
                return 'Please select a category';
            }

            if (!othersAmount.value || othersAmount.value <= 0 || othersAmount.value > totalAmount.value) {
                return 'The amount others owe must be greater than zero and cannot exceed the transaction amount';
            }

            if (!transferCategoryId.value) {
                return 'No secondary transfer categories are available';
            }
        } else if (action.value === TransactionSplitAction.MarkAsRepayment) {
            if (!transferCategoryId.value) {
                return 'No secondary transfer categories are available';
            }
        } else if (action.value === TransactionSplitAction.MarkAsRefund) {
            if (!refundCategoryId.value) {
                return 'Please select a category';
            }
        }

        return null;
    });

    function formatAmount(amount: number): string {
        return formatAmountToLocalizedNumeralsWithCurrency(parseBigDecimal(amount), currency.value);
    }

    function init(newTransaction: Transaction, newAction: TransactionSplitAction): void {
        transaction.value = newTransaction;
        action.value = newAction;
        submitting.value = false;

        remainingCategoryId.value = newTransaction.type === TransactionType.Expense ? newTransaction.expenseCategoryId : '';
        categoryItems.value = [{ categoryId: '', amount: 0 }];
        peopleCount.value = 2;
        othersAmount.value = getEqualSplitOthersAmount(newTransaction.sourceAmount, 2);
        note.value = '';
        receivableAccountId.value = receivableAccounts.value[0]?.id ?? '';
        transferCategoryId.value = getFirstVisibleSubCategoryIdOfType(allCategories.value[CategoryType.Transfer], CategoryType.Transfer, TRANSACTION_SPLIT_DEFAULT_TRANSFER_CATEGORY_NAME);
        refundCategoryId.value = '';
    }

    function addCategoryItem(): void {
        if (!canAddCategoryItem.value) {
            return;
        }

        categoryItems.value.push({
            categoryId: '',
            amount: 0
        });
    }

    function removeCategoryItem(index: number): void {
        if (categoryItems.value.length <= 1) {
            return;
        }

        categoryItems.value.splice(index, 1);
    }

    function splitEqually(): void {
        othersAmount.value = getEqualSplitOthersAmount(totalAmount.value, peopleCount.value);
    }

    function invalidateStores(): void {
        transactionsStore.updateTransactionListInvalidState(true);
        transactionsStore.updateTransactionReconciliationStatementInvalidState(true);
        accountsStore.updateAccountListInvalidState(true);
        overviewStore.updateTransactionOverviewInvalidState(true);
        statisticsStore.updateTransactionStatisticsInvalidState(true);
        explorersStore.updateTransactionExplorerInvalidState(true);
    }

    function getReceivableAccountId(): Promise<string> {
        if (selectedReceivableAccount.value) {
            return Promise.resolve(selectedReceivableAccount.value.id);
        }

        const account = Account.createNewAccount(AccountCategory.Receivables, currency.value, getCurrentUnixTime());
        account.name = tt(TRANSACTION_SPLIT_DEFAULT_RECEIVABLE_ACCOUNT_NAME);

        return accountsStore.saveAccount({
            account: account,
            subAccounts: [],
            isEdit: false,
            clientSessionId: generateRandomUUID()
        }).then(newAccount => {
            receivableAccountId.value = newAccount.id;
            return newAccount.id;
        });
    }

    function buildSplitRequest(currentTransaction: Transaction, destinationAccountId: string): TransactionSplitRequest {
        const total = formatAmount(totalAmount.value);
        const comment = prependTransactionComment(`[${tt('Split from')} ${total}]`, currentTransaction.comment);
        const partComment = prependTransactionComment(`[${tt('Part of')} ${total}]`, currentTransaction.comment);

        if (action.value === TransactionSplitAction.SplitWithPeople) {
            const owedBy = note.value.trim();
            const owedComment = prependTransactionComment(owedBy ? `[${tt('Owed by')} ${owedBy}, ${tt('Part of')} ${total}]` : `[${tt('Owed')}, ${tt('Part of')} ${total}]`, currentTransaction.comment);

            return {
                id: currentTransaction.id,
                categoryId: remainingCategoryId.value,
                amount: remainingAmount.value,
                comment: comment,
                transferItems: [{
                    categoryId: transferCategoryId.value,
                    destinationAccountId: destinationAccountId,
                    amount: othersAmount.value,
                    comment: owedComment
                }]
            };
        }

        return {
            id: currentTransaction.id,
            categoryId: remainingCategoryId.value,
            amount: remainingAmount.value,
            comment: comment,
            expenseItems: categoryItems.value.map(item => ({
                categoryId: item.categoryId,
                amount: item.amount,
                comment: partComment
            }))
        };
    }

    function doSplit(currentTransaction: Transaction, destinationAccountId: string): Promise<void> {
        return axios.post<ApiResponse<TransactionSplitResponse>>('v1/transactions/split.json', buildSplitRequest(currentTransaction, destinationAccountId)).then(response => {
            const data = response.data;

            if (!data || !data.success || !data.result) {
                return Promise.reject({ message: 'Unable to split transaction' });
            }

            return Promise.resolve();
        });
    }

    function doModify(request: TransactionModifyRequest): Promise<void> {
        return services.modifyTransaction(request).then(response => {
            const data = response.data;

            if (!data || !data.success || !data.result) {
                return Promise.reject({ message: 'Unable to save transaction' });
            }

            return Promise.resolve();
        });
    }

    function submit(): Promise<string> {
        const currentTransaction = transaction.value;
        const currentAction = action.value;

        if (!currentTransaction || inputProblemMessage.value) {
            return Promise.reject({ message: inputProblemMessage.value || 'An error occurred' });
        }

        let promise: Promise<void>;
        let successMessage: string;
        let failedMessage: string;

        if (currentAction === TransactionSplitAction.SplitByCategories) {
            promise = doSplit(currentTransaction, '');
            successMessage = 'Transaction has been split';
            failedMessage = 'Unable to split transaction';
        } else if (currentAction === TransactionSplitAction.SplitWithPeople) {
            promise = getReceivableAccountId().then(accountId => doSplit(currentTransaction, accountId));
            successMessage = 'Transaction has been split';
            failedMessage = 'Unable to split transaction';
        } else if (currentAction === TransactionSplitAction.MarkAsRepayment) {
            promise = getReceivableAccountId().then(accountId => doModify(buildMarkAsRepaymentRequest(currentTransaction.toModifyRequest(), accountId, transferCategoryId.value)));
            successMessage = 'Transaction has been marked as repayment';
            failedMessage = 'Unable to save transaction';
        } else {
            promise = doModify(buildMarkAsRefundRequest(currentTransaction.toModifyRequest(), refundCategoryId.value));
            successMessage = 'Transaction has been marked as refund';
            failedMessage = 'Unable to save transaction';
        }

        submitting.value = true;

        return promise.then(() => {
            submitting.value = false;
            invalidateStores();
            return successMessage;
        }).catch(error => {
            submitting.value = false;
            logger.error('failed to split or mark transaction', error);

            if (error && error.response && error.response.data && error.response.data.errorMessage) {
                return Promise.reject({ error: error.response.data });
            } else if (error && (error.processed || error.message || error.error)) {
                return Promise.reject(error);
            } else {
                return Promise.reject({ message: failedMessage });
            }
        });
    }

    return {
        // states
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
        // computed states
        title,
        allCategories,
        hasVisibleExpenseCategories,
        hasVisibleTransferCategories,
        totalAmount,
        currency,
        receivableAccounts,
        receivableAccountOptions,
        selectedReceivableAccount,
        remainingAmount,
        canAddCategoryItem,
        isRepaymentMoreThanOwed,
        inputProblemMessage,
        // functions
        formatAmount,
        init,
        addCategoryItem,
        removeCategoryItem,
        splitEqually,
        submit
    };
}
