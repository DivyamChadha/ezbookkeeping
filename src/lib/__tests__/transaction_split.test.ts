import { describe, expect, it } from 'vitest';

import { TransactionType } from '@/core/transaction.ts';
import type { TransactionModifyRequest } from '@/models/transaction.ts';

import {
    getEqualSplitOthersAmount,
    getTransactionSplitRemainingAmount,
    getFirstVisibleSubCategoryIdOfType,
    prependTransactionComment,
    buildMarkAsRepaymentRequest,
    buildMarkAsRefundRequest
} from '@/lib/transaction_split.ts';

const incomeRequest: TransactionModifyRequest = {
    id: '1',
    type: TransactionType.Income,
    categoryId: '100',
    time: 1790000000,
    utcOffset: 330,
    sourceAccountId: '10',
    destinationAccountId: '0',
    sourceAmount: 36100,
    destinationAmount: 0,
    hideAmount: false,
    tagIds: ['5'],
    pictureIds: [],
    comment: 'credited from friend'
};

describe('getEqualSplitOthersAmount', () => {
    it('should round the share of each other person down to a whole currency unit', () => {
        expect(getEqualSplitOthersAmount(108378, 3)).toBe(72200);
        expect(getEqualSplitOthersAmount(100000, 2)).toBe(50000);
        expect(getEqualSplitOthersAmount(100000, 3)).toBe(66600);
        expect(getEqualSplitOthersAmount(99999, 4)).toBe(74700);
    });

    it('should not round when the share is smaller than one currency unit', () => {
        expect(getEqualSplitOthersAmount(150, 3)).toBe(100);
        expect(getEqualSplitOthersAmount(100, 3)).toBe(66);
    });

    it('should not round for currencies without minor units', () => {
        expect(getEqualSplitOthersAmount(1000, 3, 1)).toBe(666);
    });

    it('should never exceed the total amount', () => {
        for (let peopleCount = 2; peopleCount <= 20; peopleCount++) {
            expect(getEqualSplitOthersAmount(108378, peopleCount)).toBeLessThan(108378);
            expect(getEqualSplitOthersAmount(7, peopleCount)).toBeLessThanOrEqual(7);
        }
    });

    it('should return zero for invalid arguments', () => {
        expect(getEqualSplitOthersAmount(0, 3)).toBe(0);
        expect(getEqualSplitOthersAmount(-100, 3)).toBe(0);
        expect(getEqualSplitOthersAmount(100000, 1)).toBe(0);
        expect(getEqualSplitOthersAmount(100000, 0)).toBe(0);
        expect(getEqualSplitOthersAmount(100000, NaN)).toBe(0);
    });
});

describe('getTransactionSplitRemainingAmount', () => {
    it('should subtract all item amounts from the total amount', () => {
        expect(getTransactionSplitRemainingAmount(87100, [])).toBe(87100);
        expect(getTransactionSplitRemainingAmount(87100, [{ categoryId: '1', amount: 27100 }])).toBe(60000);
        expect(getTransactionSplitRemainingAmount(87100, [{ categoryId: '1', amount: 27100 }, { categoryId: '2', amount: 70000 }])).toBe(-10000);
    });
});

describe('prependTransactionComment', () => {
    it('should put the prefix in front of the comment', () => {
        expect(prependTransactionComment('[Split from 871.00]', 'Spent Rs.871 At Swiggy')).toBe('[Split from 871.00] Spent Rs.871 At Swiggy');
    });

    it('should return only the prefix when the comment is empty', () => {
        expect(prependTransactionComment('[Split from 871.00]', '')).toBe('[Split from 871.00]');
    });

    it('should cut the end of the comment to fit the maximum length', () => {
        const comment = prependTransactionComment('[Split from 871.00]', 'x'.repeat(255));

        expect(Array.from(comment).length).toBe(255);
        expect(comment.startsWith('[Split from 871.00] xxx')).toBe(true);
    });

    it('should count characters instead of UTF-16 code units', () => {
        const comment = prependTransactionComment('[₹]', '😀'.repeat(300));

        expect(Array.from(comment).length).toBe(255);
    });
});

describe('buildMarkAsRepaymentRequest', () => {
    it('should turn the income into a transfer from the receivables account to the original account', () => {
        const request = buildMarkAsRepaymentRequest(incomeRequest, '20', '300');

        expect(request.id).toBe('1');
        expect(request.type).toBe(TransactionType.Transfer);
        expect(request.categoryId).toBe('300');
        expect(request.sourceAccountId).toBe('20');
        expect(request.destinationAccountId).toBe('10');
        expect(request.sourceAmount).toBe(36100);
        expect(request.destinationAmount).toBe(36100);
        expect(request.time).toBe(1790000000);
        expect(request.tagIds).toEqual(['5']);
        expect(request.comment).toBe('credited from friend');
    });
});

describe('buildMarkAsRefundRequest', () => {
    it('should turn the income into a negative expense in the same account', () => {
        const request = buildMarkAsRefundRequest(incomeRequest, '200');

        expect(request.id).toBe('1');
        expect(request.type).toBe(TransactionType.Expense);
        expect(request.categoryId).toBe('200');
        expect(request.sourceAccountId).toBe('10');
        expect(request.destinationAccountId).toBe('0');
        expect(request.sourceAmount).toBe(-36100);
        expect(request.destinationAmount).toBe(0);
    });
});

describe('getFirstVisibleSubCategoryIdOfType', () => {
    const transferCategories = [
        { id: '10', type: 3, hidden: true, subCategories: [{ id: '11', type: 3, hidden: false }] },
        {
            id: '20', type: 3, hidden: false, subCategories: [
                { id: '21', type: 2, hidden: false }, // an expense category filed under a transfer category
                { id: '22', type: 3, hidden: true },
                { id: '23', type: 3, hidden: false }
            ]
        }
    ];

    it('should skip hidden categories and secondary categories of another type', () => {
        expect(getFirstVisibleSubCategoryIdOfType(transferCategories, 3)).toBe('23');
    });

    it('should return empty when there is no matched category', () => {
        expect(getFirstVisibleSubCategoryIdOfType(transferCategories, 2)).toBe('');
        expect(getFirstVisibleSubCategoryIdOfType([], 3)).toBe('');
        expect(getFirstVisibleSubCategoryIdOfType(undefined, 3)).toBe('');
    });
});

describe('getFirstVisibleSubCategoryIdOfType with a preferred name', () => {
    const categories = [{
        id: 'p', type: 3, hidden: false, subCategories: [
            { id: 'a', name: 'Uncategorized', type: 3, hidden: false },
            { id: 'b', name: 'Due back', type: 3, hidden: false }
        ]
    }];

    it('picks the category with the preferred name', () => {
        expect(getFirstVisibleSubCategoryIdOfType(categories, 3, 'Due back')).toBe('b');
    });

    it('falls back to the first category when the name is missing', () => {
        expect(getFirstVisibleSubCategoryIdOfType(categories, 3, 'Nope')).toBe('a');
    });
});
