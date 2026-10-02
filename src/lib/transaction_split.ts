import { TransactionType } from '@/core/transaction.ts';

import type { TransactionInfoResponse, TransactionModifyRequest } from '@/models/transaction.ts';

export interface TransactionSplitCategoryLike {
    readonly id: string;
    readonly name?: string;
    readonly type: number;
    readonly hidden: boolean;
    readonly subCategories?: TransactionSplitCategoryLike[];
}

export const TRANSACTION_SPLIT_MAX_NEW_ITEMS_COUNT: number = 20;
export const TRANSACTION_SPLIT_MAX_COMMENT_LENGTH: number = 255;
export const TRANSACTION_SPLIT_DEFAULT_RECEIVABLE_ACCOUNT_NAME: string = 'Due back';
export const TRANSACTION_SPLIT_DEFAULT_TRANSFER_CATEGORY_NAME: string = 'Due back';

export enum TransactionSplitAction {
    SplitByCategories = 'splitByCategories',
    SplitWithPeople = 'splitWithPeople',
    MarkAsRepayment = 'markAsRepayment',
    MarkAsRefund = 'markAsRefund'
}

export interface TransactionSplitExpenseItemRequest {
    readonly categoryId: string;
    readonly amount: number;
    readonly comment?: string;
}

export interface TransactionSplitTransferItemRequest {
    readonly categoryId: string;
    readonly destinationAccountId: string;
    readonly amount: number;
    readonly destinationAmount?: number;
    readonly comment?: string;
}

export interface TransactionSplitRequest {
    readonly id: string;
    readonly categoryId: string;
    readonly amount: number;
    readonly comment?: string;
    readonly expenseItems?: TransactionSplitExpenseItemRequest[];
    readonly transferItems?: TransactionSplitTransferItemRequest[];
}

export interface TransactionSplitResponse {
    readonly transaction: TransactionInfoResponse;
    readonly newTransactions: TransactionInfoResponse[];
}

export interface TransactionSplitCategoryItem {
    categoryId: string;
    amount: number;
}

// Returns the total amount the other people owe when the amount is split equally between all people (including the payer).
// The share of each other person is rounded down to a whole currency unit, because people pay back round amounts,
// so the payer takes the fraction and nothing is left owed after everyone has paid back.
export function getEqualSplitOthersAmount(totalAmount: number, peopleCount: number, minorUnitsPerUnit: number = 100): number {
    if (!Number.isFinite(totalAmount) || !Number.isFinite(peopleCount) || totalAmount <= 0 || peopleCount < 2) {
        return 0;
    }

    peopleCount = Math.floor(peopleCount);

    const exactShare = Math.floor(totalAmount / peopleCount);
    let share = exactShare;

    if (minorUnitsPerUnit > 1 && exactShare >= minorUnitsPerUnit) {
        share = Math.floor(exactShare / minorUnitsPerUnit) * minorUnitsPerUnit;
    }

    return share * (peopleCount - 1);
}

export function getTransactionSplitRemainingAmount(totalAmount: number, items: TransactionSplitCategoryItem[]): number {
    let remainingAmount = totalAmount;

    for (const item of items) {
        remainingAmount -= item.amount || 0;
    }

    return remainingAmount;
}

// Returns the first visible secondary category of the specified type. A secondary category whose own type differs from
// the type of its primary category (e.g. an expense category filed under a transfer category) is skipped, because the
// server rejects it for this type of transaction.
export function getFirstVisibleSubCategoryIdOfType(categories: TransactionSplitCategoryLike[] | undefined, categoryType: number, preferredName?: string): string {
    if (!categories) {
        return '';
    }

    if (preferredName) {
        for (const primaryCategory of categories) {
            if (primaryCategory.hidden || primaryCategory.type !== categoryType || !primaryCategory.subCategories) {
                continue;
            }

            for (const secondaryCategory of primaryCategory.subCategories) {
                if (!secondaryCategory.hidden && secondaryCategory.type === categoryType && secondaryCategory.name === preferredName) {
                    return secondaryCategory.id;
                }
            }
        }
    }

    for (const primaryCategory of categories) {
        if (primaryCategory.hidden || primaryCategory.type !== categoryType || !primaryCategory.subCategories) {
            continue;
        }

        for (const secondaryCategory of primaryCategory.subCategories) {
            if (!secondaryCategory.hidden && secondaryCategory.type === categoryType) {
                return secondaryCategory.id;
            }
        }
    }

    return '';
}

// Puts a short note in front of the comment and cuts the end of the comment to fit the maximum comment length
export function prependTransactionComment(prefix: string, comment: string, maxLength: number = TRANSACTION_SPLIT_MAX_COMMENT_LENGTH): string {
    const finalComment = comment ? `${prefix} ${comment}` : prefix;
    const characters = Array.from(finalComment);

    if (characters.length <= maxLength) {
        return finalComment;
    }

    return characters.slice(0, maxLength).join('').trimEnd();
}

// An income transaction which is actually money paid back by others becomes a transfer from the receivables account to the account which received the money
export function buildMarkAsRepaymentRequest(request: TransactionModifyRequest, receivableAccountId: string, transferCategoryId: string): TransactionModifyRequest {
    return {
        ...request,
        type: TransactionType.Transfer,
        categoryId: transferCategoryId,
        sourceAccountId: receivableAccountId,
        destinationAccountId: request.sourceAccountId,
        sourceAmount: request.sourceAmount,
        destinationAmount: request.sourceAmount
    };
}

// An income transaction which is actually a refund becomes a negative expense, so it reduces the total amount of the expense category
export function buildMarkAsRefundRequest(request: TransactionModifyRequest, expenseCategoryId: string): TransactionModifyRequest {
    return {
        ...request,
        type: TransactionType.Expense,
        categoryId: expenseCategoryId,
        sourceAmount: -request.sourceAmount,
        destinationAccountId: '0',
        destinationAmount: 0
    };
}
